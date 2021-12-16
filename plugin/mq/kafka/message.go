package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type kafkaMessage struct {
	m *kafka.Message
}

func (k *kafkaMessage) Partition() int {
	return int(k.m.TopicPartition.Partition)
}

func (k *kafkaMessage) Topic() string {
	return *k.m.TopicPartition.Topic
}

func (k *kafkaMessage) Key() string {
	return string(k.m.Key)
}

func (k *kafkaMessage) MetaInfo() map[string]string {
	return headerLisToMeatInfo(k.m.Headers)
}

func (k *kafkaMessage) PayLoad() []byte {
	return k.m.Value
}

func (k *kafkaMessage) SeqID() int64 {
	return int64(k.m.TopicPartition.Offset)
}

func (k *kafkaMessage) SetSeqID(seqID int64) {
	k.m.TopicPartition.Offset = kafka.Offset(seqID)
}

func (k *kafkaMessage) String() string {
	return k.m.String()
}

func metaInfoToHeaderList(metaInfo map[string]string) []kafka.Header {
	res := make([]kafka.Header, len(metaInfo))
	idx := 0
	for k, v := range metaInfo {
		res[idx] = kafka.Header{
			Key:   k,
			Value: []byte(v),
		}
		idx += 1
	}
	return res
}

func headerLisToMeatInfo(header []kafka.Header) map[string]string {
	res := make(map[string]string, 0)
	for _, v := range header {
		res[v.Key] = string(v.Value[0])
	}
	return res
}
