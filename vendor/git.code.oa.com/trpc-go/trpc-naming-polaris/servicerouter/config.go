package servicerouter

// Config 配置
type Config struct {
	// Enable 配置是否开启服务路由功能
	Enable bool
	// EnableCanary 配置是否开启金丝雀功能
	EnableCanary bool
}

const (
	setEnableKey   string = "internal-enable-set"
	setNameKey     string = "internal-set-name"
	setEnableValue string = "Y"
)
