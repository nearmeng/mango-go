package kafka

import (
	"context"

	"github.com/nearmeng/mango-go/plugin/mq"
)

type KafkaIns struct {
	mqConfig  *mq.MQConfig
	mqClient  mq.Client
	mqReactor mq.Reactor

	mqReader mq.Reader
	mqWriter mq.Writer
}

func NewKafka(conf *mq.MQConfig) (*KafkaIns, error) {
	ctx := context.Background()

	k := &KafkaIns{
		mqConfig:  conf,
		mqClient:  nil,
		mqReactor: nil,
	}

	client, err := NewClient(ctx, conf.ClientConfig)
	if err != nil {
		return nil, err
	}

	k.mqClient = client

	reactor, err := NewReactor(ctx, conf.ReactorConfig)
	if err != nil {
		return nil, err
	}

	k.mqReactor = reactor

	reader, err := k.mqClient.NewReader(ctx, conf.ReaderConfig)
	if err != nil {
		return nil, err
	}
	k.mqReader = reader

	writer, err := k.mqClient.NewWriter(ctx, conf.WriterConfig)
	if err != nil {
		return nil, err
	}
	k.mqWriter = writer

	return k, nil
}

func (k *KafkaIns) SetConfig(conf *mq.MQConfig) {
	k.mqConfig = conf
}

func (k *KafkaIns) GetReader() mq.Reader {
	return k.mqReader
}

func (k *KafkaIns) GetWriter() mq.Writer {
	return k.mqWriter
}

func (k *KafkaIns) GetClient() mq.Client {
	return k.mqClient
}

func (k *KafkaIns) GetReactor() mq.Reactor {
	return k.mqReactor
}
