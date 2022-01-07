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

// Package prometheus prometheus metrics
package prometheus

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	yaml "gopkg.in/yaml.v2"

	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/admin"
	"git.code.oa.com/trpc-go/trpc-go/log"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/metric"
)

// Setup
func Setup(tenantID string, etcdEndpoints []string, opts ...metric.SetupOption) {
	admin.HandleFunc("/metrics", metric.LimitMetricsHandler().ServeHTTP)
	if tenantID == "" {
		tenantID = "default"
	}
	addr, needListen := getPrometheusServerAddr()
	log.Infof("tpstelemetry: setup prometheus metrics server at http://%s/metrics", addr)
	if needListen {
		// admin端口未启动，启动http
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/metrics", metric.LimitMetricsHandler().ServeHTTP)
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Errorf("tpstelemetry: trpc_go.yaml admin service "+
					"is disable and failed to serve prometheus metrics at loopback, loopback addr:%s, err:%s", addr, err)
			} else {
				log.Errorf("tpstelemetry: trpc_go.yaml admin service "+
					"is disable, fallback to serve prometheus metrics at loopback addr:%s", addr)
			}
		}()
	}
	// 服务注册
	cfg := trpc.GlobalConfig()
	containerName := cfg.Global.ContainerName
	if containerName == "" {
		// containerName会用来在服务详情Dashboard中group by为单机曲线, 没有设置containerName时, 取addr作为containerName
		containerName = addr
	}
	instance := &metric.Instance{
		TenantID: tenantID,
		Addr:     addr,
		Metadata: map[string]string{
			"app":            cfg.Server.App,
			"server":         cfg.Server.Server,
			"namespace":      cfg.Global.Namespace,
			"env_name":       cfg.Global.EnvName,
			"container_name": containerName,
		},
	}
	go func() {
		if len(etcdEndpoints) == 0 {
			log.Warnf("tpstelemetry: trpc config " +
				"plugins=>telemetry=>tpstelemetry.metrics.registry_endpoints is empty," +
				"will not register metrics endpoint")
			return
		}
		setupOpts := []metric.SetupOption{metric.WithEnabled(true),
			metric.WithRegistryEndpoints(etcdEndpoints),
			metric.WithTTL(time.Second * 60),
			metric.WithInstance(instance)}
		setupOpts = append(setupOpts, opts...)
		err := metric.Setup(setupOpts...)
		if err != nil {
			log.Errorf("tpstelemetry: metrics endpoint register etcd err:%v, endpoints:%v", err, etcdEndpoints)
			return
		}
	}()
}

func getPrometheusServerAddr() (addr string, shouldListen bool) {
	adminConfig := trpc.GlobalConfig().Server.Admin
	if adminConfig.Port > 0 {
		return fmt.Sprintf("%s:%d", adminConfig.IP, adminConfig.Port), false
	}
	// 没有启动trpc_admin, 尝试使用配置文件里的global里的local_ip:admin_port启用admin
	if ip, port, err := getAdminAddrFromTRPCConfig(); err == nil {
		return fmt.Sprintf("%s:%d", ip, port), true
	}
	// 当没有启用admin功能且没有admin_port时, 监听本地地址
	// 当走到这里，天机阁的prom server将无法拉到这个节点的metrics，只能在本地通过curl http://127.0.0.1:12621/metrics 拉到metrics
	return "127.0.0.1:12621", true
}

func getAdminAddrFromTRPCConfig() (ip string, port uint16, err error) {
	buf, err := ioutil.ReadFile(trpc.ServerConfigPath)
	if err != nil {
		return "", 0, fmt.Errorf("read file err:%s", err)
	}
	cfg := &trpcConfig{}
	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		return "", 0, fmt.Errorf("unmarshal file err:%s", err)
	}
	if cfg.Global.LocalIP != "" && cfg.Global.AdminPort > 0 {
		return cfg.Global.LocalIP, cfg.Global.AdminPort, nil
	}
	return "", 0, fmt.Errorf("invalid ip port:%s %d", cfg.Global.LocalIP, cfg.Global.AdminPort)
}

type trpcConfig struct {
	Global struct {
		LocalIP   string `yaml:"local_ip"`
		AdminPort uint16 `yaml:"admin_port"`
	}
}
