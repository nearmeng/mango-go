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

type pulsarClient struct {
	pulsarClient pulsar.Client
}

func NewClient(ctx context.Context, clientConfig *mq.ClientConfig) (mq.Client, error) {
	if clientConfig.LogPath == "" {
		clientConfig.LogPath = "./log/pulsar-go.log"
	}
	if err := os.MkdirAll(path.Dir(clientConfig.LogPath), 0777); err != nil {
		log.Error("fail to create pulsar log dir,err: %v", err)
	}
	lFile, err := os.Create(clientConfig.LogPath)
	if err != nil {
		log.Error("fail to init pulsar log,err: %v", err)
		return nil, err
	}
	llog := logrus.StandardLogger()
	llog.SetOutput(lFile)
	pulsarCli, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:    fmt.Sprintf("%s", clientConfig.Url),
		Logger: log2.NewLoggerWithLogrus(llog),
	})
	if err != nil {
		log.Error("fail to creat pulsar message queue,error: %v,client config: %+v", err, clientConfig)
		return nil, err
	}
	cli := &pulsarClient{
		pulsarClient: pulsarCli,
	}
	return cli, nil
}

func (p *pulsarClient) ResourceManager() mq.ResourceManager {
	panic("implement me")
}
func (p *pulsarClient) NewReader(ctx context.Context, config *mq.ReaderConfig) (mq.Reader, error) {
	// 懒加载
	res := &pulsarReader{
		config: config,
		client: p.pulsarClient,
		m:      &sync.Mutex{},
	}
	return res, nil
}
func (p *pulsarClient) NewWriter(ctx context.Context, config *mq.WriterConfig) (mq.Writer, error) {
	w := &pulsarWriter{
		config: config,
		client: p.pulsarClient,
		m:      &sync.Mutex{},
	}
	return w, nil
}
func (p *pulsarClient) TopicPartitions(topic string) ([]string, error) {
	panic("implement me")
}

func (p *pulsarClient) Close() {
	p.pulsarClient.Close()
}
