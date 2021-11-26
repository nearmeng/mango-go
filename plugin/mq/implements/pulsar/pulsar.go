package pulsar

import (
	"context"

	"github.com/nearmeng/mango-go/plugin/mq"
)

type PulsarIns struct {
	mqConfig *mq.MQConfig
	mqClient mq.Client

	mqReader mq.Reader
	mqWriter mq.Writer
}

func NewPulsar(conf *mq.MQConfig) (*PulsarIns, error) {
	ctx := context.Background()

	k := &PulsarIns{
		mqConfig: conf,
		mqClient: nil,

		mqReader: nil,
		mqWriter: nil,
	}

	client, err := NewClient(ctx, conf.ClientConfig)
	if err != nil {
		return nil, err
	}

	k.mqClient = client

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

func (k *PulsarIns) SetConfig(conf *mq.MQConfig) {
	k.mqConfig = conf
}

func (k *PulsarIns) GetReader() mq.Reader {
	return k.mqReader
}

func (k *PulsarIns) GetWriter() mq.Writer {
	return k.mqWriter
}

func (k *PulsarIns) GetClient() mq.Client {
	return k.mqClient
}
