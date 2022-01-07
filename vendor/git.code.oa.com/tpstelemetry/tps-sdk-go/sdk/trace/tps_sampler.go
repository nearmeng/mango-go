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

// Package trace trace 组件
package trace

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"git.code.oa.com/tpstelemetry/tpstelemetry-protocol/tpstelemetry/proto/sampler"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ sdktrace.Sampler = &Sampler{}

// DefaultSampler 默认采样器
func DefaultSampler() sdktrace.Sampler {
	return NewSampler("", 0, "", 0.001)
}

// SamplerOptions 设置选项.
type SamplerOptions struct {
	// DefaultSamplingDecision 默认的采样Decision
	DefaultSamplingDecision sdktrace.SamplingDecision
}

// defaultSamplerOptions 默认的选项.
var defaultSamplerOptions = SamplerOptions{DefaultSamplingDecision: sdktrace.Drop}

// SamplerOption 选项应用函数.
type SamplerOption func(*SamplerOptions)

// Sampler 采样器.
type Sampler struct {
	traceIDUpperBound  uint64
	description        string
	samplerServiceAddr string
	syncInterval       time.Duration
	tenantID           string
	client             sampler.SamplerServiceClient
	sampledKvs         atomic.Value // map[string]map[string]bool
	debug              bool
	opt                SamplerOptions
}

// NewSampler 创建一个采样器.
func NewSampler(
	samplerServiceAddr string,
	syncInterval time.Duration,
	tpsTenantID string,
	fraction float64,
	opts ...SamplerOption,
) sdktrace.Sampler {
	if fraction >= 1 {
		fraction = 1
	}
	if fraction <= 0 {
		fraction = 0
	}
	if syncInterval == 0 {
		syncInterval = time.Second * 10
	}
	ws := &Sampler{
		traceIDUpperBound:  uint64(fraction * (1 << 63)),
		samplerServiceAddr: samplerServiceAddr,
		syncInterval:       syncInterval,
		tenantID:           tpsTenantID,
		description: fmt.Sprintf("TpsSampler{fraction=%g,tenantID=%s}",
			fraction, tpsTenantID),
		sampledKvs: atomic.Value{},
		opt:        defaultSamplerOptions,
	}
	for _, v := range opts {
		v(&ws.opt)
	}
	if ws.samplerServiceAddr != "" {
		ws.sampledKvs.Store(make(map[string]map[string]bool))
		go ws.updateDyeingMetadataDaemon()
	}
	if tpsTraceEnv := os.Getenv("TPS_TRACE"); strings.Contains(tpsTraceEnv, "sampler") {
		log.Printf("[tpstelemetry][I]env TPS_TRACE:%s", tpsTraceEnv)
		// export TPS_TRACE=sampler
		ws.debug = true
	}
	return ws
}

// updateDyeingMetadataDaemon
func (ws *Sampler) updateDyeingMetadataDaemon() {
	for {
		ws.updateDyeingMetadata()
		time.Sleep(ws.syncInterval)
	}
}

// updateDyeingMetadata
func (ws *Sampler) updateDyeingMetadata() {
	if ws.client == nil {
		cc, err := grpc.Dial(ws.samplerServiceAddr, grpc.WithInsecure())
		if err != nil {
			return
		}
		ws.client = sampler.NewSamplerServiceClient(cc)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"x-tps-tenantid": ws.tenantID,
	}))
	rsp, err := ws.client.GetSampler(ctx, &sampler.GetSamplerRequest{}, grpc.WaitForReady(true))
	if err != nil {
		if ws.debug {
			log.Printf("[tpstelemetry][E] Get sampler err:%v", err)
		}
		return
	}
	sampledKvs := make(map[string]map[string]bool)
	for _, v := range rsp.GetAttributes() {
		sampledKv := sampledKvs[v.Key]
		if sampledKv == nil {
			sampledKv = make(map[string]bool)
			sampledKvs[v.Key] = sampledKv
		}
		for _, vv := range v.GetValues() {
			sampledKv[vv] = true
		}
	}
	if ws.debug {
		log.Printf("[tpstelemetry][I] sampledKvs:%+v", sampledKvs)
	}
	ws.sampledKvs.Store(sampledKvs)
}

var (
	TPSForceSampler = attribute.Key("tps.force.sample")
	// 命中染色后加的标识attribute, 用于之后的延迟采样判断, key需要符合w3c格式才行, 不能带点号.
	traceStateDyeing    = attribute.Key("tps_dyeing")
	dyeingTraceState, _ = trace.TraceState{}.Insert(attribute.Bool(string(traceStateDyeing), true))
)

// ShouldSample ShouldSample
func (ws *Sampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if p.ParentContext.IsSampled() {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}
	if ws.samplerServiceAddr != "" {
		sampledKvs, ok := ws.sampledKvs.Load().(map[string]map[string]bool)
		if ok {
			for _, attr := range p.Attributes {
				key := string(attr.Key)
				if sampledKvs[key][attr.Value.AsString()] {
					return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample,
						Tracestate: dyeingTraceState}
				}
				if attr.Key == TPSForceSampler && attr.Value.AsString() != "" {
					return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample,
						Tracestate: dyeingTraceState}
				}
			}
		}
	}
	x := binary.BigEndian.Uint64(p.TraceID[0:8]) >> 1
	if x < ws.traceIDUpperBound {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}
	return sdktrace.SamplingResult{Decision: ws.opt.DefaultSamplingDecision}
}

// Description Description
func (ws *Sampler) Description() string {
	return ws.description
}
