package mq

type MQConfig struct {
	ReaderConfig  []ReaderConfig `mapstructure:"reader_config"`
	WriterConfig  []WriterConfig `mapstructure:"writer_config"`
	ClientConfig  ClientConfig   `mapstructure:"client_config"`
	ReactorConfig ReactorConfig  `mapstructure:"reactor_config"`
}

// ReaderConfig
//  @Description: mq reader配置项
type ReaderConfig struct {
	Topic            []string          `mapstructure:"topic"`
	AutoCommit       bool              `mapstructure:"auto_commit"`
	TopicPattern     string            `mapstructure:"topic_pattern"`
	OffsetReset      string            `mapstructure:"offset_reset"` // 查找不到提交offset记录时的策略
	ReaderName       string            `mapstructure:"reader_name"`
	StartOffset      int64             `mapstructure:"start_offset"`
	SubscriptionType int               `mapstructure:"subscription_type"` // pulsar conf
	Partition        int               `mapstructure:"partition"`
	Properties       map[string]string `mapstructure:"properties"` //可拓展接口
}

// Balancer 负载均衡函数，用于生产者的分区选择策略定制化
type Balancer func(Message, []int) int

// WriterConfig
// @Description:  mq writer配置项
type WriterConfig struct {
	WriterName   string                      `mapstructure:"writer_name"` // writer实现exactly once时需要给writer指定一个name
	Topic        string                      `mapstructure:"topic"`       // 消息维度和Writer维度都设置了topic可以自行指定优先级，这里可以规范成，消息指定的topic优先级大于WriterConfig配置的Topic，TODO: 如何让开发者不违法这个约定
	Balancer     Balancer                    `mapstructure:"-"`           // 分区策略
	Batch        bool                        `mapstructure:"batch"`
	MaxAttempts  int                         `mapstructure:"max_attempts"`
	BatchSize    uint                        `mapstructure:"batch_size"`    // 被缓存的消息数量
	BatchBytes   uint                        `mapstructure:"batch_bytes"`   // 消息大小
	CallBack     func(int64, Message, error) `mapstructure:"-"`             // TODO: 异步回调函数待接入
	WriteTimeout int                         `mapstructure:"write_timeout"` // 消息写入后多久没有被broker确认后重试，单位：s
	Properties   map[string]string           `mapstructure:"properties"`    //可拓展接口
}

// ClientConfig
// @Description: mq 客户端相关配置项
type ClientConfig struct {
	Url        string            `mapstructure:"url"`
	LogPath    string            `mapstructure:"log_path"`
	Properties map[string]string `mapstructure:"properties"` //可拓展接口
}

// ReactorConfig
// @Description: mq reactor相关配置项
// Deprecated: 封装较为简单建议使用 Client API
type ReactorConfig struct {
	Url         string            `mapstructure:"url"`
	ReactorName string            `mapstructure:"reactor_name"`
	Properties  map[string]string `mapstructure:"properties"` //可拓展接口
}
