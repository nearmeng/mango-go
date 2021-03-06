package kafka

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
)

type KafkaClient struct {
	mqConfig *mq.MQConfig

	mqReactor mq.Reactor
	mqReader  map[string]mq.Reader
	mqWriter  map[string]mq.Writer
}

func NewClient(conf *mq.MQConfig) (mq.Client, error) {
	cli := &KafkaClient{
		mqConfig:  conf,
		mqReactor: nil,
		mqReader:  make(map[string]mq.Reader),
		mqWriter:  make(map[string]mq.Writer),
	}

	reactor, err := NewReactor(context.Background(), &conf.ReactorConfig)
	if err != nil {
		return nil, err
	}
	cli.mqReactor = reactor

	for _, readerConf := range conf.ReaderConfig {

		reader, err := cli.NewReader(context.Background(), &readerConf)
		if err != nil {
			return nil, err
		}

		cli.mqReader[readerConf.ReaderName] = reader
		log.Info("kafka new reader %s\n", readerConf.ReaderName)
	}

	for _, writerConf := range conf.WriterConfig {

		writer, err := cli.NewWriter(context.Background(), &writerConf)
		if err != nil {
			return nil, err
		}

		cli.mqWriter[writerConf.WriterName] = writer
		log.Info("kafka new writer %s\n", writerConf.WriterName)
	}

	return cli, nil
}

func (p *KafkaClient) SetConfig(conf *mq.MQConfig) {
	p.mqConfig = conf
}

func (p *KafkaClient) GetReader(name string) mq.Reader {
	return p.mqReader[name]
}

func (p *KafkaClient) GetWriter(name string) mq.Writer {
	return p.mqWriter[name]
}

func (p *KafkaClient) GetReactor() mq.Reactor {
	return p.mqReactor
}

func (p *KafkaClient) ResourceManager() mq.ResourceManager {
	panic("implement me")
}

func (p *KafkaClient) NewReader(ctx context.Context, config *mq.ReaderConfig) (mq.Reader, error) {
	consumer, err := kafka.NewConsumer(withReaderConfig(withClientConfig(&kafka.ConfigMap{}, &p.mqConfig.ClientConfig), config))
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

func (p *KafkaClient) NewWriter(ctx context.Context, config *mq.WriterConfig) (mq.Writer, error) {

	producer, err := kafka.NewProducer(withWriterConfig(withClientConfig(&kafka.ConfigMap{}, &p.mqConfig.ClientConfig), config))
	if err != nil {
		log.Error("fail to create producer, err:%v", err)
		return nil, err
	}
	// TODO: ????????????????????????
	//if config.Balancer != nil {
	//	log.Debugf("use user config balancer...")
	//	writer.Balancer = &customBalancer{balancer: config.Balancer}
	//}
	w := &kafkaWriter{
		p:      producer,
		config: config,
		close:  make(chan bool, 1),
	}
	// ???????????????????????????????????????
	go w.runAfterInterceptor(ctx)
	return w, nil
}

func (p *KafkaClient) TopicPartitions(topic string) ([]string, error) {
	return []string{""}, nil
}

func (p *KafkaClient) Close() {
	log.Info("client close")
}
