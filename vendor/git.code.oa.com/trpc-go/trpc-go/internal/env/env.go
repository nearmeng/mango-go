// Package env 框架内部使用的环境变量。
package env

// 统一定义框架所有用到的环境变量的key
const (
	// TRPC_LOG_TRACE=1 开启trace输出
	// 环境变量控制trace级别日志输出
	// 由于框架日志使用的是zap开源库，而zap没有trace级别，所以通过环境变量来控制是否输出trace日志
	LogTrace = "TRPC_LOG_TRACE"
)
