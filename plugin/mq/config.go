package mq

type MQConfig struct {
	ReaderConfig  *ReaderConfig  `json:"reader" yaml:"reader"`
	WriterConfig  *WriterConfig  `json:"writer" yaml:"writer"`
	ClientConfig  *ClientConfig  `json:"client" yaml:"client"`
	ReactorConfig *ReactorConfig `json:"reactor" yaml:"reactor"`
}

// ReaderConfig
//  @Description: mq reader配置项
//
type ReaderConfig struct {
	Topic            []string          `json:"topic" yaml:"topic"`
	AutoCommit       bool              `json:"auto_commit" yaml:"auto_commit"`
	TopicPattern     string            `json:"topic_pattern" yaml:"topic_pattern"`
	OffsetReset      string            `json:"offset_reset" yaml:"offset_reset"` // 查找不到提交offset记录时的策略
	ReaderName       string            `json:"reader_name"yaml:"reader_name"`
	StartOffset      int64             `json:"start_offset" yaml:"start_offset"`
	SubscriptionType int               `json:"subscription_type" yaml:"subscription_type"` // pulsar conf
	Partition        int               `json:"partition" yaml:"partition"`
	Properties       map[string]string `json:"properties" yaml:"properties"` //可拓展接口
}

// Balancer 负载均衡函数，用于生产者的分区选择策略定制化
type Balancer func(Message, []int) int

// WriterConfig
// @Description:  mq writer配置项
type WriterConfig struct {
	WriterName   string                      `json:"writer_name" yaml:"writer_name"` // writer实现exactly once时需要给writer指定一个name
	Topic        string                      `json:"topic" yaml:"topic"`             // 消息维度和Writer维度都设置了topic可以自行指定优先级，这里可以规范成，消息指定的topic优先级大于WriterConfig配置的Topic，TODO: 如何让开发者不违法这个约定
	Balancer     Balancer                    `json:"-" yaml:"-"`                     // 分区策略
	Batch        bool                        `json:"batch" yaml:"batch"`
	MaxAttempts  int                         `json:"max_attempts" yaml:"max_attempts"`
	BatchSize    uint                        `json:"batch_size" yaml:"batch_size"`       // 被缓存的消息数量
	BatchBytes   uint                        `json:"batch_bytes" yaml:"batch_bytes"`     // 消息大小
	CallBack     func(int64, Message, error) `json:"-" yaml:"-"`                         // TODO: 异步回调函数待接入
	WriteTimeout int                         `json:"write_timeout" yaml:"write_timeout"` // 消息写入后多久没有被broker确认后重试，单位：s
	Properties   map[string]string           `json:"properties" yaml:"properties"`       //可拓展接口
}

// ClientConfig
// @Description: mq 客户端相关配置项
type ClientConfig struct {
	Url        string            `json:"url" yaml:"url"`
	LogPath    string            `json:"log_path" yaml:"log_path"`
	Properties map[string]string `json:"properties" yaml:"properties"` //可拓展接口
}

// ReactorConfig
// @Description: mq reactor相关配置项
// Deprecated: 封装较为简单建议使用 Client API
type ReactorConfig struct {
	Url         string            `json:"url" yaml:"url"`
	ReactorName string            `json:"reactor_name" yaml:"reactor_name"`
	Properties  map[string]string `json:"properties" yaml:"properties"` //可拓展接口
}
