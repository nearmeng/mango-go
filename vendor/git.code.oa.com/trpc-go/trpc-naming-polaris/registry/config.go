package registry

// Config 配置
type Config struct {
	// ServiceToken 服务访问Token
	ServiceToken string
	// Protocol 服务端访问方式，支持 http grpc，默认 grpc
	Protocol string
	// HeartBeat 上报心跳时间间隔，默认为建议 为TTL/2
	HeartBeat int
	// EnableRegister 默认只上报心跳，不注册服务，为 true 则启动注册
	EnableRegister bool
	// Weight
	Weight int
	// TTL 单位s，服务端检查周期实例是否健康的周期
	TTL int
	// InstanceID 实例名
	InstanceID string
	// Namespace 命名空间
	Namespace string
	// ServiceName 服务名
	ServiceName string
	// BindAddress 指定上报地址
	BindAddress string
	// Metadata 用户自定义 metadata 信息
	Metadata map[string]string
	// DisableHealthCheck 禁用健康检查
	DisableHealthCheck bool
}
