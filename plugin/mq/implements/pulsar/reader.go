package pulsar

import (
	"context"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
	"github.com/pkg/errors"
)

type pulsarReader struct {
	consumer        pulsar.Consumer
	client          pulsar.Client
	config          *mq.ReaderConfig
	m               *sync.Mutex
	readInterceptor []*mq.ReaderInterceptor
}

func (p *pulsarReader) ReadMessageWithOption(ctx context.Context, info mq.QueueMetaOption) (mq.Message, error) {
	panic("implement me")
}

// AddReaderInterceptor 只有在接收消息前修改interceptor才能生效
func (p *pulsarReader) AddReaderInterceptor(interceptor mq.ReaderInterceptor) {
	if p.readInterceptor == nil {
		p.readInterceptor = make([]*mq.ReaderInterceptor, 0)
	}
	p.readInterceptor = append(p.readInterceptor, &interceptor)
}

// RemoveReaderInterceptor  只有在接收消息前修改interceptor才能生效
func (p *pulsarReader) RemoveReaderInterceptor(interceptor mq.ReaderInterceptor) {
	idx := -1
	for i, inter := range p.readInterceptor {
		if inter == &interceptor {
			idx = i
			break
		}
	}
	if idx != -1 {
		p.readInterceptor = append(p.readInterceptor[:idx], p.readInterceptor[idx+1:]...)
	}
	return
}

func (p *pulsarReader) Ack(ctx context.Context, messages ...mq.Message) {
	if p.config.AutoCommit {
		log.Error("auto_commit is enable, ack() is useless")
	}
	for _, msg := range messages {
		p.consumer.AckID(msg.(*pulsarMessage).MessageID())
		// 执行拦截器
		if len(p.readInterceptor) != 0 {
			go func() {
				for _, inter := range p.readInterceptor {
					(*inter).AfterRead(ctx, msg)
				}
			}()
		}
	}
}

func (p *pulsarReader) GetConfig() *mq.ReaderConfig {
	return p.config
}

type interceptor struct {
	inter *mq.ReaderInterceptor
}

func (i interceptor) BeforeConsume(message pulsar.ConsumerMessage) {
	(*i.inter).PreRead(context.Background(), &pulsarMessage{message.Message})
}

func (i interceptor) OnAcknowledge(consumer pulsar.Consumer, msgID pulsar.MessageID) {

}

func (i interceptor) OnNegativeAcksSend(consumer pulsar.Consumer, msgIDs []pulsar.MessageID) {
}

func (p *pulsarReader) init(ctx context.Context) error {
	log.Info("init pulsar reader")
	cfg := consumerOptions(p.config)
	cfg.Interceptors = make([]pulsar.ConsumerInterceptor, 0)
	// 注入拦截器
	for _, inter := range p.readInterceptor {
		cfg.Interceptors = append(cfg.Interceptors, interceptor{inter: inter})
	}
	// 生成consumer
	c, err := p.client.Subscribe(*cfg)
	if err != nil {
		log.Error("fail to create pulsar message,err:%v", err)
		return err
	}
	p.consumer = c
	return nil
}
func (p *pulsarReader) ReadMessage(ctx context.Context) (res mq.Message, err error) {
	pMsg := &pulsarMessage{}
	// 懒加载
	if p.consumer == nil {
		p.m.Lock()
		defer p.m.Unlock()
		if p.consumer == nil {
			if err := p.init(ctx); err != nil {
				log.Error("fail to init consumer,err: %v", err)
				return nil, err
			}
		}
	}
	done := make(chan interface{}, 1)
	go func() {
		defer close(done)
		msg, err := p.consumer.Receive(ctx)
		if msg != nil {
			done <- msg
		} else {
			log.Error("receive fail, err:%v", err)
			done <- errors.Wrap(err, "receive failed")
		}
	}()
	//msg, err := p.consumer.Receive(ctx)
	//if err != nil {
	//	log.Error("receive fail, err:%v", err)
	//	return nil, err
	//}
	var val interface{}
	select {
	case <-ctx.Done():
		val = <-done
	case val = <-done:
	}
	switch i := val.(type) {
	case error:
		err = i
	case pulsar.Message:
		pMsg.msg = i
	}
	res = pMsg
	if p.config.AutoCommit {
		p.Ack(ctx, res)
	}
	return res, err
}
func (p *pulsarReader) SetConfig(config *mq.ReaderConfig) {
	p.config = config
}
func (p *pulsarReader) Close() {
	if p.consumer != nil {
		p.consumer.Close()
	}
}
