package selector

import (
	"time"

	"git.code.oa.com/polaris/polaris-go/pkg/config"
)

// Config selector配置结构
type Config struct {
	// RefreshInterval 列表刷新时间单位毫秒
	RefreshInterval int
	// ServerAddrs 服务地址
	ServerAddrs []string
	// Protocol 协议类型
	Protocol string
	// Enable ServiceRouter
	Enable bool
	// Timeout 获取信息北极星后台超时，单位ms
	Timeout int
	// ConnectTimeout 连接北极星后台超时，单位ms
	ConnectTimeout int
	// Enable 金丝雀
	EnableCanary bool
	// UseBuildin 使用sdk默认的埋点地址
	UseBuildin bool
	// ReportTimeout 如果设置了，则下游超时，并且少于设置的值，则忽略错误不上报
	ReportTimeout *time.Duration
	// EnableTransMeta 开启设置，则将'selector-meta-'前缀的透传字段摘除前缀后，填入SourceService的metaData
	EnableTransMeta bool
	// 设置本地cache存储地址
	LocalCachePersistDir string
}

const (
	setEnableKey       string = "internal-enable-set"
	setNameKey         string = "internal-set-name"
	setEnableValue     string = "Y"
	containerKey       string = "container_name"
	selectorMetaPrefix string = "selector-meta-"

	LoadBalanceWR        = "polaris_wr"        // 默认负载均衡器,权重随机
	LoadBalanceHash      = "polaris_hash"      // 负载均衡器,普通hash
	LoadBalancerRingHash = "polaris_ring_hash" // 负载均衡器,一致性hash环
	LoadBalanceMaglev    = "polaris_maglev"    // 负载均衡器,maglev hash
	LoadBalanceL5Cst     = "polaris_l5cst"     // 负载均衡器,l5一致性hash兼容
)

var loadBalanceMap map[string]string = map[string]string{
	LoadBalanceWR:        config.DefaultLoadBalancerWR,
	LoadBalanceHash:      config.DefaultLoadBalancerHash,
	LoadBalancerRingHash: config.DefaultLoadBalancerRingHash,
	LoadBalanceMaglev:    config.DefaultLoadBalancerMaglev,
	LoadBalanceL5Cst:     config.DefaultLoadBalancerL5CST,
}
