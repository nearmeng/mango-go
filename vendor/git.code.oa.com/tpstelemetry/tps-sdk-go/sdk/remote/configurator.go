// Copyright 2021 The TpsTelemetry Authors
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

// Package remote 处理天机阁远端控制相关逻辑
package remote

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"git.code.oa.com/tpstelemetry/tpstelemetry-protocol/tpstelemetry/proto/operation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ConfigApplyFunc 配置应用函数.
type ConfigApplyFunc func(config *operation.Operation) error

// Configurator 配置器接口, 可注册配置变更处理函数, 配置有变化时调用它.
type Configurator interface {
	RegisterConfigApplyFunc(fn ConfigApplyFunc)
}

// remoteConfigurator 天机阁远程配置器
type remoteConfigurator struct {
	remoteServiceAddr string
	syncInterval      time.Duration
	tenantID          string
	app               string
	server            string
	debug             bool

	client              operation.OperationServiceClient
	lastConfig          *operation.Operation
	configApplyFuncList []ConfigApplyFunc
	// mu Protect lastConfig/configApplyFuncList.
	mu sync.Mutex
}

// NewRemoteConfigurator 创建远端配置器
func NewRemoteConfigurator(remoteServiceAddr string, syncInterval time.Duration,
	tenantID, app, server string) Configurator {
	if syncInterval == 0 {
		syncInterval = time.Minute
	}
	rc := &remoteConfigurator{
		remoteServiceAddr: remoteServiceAddr,
		syncInterval:      syncInterval,
		tenantID:          tenantID,
		app:               app,
		server:            server,
	}
	// export TPS_TRACE=remote
	if tpsTraceEnv := os.Getenv("TPS_TRACE"); strings.Contains(tpsTraceEnv, "remote") {
		log.Printf("tpstelemetry: env TPS_TRACE:%s", tpsTraceEnv)
		rc.debug = true
	}
	if rc.remoteServiceAddr != "" {
		go rc.syncDaemon()
	}
	return rc
}

func (rc *remoteConfigurator) syncDaemon() {
	for {
		rc.sync()
		time.Sleep(rc.syncInterval)
	}
}

func (rc *remoteConfigurator) sync() {
	if rc.client == nil {
		cc, err := grpc.Dial(rc.remoteServiceAddr, grpc.WithInsecure())
		if err != nil {
			if rc.debug {
				log.Printf("tpstelemetry: remote dial err:%v", err)
			}
			return
		}
		rc.client = operation.NewOperationServiceClient(cc)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"x-tps-tenantid": rc.tenantID,
	}))
	req := &operation.GetOperationRequest{
		Tenant: rc.tenantID,
		App:    rc.app,
		Server: rc.server,
	}
	rsp, err := rc.client.GetOperation(ctx, req, grpc.WaitForReady(true))
	if err != nil {
		if rc.debug {
			log.Printf("tpstelemetry: remote GetOperation err:%v", err)
		}
		return
	}
	if rc.debug {
		log.Printf("tpstelemetry: remote GetOperation result:%+v", rsp)
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.lastConfig = rsp.GetOperation()
	for _, v := range rc.configApplyFuncList {
		if err = v(rsp.GetOperation()); err != nil {
			if rc.debug {
				log.Printf("tpstelemetry: remote apply err:%v", err)
			}
		}
	}
}

// RegisterConfigApplyFunc 注册配置应用的函数, 每次配置同步后会调用它
func (rc *remoteConfigurator) RegisterConfigApplyFunc(fn ConfigApplyFunc) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.configApplyFuncList = append(rc.configApplyFuncList, fn)
	// Apply on register for async setup.
	if rc.lastConfig != nil {
		if err := fn(rc.lastConfig); err != nil && rc.debug {
			log.Printf("tpstelemetry: remote apply err:%v", err)
		}
	}
}
