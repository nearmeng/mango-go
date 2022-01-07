// Copyright 2020 The TpsTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metric metric 子系统
package metric

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hanjm/etcd/clientv3" // why not go.etcd.io/etcd/clientv3
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	// DefaultRegisterTTL default register ttl
	DefaultRegisterTTL = time.Second * 60
	// DefaultDialTimeout default dail timeout
	DefaultDialTimeout = time.Second * 5
)

// Registry register or unregister instance to registry
type Registry interface {
	// Register register instance to registry
	Register(ctx context.Context, ins *Instance, ttl time.Duration) (context.CancelFunc, error)
}

// etcdRegistry register or unregister instance to etcd
type etcdRegistry struct {
	cli         *clientv3.Client
	tenantID    string
	registerTTL time.Duration
	err         error
}

// NewEtcdRegistry new etcd registry
func NewEtcdRegistry(etcdEndpoints []string) Registry {
	r := &etcdRegistry{
		registerTTL: DefaultRegisterTTL,
	}
	r.cli, r.err = clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: DefaultDialTimeout,
		DialOptions: []grpc.DialOption{grpc.WithBlock(), // 这样才能让DialTimeout生效
			grpc.WithChainStreamInterceptor(r.streamInterceptor()),
			grpc.WithUnaryInterceptor(r.unaryInterceptor()),
		},
	})
	return r
}

// Register register instance to etcd
func (e *etcdRegistry) Register(ctx context.Context, ins *Instance, ttl time.Duration) (context.CancelFunc, error) {
	if err := e.err; err != nil {
		return nil, err
	}
	e.tenantID = ins.TenantID
	cctx, cancel := context.WithCancel(ctx)
	leaseID, err := e.register(cctx, ins, ttl)
	if err != nil {
		cancel()
		return nil, err
	}
	ch := make(chan struct{}, 1)
	cancelFunc := func() {
		cancel()
		<-ch
	}
	go func() {
		leaseID := leaseID
		for {
			err := e.keepAlive(cctx, leaseID)
			select {
			case <-cctx.Done():
				_ = e.unregister(context.Background(), ins)
				ch <- struct{}{}
				return
			default:
			}
			if err != nil {
				retryWait := e.registerTTL/3 + time.Duration(rand.Int63n(int64(e.registerTTL/3)))
				log.Printf("[E]tpstelemetry: keepAlive error:%v, will retry after %v", err, retryWait)
				time.Sleep(retryWait)
				leaseID, _ = e.register(cctx, ins, ttl)
			}
		}
	}()
	return cancelFunc, nil
}

func (e *etcdRegistry) register(
	ctx context.Context, ins *Instance, ttl time.Duration) (clientv3.LeaseID, error) {
	if err := e.err; err != nil {
		return 0, err
	}
	ttlResp, err := e.cli.Grant(ctx, int64(ttl.Seconds()))
	if err != nil {
		return 0, err
	}
	_, err = e.cli.Put(ctx, toKey(ins), toValue(ins), clientv3.WithLease(ttlResp.ID))
	if err != nil {
		return 0, err
	}
	return ttlResp.ID, nil
}

// keepAlive 阻塞直到context canceled或者底层连接长时间断开超过TTL, ch会被close, 调用方需要重新发起lease
func (e *etcdRegistry) keepAlive(
	ctx context.Context, id clientv3.LeaseID) error {
	ch, err := e.cli.KeepAlive(ctx, id)
	if err != nil {
		return fmt.Errorf("keepAlive err:%w", err)
	}
	for range ch {
	} // noop
	return fmt.Errorf("keepAlive ch closed")
}

func (e *etcdRegistry) unregister(ctx context.Context, ins *Instance) error {
	if err := e.err; err != nil {
		return err
	}
	if _, err := e.cli.Delete(ctx, toKey(ins)); err != nil {
		return err
	}
	return nil
}

func (*etcdRegistry) contextWithTenantID(ctx context.Context, tenantID string) context.Context {
	const tenantHeaderKey = "x-tps-tenantid" // grpc的header为小写
	return metadata.AppendToOutgoingContext(ctx, tenantHeaderKey, tenantID)
}

func (e *etcdRegistry) unaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(e.contextWithTenantID(ctx, e.tenantID), method, req, reply, cc, opts...)
	}
}

func (e *etcdRegistry) streamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer,
		opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(e.contextWithTenantID(ctx, e.tenantID), desc, cc, method, opts...)
	}
}
