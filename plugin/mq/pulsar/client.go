package pulsar

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"

	log2 "github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
	"github.com/sirupsen/logrus"

	"github.com/apache/pulsar-client-go/pulsar"
)

type PulsarClient struct {
	mqConfig     *mq.MQConfig
	pulsarClient pulsar.Client
	mqReader     map[string]mq.Reader
	mqWriter     map[string]mq.Writer
}

func NewClient(conf *mq.MQConfig) (mq.Client, error) {
	logPath := conf.ClientConfig.LogPath

	if logPath == "" {
		conf.ClientConfig.LogPath = "./log/pulsar-go.log"
	}

	if err := os.MkdirAll(path.Dir(logPath), 0777); err != nil {
		log.Error("fail to create pulsar log dir,err: %v", err)
	}

	lFile, err := os.Create(logPath)
	if err != nil {
		log.Error("fail to init pulsar log,err: %v", err)
		return nil, err
	}

	llog := logrus.StandardLogger()
	llog.SetOutput(lFile)
	llog.SetLevel(logrus.DebugLevel)
	pulsarCli, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:    fmt.Sprintf("%s", conf.ClientConfig.Url),
		Logger: log2.NewLoggerWithLogrus(llog),
	})
	if err != nil {
		log.Error("fail to creat pulsar message queue,error: %v,client config: %+v", err, conf.ClientConfig)
		return nil, err
	}

	cli := &PulsarClient{
		mqConfig:     conf,
		pulsarClient: pulsarCli,
		mqReader:     make(map[string]mq.Reader),
		mqWriter:     make(map[string]mq.Writer),
	}

	for _, readerConf := range conf.ReaderConfig {
		reader, err := cli.NewReader(context.Background(), &readerConf)
		if err != nil {
			return nil, err
		}

		cli.mqReader[readerConf.ReaderName] = reader
	}

	for _, writerConf := range conf.WriterConfig {
		writer, err := cli.NewWriter(context.Background(), &writerConf)
		if err != nil {
			return nil, err
		}

		cli.mqWriter[writerConf.WriterName] = writer
	}

	return cli, nil
}

func (p *PulsarClient) SetConfig(conf *mq.MQConfig) {
	p.mqConfig = conf
}

func (p *PulsarClient) GetReader(name string) mq.Reader {
	return p.mqReader[name]
}

func (p *PulsarClient) GetWriter(name string) mq.Writer {
	return p.mqWriter[name]
}

func (p *PulsarClient) ResourceManager() mq.ResourceManager {
	panic("implement me")
}

func (p *PulsarClient) NewReader(ctx context.Context, config *mq.ReaderConfig) (mq.Reader, error) {
	// 懒加载
	res := &pulsarReader{
		config: config,
		client: p.pulsarClient,
		m:      &sync.Mutex{},
	}
	return res, nil
}
func (p *PulsarClient) NewWriter(ctx context.Context, config *mq.WriterConfig) (mq.Writer, error) {
	w := &pulsarWriter{
		config: config,
		client: p.pulsarClient,
		m:      &sync.Mutex{},
	}
	return w, nil
}
func (p *PulsarClient) TopicPartitions(topic string) ([]string, error) {
	panic("implement me")
}

func (p *PulsarClient) Close() {
	p.pulsarClient.Close()
}
