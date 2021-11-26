package mq
// Message
// @Description: mq 消息定义接口
type Message interface {
	Key() string                 // 消息分区
	MetaInfo() map[string]string // 消息头部信息
	PayLoad() []byte             // 负载，消息主体
	SeqID() int64                // 消息序列号
	SetSeqID(seqID int64)
	Topic() string // 队列(topic)名字
	Partition() int // 分区
}
