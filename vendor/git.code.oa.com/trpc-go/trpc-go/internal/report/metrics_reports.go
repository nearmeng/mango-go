// Package report 框架内部上报数据
package report

import (
	"git.code.oa.com/trpc-go/trpc-go/metrics"
)

// 统一定义框架内部用到的所有metrics上报, 所有框架上报属性都以 trpc. 为前缀
var (
	// -----------------------------server----------------------------- //
	// service服务启动成功
	ServiceStart = metrics.Counter("trpc.ServiceStart")
	// service没有配置服务协议，框架配置必须填写protocol字段
	ServerCodecEmpty = metrics.Counter("trpc.ServerCodecEmpty")
	// 业务处理函数返回失败，一般不会出现，只有encode失败无法回包才会返回
	ServiceHandleFail = metrics.Counter("trpc.ServiceHandleFail")
	// server请求解包失败，一般是非法包
	ServiceCodecDecodeFail = metrics.Counter("trpc.ServiceCodecDecodeFail")
	// server响应打包失败，一般不会出现，有出现的话，一般是插件打解包有bug
	ServiceCodecEncodeFail = metrics.Counter("trpc.ServiceCodecEncodeFail")
	// 请求方法名错误，一般是上游调用方参数填写错误
	ServiceHandleRPCNameInvalid = metrics.Counter("trpc.ServiceHandleRpcNameInvalid")
	// server请求反序列化包体失败，一般是上下游pb协议没有对齐
	ServiceCodecUnmarshalFail = metrics.Counter("trpc.ServiceCodecUnmarshalFail")
	// server响应序列化包体失败
	ServiceCodecMarshalFail = metrics.Counter("trpc.ServiceCodecMarshalFail")
	// server请求解压缩包体失败，一般是上下游协议没有对齐，或者压缩方式没有对齐
	ServiceCodecDecompressFail = metrics.Counter("trpc.ServiceCodecDecompressFail")
	// server响应压缩包体失败
	ServiceCodecCompressFail = metrics.Counter("trpc.ServiceCodecCompressFail")

	// -----------------------------transport----------------------------- //
	// tcp收包后处理函数返回失败，一般不会出现，只有encode失败无法回包才会返回，与ServiceHandleFail一致
	TCPServerTransportHandleFail = metrics.Counter("trpc.TcpServerTransportHandleFail")
	// udp收包后处理函数返回失败，与TCPServerTransportHandleFail类似
	UDPServerTransportHandleFail = metrics.Counter("trpc.UdpServerTransportHandleFail")
	// tcp收到end of file错误，长连接下，客户端主动断开连接都是正常的，正常监控都会有大量的该属性上报
	TCPServerTransportReadEOF = metrics.Counter("trpc.TcpServerTransportReadEOF")
	// tcp写响应包失败，一般是上游客户端已经断开连接
	TCPServerTransportWriteFail = metrics.Counter("trpc.TcpServerTransportWriteFail")
	// tcp服务端接收到的请求包大小
	TCPServerTransportReceiveSize = metrics.Gauge("trpc.TcpServerTransportReceiveSize")
	// tcp 服务端接发送的响应包大小
	TCPServerTransportSendSize = metrics.Gauge("trpc.TcpServerTransportSendSize")
	// udp写响应包失败，一般是上游客户端已经超时断开，没有监听该端口
	UDPServerTransportWriteFail = metrics.Counter("trpc.UdpServerTransportWriteFail")
	// tcp长连接下超过空闲时间没有收到请求，主动释放该连接，正常
	TCPServerTransportIdleTimeout = metrics.Counter("trpc.TcpServerTransportIdleTimeout")
	// tcp readframe收包失败，一般是非法包
	TCPServerTransportReadFail = metrics.Counter("trpc.TcpServerTransportReadFail")
	// udp readframe收包失败，一般是非法包
	UDPServerTransportReadFail = metrics.Counter("trpc.UdpServerTransportReadFail")
	// udp readframe解包后还剩多余的数据
	UDPServerTransportUnRead = metrics.Counter("trpc.UdpServerTransportUnRead")
	// udp client readframe收包失败，一般是非法包
	UDPClientTransportReadFail = metrics.Counter("trpc.UdpClientTransportReadFail")
	// udp client readframe解包后还剩多余的数据
	UDPClientTransportUnRead = metrics.Counter("trpc.UdpClientTransportUnRead")
	// udp 服务端接收到的请求包大小
	UDPServerTransportReceiveSize = metrics.Gauge("trpc.UdpServerTransportReceiveSize")
	// udp 服务端接发送的响应包大小
	UDPServerTransportSendSize = metrics.Gauge("trpc.UdpServerTransportSendSize")
	// tcp协程池下接收队列溢出，请求量过大，过载了
	TCPServerTransportJobQueueFullFail = metrics.Counter("trpc.TcpServerTransportJobQueueFullFail")
	// udp协程池下接收队列溢出，请求量过大，过载了
	UDPServerTransportJobQueueFullFail = metrics.Counter("trpc.UdpServerTransportJobQueueFullFail")
	// 请求被过载保护限流了
	TCPServerTransportRequestLimitedByOverloadCtrl = metrics.Counter("trpc.TcpServerTransportRequestLimitedByOverloadCtrl")
	// TCPServerAsyncGoroutineScheduleDelay 为开启异步时，协程池的调度耗时，单位为 us。
	// 不要改变名字，过载保护算法依赖它。
	TCPServerAsyncGoroutineScheduleDelay = metrics.Gauge("trpc.TcpServerAsyncGoroutineScheduleDelay_us")

	// -----------------------------log----------------------------- //
	// 打印日志过快，队列无法承受，丢弃日志
	LogQueueDropNum = metrics.Counter("trpc.LogQueueDropNum")
	// 日志写入量
	LogWriteSize = metrics.Counter("trpc.LogWriteSize")
	// -----------------------------client----------------------------- //
	// 客户端寻址下游节点失败，一般是名字服务配置异常，或者所有节点熔断
	SelectNodeFail = metrics.Counter("trpc.SelectNodeFail")
	// 客户端调用没有配置protocol协议
	ClientCodecEmpty = metrics.Counter("trpc.ClientCodecEmpty")
	// 加载客户端配置失败，一般是client配置有问题
	LoadClientConfigFail = metrics.Counter("trpc.LoadClientConfigFail")
	// 加载客户端拦截器失败，一般是配置文件中的client filter数组配置了不存在的拦截器
	LoadClientFilterConfigFail = metrics.Counter("trpc.LoadClientFilterConfigFail")
	// tcp客户端端接发出请求包大小
	TCPClientTransportSendSize = metrics.Gauge("trpc.TcpClientTransportSendSize")
	// tcp客户端端收到响应包大小
	TCPClientTransportReceiveSize = metrics.Gauge("trpc.TcpClientTransportReceiveSize")
	// udp 客户端端接发出请求包大小
	UDPClientTransportSendSize = metrics.Gauge("trpc.UdpClientTransportSendSize")
	// udp 客户端端收到响应包大小
	UDPClientTransportReceiveSize = metrics.Gauge("trpc.UdpClientTransportReceiveSize")

	// -----------------------------connection pool----------------------------- //
	// 连接池新建连接数
	ConnectionPoolGetNewConnection = metrics.Counter("trpc.ConnectionPoolGetNewConnection")
	// 连接池获取连接出错
	ConnectionPoolGetConnectionErr = metrics.Counter("trpc.ConnectionPoolGetConnectionErr")
	// 连接池对端异常关闭
	ConnectionPoolRemoteErr = metrics.Counter("trpc.ConnectionPoolRemoteErr")
	// 连接池对端返回 EOF
	ConnectionPoolRemoteEOF = metrics.Counter("trpc.ConnectionPoolRemoteEOF")
	// 连接池连接空闲超时
	ConnectionPoolIdleTimeout = metrics.Counter("trpc.ConnectionPoolIdleTimeout")
	// 连接池连接生命周期结束
	ConnectionPoolLifetimeExceed = metrics.Counter("trpc.ConnectionPoolLifetimeExceed")
	// 连接池超过最大连接数限制
	ConnectionPoolOverLimit = metrics.Counter("trpc.ConnectionPoolOverLimit")

	// -----------------------------multiplexed----------------------------- //
	// 连接复用TCP重连失败
	MultiplexedTCPReconnectErr = metrics.Counter("trpc.MultiplexedReconnectErr")

	// -----------------------------other----------------------------- //
	// trpc.GoAndWait 的 panic 次数
	PanicNum = metrics.Counter("trpc.PanicNum")
)
