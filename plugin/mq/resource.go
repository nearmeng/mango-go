package mq

// ResourceManager mq的服务端信息管理接口
// @Description:
type ResourceManager interface {
	CreateNamespace(string) error
	DeleteNamespace(string) error
	ListNamespace() ([]string, error)
	CreateQueue(string) error
	DeleteQueue(string) error
}
