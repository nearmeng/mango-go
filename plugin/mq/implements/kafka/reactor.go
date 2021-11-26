package kafka

import (
	"context"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
)

// KafkaReactor 目前分配策略 handlers : topic / handler = N / 1; readers : topic / message =  1 : 1 ; 后续可以支持更高效的调度
type kafkaReactor struct {
	config   *mq.ReactorConfig
	handlers map[string]mq.MessageHandler
	consumer map[string]*kafka.Consumer
	stop     chan bool
}

func NewReactor(ctx context.Context, config *mq.ReactorConfig) (mq.Reactor, error) {
	return &kafkaReactor{
		config:   config,
		handlers: make(map[string]mq.MessageHandler, 0),
		consumer: make(map[string]*kafka.Consumer, 0),
		stop:     make(chan bool, 1),
	}, nil
}

func (k *kafkaReactor) Register(ctx context.Context, topics []string, handler mq.MessageHandler) {
	for _, topic := range topics {
		k.handlers[topic] = handler
		if k.consumer[topic] == nil {
			c, err := kafka.NewConsumer(withReactorConfig(&kafka.ConfigMap{}, k.config))
			if err != nil {
				log.Error("fail to create consumer,topic:%v,err:%v", topic, err)
				continue
			}
			_ = c.Subscribe(topic, nil)
			k.consumer[topic] = c
		}
	}
}

func (k *kafkaReactor) Run(ctx context.Context) {
	for {
		select {
		case <-k.stop:
			log.Debug("reactor out...")
			return
		default:
			log.Debug("poll ...")
			for topic, consumer := range k.consumer {
				c := kafkaContext{
					consumer: consumer,
					c:        ctx,
				}
				msg, err := consumer.ReadMessage(-1)
				if err != nil {
					log.Error("fail to read msg from %v", topic)
					continue
				}
				handler := k.handlers[topic]
				go func(context.Context) {
					m := kafkaMessage{m: msg}
					handler.Handle(&c, &m)
				}(ctx)
			}
		}
	}

}
func (k *kafkaReactor) Close(ctx context.Context) {
	k.stop <- true
	log.Debug("close consumer")
	for _, c := range k.consumer {
		_ = c.Close()
	}
	log.Debug("close reactor")
	return
}

type kafkaContext struct {
	consumer *kafka.Consumer
	c        context.Context
}

func (ctx *kafkaContext) Deadline() (deadline time.Time, ok bool) {
	return ctx.c.Deadline()
}

func (ctx *kafkaContext) Done() <-chan struct{} {
	return ctx.c.Done()
}

func (ctx *kafkaContext) Err() error {
	return ctx.Err()
}

func (ctx *kafkaContext) Value(key interface{}) interface{} {
	return ctx.Value(key)
}

func (ctx *kafkaContext) Ack(messages ...mq.Message) {
	_, _ = ctx.consumer.Commit()
}
