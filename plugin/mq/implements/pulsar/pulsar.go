package pulsar

import (
	"context"

	"github.com/nearmeng/mango-go/plugin/mq"
)

type PulsarIns struct {
	mqConfig *mq.MQConfig
	mqClient mq.Client

	mqReader map[string]mq.Reader
	mqWriter map[string]mq.Writer
}

func NewPulsar(conf *mq.MQConfig) (*PulsarIns, error) {
	ctx := context.Background()

	k := &PulsarIns{
		mqConfig: conf,
		mqClient: nil,

		mqReader: make(map[string]mq.Reader),
		mqWriter: make(map[string]mq.Writer),
	}

	client, err := NewClient(ctx, &conf.ClientConfig)
	if err != nil {
		return nil, err
	}

	k.mqClient = client

	for _, readerConf := range conf.ReaderConfig {
		reader, err := k.mqClient.NewReader(ctx, &readerConf)
		if err != nil {
			return nil, err
		}

		k.mqReader[readerConf.ReaderName] = reader
	}

	for _, writerConf := range conf.WriterConfig {
		writer, err := k.mqClient.NewWriter(ctx, &writerConf)
		if err != nil {
			return nil, err
		}

		k.mqWriter[writerConf.WriterName] = writer
	}

	return k, nil
}

func (k *PulsarIns) SetConfig(conf *mq.MQConfig) {
	k.mqConfig = conf
}

func (k *PulsarIns) GetReader(name string) mq.Reader {
	return k.mqReader[name]
}

func (k *PulsarIns) GetWriter(name string) mq.Writer {
	return k.mqWriter[name]
}

func (k *PulsarIns) GetClient() mq.Client {
	return k.mqClient
}
