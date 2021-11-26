package kafka

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
)

type kafkaClient struct {
	clientConfig *mq.ClientConfig
}

func NewClient(ctx context.Context, clientConfig *mq.ClientConfig) (mq.Client, error) {
	cli := &kafkaClient{
		clientConfig: clientConfig,
	}
	return cli, nil
}

func (p *kafkaClient) ResourceManager() mq.ResourceManager {
	panic("implement me")
}

func (p *kafkaClient) NewReader(ctx context.Context, config *mq.ReaderConfig) (mq.Reader, error) {
	log.Info("init reader...")
	consumer, err := kafka.NewConsumer(withReaderConfig(withClientConfig(&kafka.ConfigMap{}, p.clientConfig), config))
	if err != nil {
		log.Error("fail to creat consumer,err:%v", err)
		return nil, err
	}
	_ = consumer.SubscribeTopics(config.Topic, nil)
	r := &kafkaReader{
		consumer: consumer,
		config:   config,
	}
	return r, nil
}

func (p *kafkaClient) NewWriter(ctx context.Context, config *mq.WriterConfig) (mq.Writer, error) {
	producer, err := kafka.NewProducer(withWriterConfig(withClientConfig(&kafka.ConfigMap{}, p.clientConfig), config))
	if err != nil {
		log.Error("fail to create producer, err:%v", err)
		return nil, err
	}
	// TODO: 用户指定分区函数
	//if config.Balancer != nil {
	//	log.Debugf("use user config balancer...")
	//	writer.Balancer = &customBalancer{balancer: config.Balancer}
	//}
	w := &kafkaWriter{
		p:      producer,
		config: config,
		close:  make(chan bool, 1),
	}
	// 启动一个协程处理后置拦截器
	go w.runAfterInterceptor(ctx)
	return w, nil
}

func (p *kafkaClient) TopicPartitions(topic string) ([]string, error) {
	return []string{""}, nil
}

func (p *kafkaClient) Close() {
	log.Info("client close")
}
