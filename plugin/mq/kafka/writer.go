package kafka

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
)

type kafkaWriter struct {
	p                 *kafka.Producer
	config            *mq.WriterConfig
	writeInterceptors []*mq.WriterInterceptor
	close             chan bool
}

func (w *kafkaWriter) AddWriterInterceptor(interceptor mq.WriterInterceptor) {
	if w.writeInterceptors == nil {
		w.writeInterceptors = make([]*mq.WriterInterceptor, 0)
	}
	w.writeInterceptors = append(w.writeInterceptors, &interceptor)
}
func (w *kafkaWriter) runAfterInterceptor(ctx context.Context) {
	for true {
		select {
		case e := <-w.p.Events():
			switch e.(type) {
			case *kafka.Message:
				w.invokeAfterInterceptor(ctx, &kafkaMessage{m: e.(*kafka.Message)})
			}
		case <-w.close:
			log.Debug("close event loop..")
			return
		}
	}
}

func (w *kafkaWriter) RemoveWriterInterceptor(interceptor mq.WriterInterceptor) {
	idx := -1
	for i, inter := range w.writeInterceptors {
		// TODO: 可以优化成类型相等
		if inter == &interceptor {
			idx = i
			break
		}
	}
	w.writeInterceptors = append(w.writeInterceptors[:idx], w.writeInterceptors[idx+1:]...)
}
func (w *kafkaWriter) invokePreInterceptor(ctx context.Context, messages ...mq.Message) {
	for _, msg := range messages {
		for _, inter := range w.writeInterceptors {
			(*inter).PreWrite(ctx, msg)
		}
	}
}
func (w *kafkaWriter) invokeAfterInterceptor(ctx context.Context, messages ...mq.Message) {
	log.Debug("invoke after interceptor, len: %v", len(w.writeInterceptors))
	for _, msg := range messages {
		for _, inter := range w.writeInterceptors {
			(*inter).AfterWrite(ctx, msg)
		}
	}
}

func (w *kafkaWriter) WriteMessageAsync(ctx context.Context, msg mq.Message, callBack mq.CallBackFunc) error {
	c := make(chan kafka.Event, 1)
	kafkaMsg := w.constructMessage(ctx, msg)
	go w.invokePreInterceptor(ctx, msg)
	if err := w.p.Produce(kafkaMsg, c); err != nil {
		close(c)
		return err
	}
	go w.invokeAfterInterceptor(ctx, msg)

	go func() {
		defer close(c)
		val := <-c
		switch i := val.(type) {
		case *kafka.Message:
			callBack(int64(i.TopicPartition.Offset), &kafkaMessage{i}, nil)
		case error:
			callBack(msg.SeqID(), msg, i)
		}
	}()
	return nil
}

func (w *kafkaWriter) WriteMessage(ctx context.Context, msg mq.Message) (int64, error) {
	c := make(chan kafka.Event, 1)
	errCh := make(chan error, 1)
	kafkaMsg := w.constructMessage(ctx, msg)
	go w.invokePreInterceptor(ctx, msg)
	go func() {
		defer close(errCh)
		select {
		case errCh <- w.p.Produce(kafkaMsg, c): // 1
		case <-ctx.Done():
			return
		}
	}()

	for {
		select {
		// context取消
		case <-ctx.Done():
			// TODO: 这里原生API默认非阻塞写，不支持通过context提前结束，提前退出存在channel泄漏,关闭channel可能会panic
			return 0, errors.New("write message operate cancel")
		// produce 返回错误
		case err := <-errCh:
			// 没有错误继续等待
			if err != nil {
				return 0, err
			}
		// produce 成功
		case val := <-c:
			go w.invokeAfterInterceptor(ctx, msg)
			switch i := val.(type) {
			case *kafka.Message:
				return int64(i.TopicPartition.Offset), nil
			}
		}
	}

}

func (w *kafkaWriter) constructMessage(ctx context.Context, msg mq.Message) *kafka.Message {
	topic := msg.Topic()
	var partitions []int
	if w.config.Properties["partition"] != "" {
		p, _ := strconv.ParseInt(w.config.Properties["partition"], 10, 64)
		partitions = make([]int, p)
		for i := 0; i < int(p); i++ {
			partitions[i] = i
		}
	}
	m := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value:   msg.PayLoad(),
		Key:     []byte(msg.Key()),
		Headers: metaInfoToHeaderList(msg.MetaInfo()),
	}
	if msg.SeqID() > 0 {
		m.TopicPartition.Offset = kafka.Offset(msg.SeqID())
	}
	if w.config.Balancer != nil && partitions != nil {
		m.TopicPartition.Partition = int32(w.config.Balancer(msg, partitions))
	}
	return m
}

func (w *kafkaWriter) Close() {
	log.Debug("wait to flush...,%v", fmt.Sprintf("%dms", 15*1000))
	w.p.Flush(10 * 1000)
	w.close <- true
	w.p.Close()
}

func (w *kafkaWriter) GetConfig() *mq.WriterConfig {
	return w.config
}
func (w *kafkaWriter) SetConfig(config *mq.WriterConfig) {
	w.config = config
}
