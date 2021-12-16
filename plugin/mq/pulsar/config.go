package pulsar

import (
	"context"
	"strconv"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
)

func consumerOptions(config *mq.ReaderConfig) *pulsar.ConsumerOptions {
	res := &pulsar.ConsumerOptions{}
	log.Debug("reader config: %+v", config)
	if config.TopicPattern != "" && len(config.Topic) != 0 {
		log.Error("Topic or TopicPattern only one  can be assigned")
	}
	if len(config.Topic) == 1 {
		res.Topic = config.Topic[0]
	} else {
		res.Topics = config.Topic
	}
	res.TopicsPattern = config.TopicPattern
	res.SubscriptionName = config.ReaderName
	res.Type = pulsar.SubscriptionType(config.SubscriptionType)
	res.Properties = config.Properties
	return res
}

func producerOption(ctx context.Context, config *mq.WriterConfig) *pulsar.ProducerOptions {
	cfg := &pulsar.ProducerOptions{
		Topic:      config.Topic,
		Name:       config.WriterName,
		Properties: config.Properties,
	}
	if config.BatchSize > 0 {
		cfg.BatchingMaxMessages = config.BatchSize
	}
	if config.BatchBytes > 0 {
		cfg.BatchingMaxSize = config.BatchBytes
	}
	if config.Balancer != nil {
		cfg.MessageRouter = BalancerWrapper(ctx, config.Balancer)
	}
	if config.WriterName == "" {
		cfg.Name = "default_producer"
	}
	if config.WriteTimeout != 0 {
		cfg.SendTimeout = time.Duration(config.WriteTimeout) * time.Second
	}
	if config.Properties["compression_type"] != "" {
		t, err := strconv.ParseInt(config.Properties["compression_type"], 10, 64)
		if err != nil {
			log.Error("parse compression_type failed,err:%v", err)
		} else {
			cfg.CompressionType = pulsar.CompressionType(t)
		}
	}
	return cfg
}

// BalancerWrapper 用于封装配置的分区策略，生成MessageRouter函数
func BalancerWrapper(ctx context.Context, balancer mq.Balancer) func(*pulsar.ProducerMessage, pulsar.TopicMetadata) int {
	return func(message *pulsar.ProducerMessage, metadata pulsar.TopicMetadata) int {
		m := &producerMessage{m: message}
		partitions := make([]int, metadata.NumPartitions())
		for i := 0; i < len(partitions); i++ {
			partitions[i] = i
		}
		return balancer(m, partitions)
	}
}
