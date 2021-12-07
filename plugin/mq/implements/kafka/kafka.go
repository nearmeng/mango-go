package kafka

import (
	"context"
	"fmt"

	"github.com/nearmeng/mango-go/plugin/mq"
)

type KafkaIns struct {
	mqConfig  *mq.MQConfig
	mqClient  mq.Client
	mqReactor mq.Reactor

	mqReader map[string]mq.Reader
	mqWriter map[string]mq.Writer
}

func NewKafka(conf *mq.MQConfig) (*KafkaIns, error) {
	ctx := context.Background()

	k := &KafkaIns{
		mqConfig:  conf,
		mqClient:  nil,
		mqReactor: nil,

		mqReader: make(map[string]mq.Reader),
		mqWriter: make(map[string]mq.Writer),
	}

	client, err := NewClient(ctx, &conf.ClientConfig)
	if err != nil {
		return nil, err
	}

	k.mqClient = client

	reactor, err := NewReactor(ctx, &conf.ReactorConfig)
	if err != nil {
		return nil, err
	}

	k.mqReactor = reactor

	for _, readerConf := range conf.ReaderConfig {

		fmt.Printf("reader conf is %v\n", readerConf)
		reader, err := k.mqClient.NewReader(ctx, &readerConf)
		if err != nil {
			return nil, err
		}

		k.mqReader[readerConf.ReaderName] = reader

		fmt.Printf("kafka new reader %s\n", readerConf.ReaderName)
	}

	for _, writerConf := range conf.WriterConfig {

		fmt.Printf("writer conf is %v\n", writerConf)
		writer, err := k.mqClient.NewWriter(ctx, &writerConf)
		if err != nil {
			return nil, err
		}

		fmt.Printf("kafka new writer %s\n", writerConf.WriterName)

		k.mqWriter[writerConf.WriterName] = writer
	}

	return k, nil
}

func (k *KafkaIns) SetConfig(conf *mq.MQConfig) {
	k.mqConfig = conf
}

func (k *KafkaIns) GetReader(name string) mq.Reader {
	return k.mqReader[name]
}

func (k *KafkaIns) GetWriter(name string) mq.Writer {
	return k.mqWriter[name]
}

func (k *KafkaIns) GetClient() mq.Client {
	return k.mqClient
}

func (k *KafkaIns) GetReactor() mq.Reactor {
	return k.mqReactor
}
