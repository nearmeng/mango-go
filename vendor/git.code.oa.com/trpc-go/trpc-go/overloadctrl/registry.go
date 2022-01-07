package overloadctrl

// AnyMethod 表示任意方法名。
const AnyMethod = "*"

var (
	clientBuilders = make(map[string]Builder)
	serverBuilders = make(map[string]Builder)
)

// Builder 定义了过载保护构造器的形式。
type Builder func(*ServiceMethodInfo) OverloadController

// ServiceMethodInfo 是被调的信息。
type ServiceMethodInfo struct {
	ServiceName string
	MethodName  string
}

// RegisterClient 注册客户端过载保护构造器。
func RegisterClient(name string, newOC Builder) {
	clientBuilders[name] = newOC
}

// RegisterServer 注册服务端过载保护构造器。
func RegisterServer(name string, newOC Builder) {
	serverBuilders[name] = newOC
}

// GetClient 获取客户端过载保护构造器。
func GetClient(name string) Builder {
	return clientBuilders[name]
}

// GetServer 获取服务端过载保护构造器。
func GetServer(name string) Builder {
	return serverBuilders[name]
}
