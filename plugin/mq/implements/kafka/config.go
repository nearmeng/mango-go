package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nearmeng/mango-go/plugin/mq"
)

func configMap(config *mq.ClientConfig) *kafka.ConfigMap {
	res := &kafka.ConfigMap{
		"bootstrap.servers": config.Url,
	}

	return res
}

func withClientConfig(config *kafka.ConfigMap, clientConfig *mq.ClientConfig) *kafka.ConfigMap {
	if config != nil {
		_ = config.SetKey("bootstrap.servers", clientConfig.Url)
	}
	return config
}
func withWriterConfig(config *kafka.ConfigMap, writerConfig *mq.WriterConfig) *kafka.ConfigMap {
	if config != nil {
		if writerConfig.BatchBytes != 0 {
			_ = config.SetKey("batch.size", writerConfig.BatchBytes)
		}
		if writerConfig.WriteTimeout != 0 {
			_ = config.SetKey("request.timeout.ms", writerConfig.WriteTimeout)
		}
	}
	return config
}
func withReaderConfig(config *kafka.ConfigMap, readerConfig *mq.ReaderConfig) *kafka.ConfigMap {
	if config != nil {
		_ = config.SetKey("enable.auto.commit", readerConfig.AutoCommit)
		_ = config.SetKey("group.id", readerConfig.ReaderName)
		if readerConfig.OffsetReset != "" {
			_ = config.SetKey("auto.offset.reset", readerConfig.OffsetReset)
		}
	}
	return config
}
func withReactorConfig(config *kafka.ConfigMap, reactorConfig *mq.ReactorConfig) *kafka.ConfigMap {
	if config != nil {
		_ = config.SetKey("bootstrap.servers", reactorConfig.Url)
		_ = config.SetKey("group.id", reactorConfig.ReactorName)
	}
	return config
}
