package kafka

import (
	"context"
	"errors"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
)

type kafkaReader struct {
	consumer         *kafka.Consumer
	config           *mq.ReaderConfig
	readInterceptors []*mq.ReaderInterceptor
	startOffset      int64
}

func (r *kafkaReader) ReadMessage(ctx context.Context) (mq.Message, error) {

	done := make(chan interface{}, 1)
	go func() {
		defer close(done)
		for {
			m, err := r.consumer.ReadMessage(0)
			kafkaErr := err.(kafka.Error)
			// 不是超时错误
			if kafkaErr.Code() != kafka.ErrTimedOut {
				done <- err
				return
			}
			if m != nil {
				done <- m
				return
			}
			select {
			case <-ctx.Done():
				done <- errors.New("operate time out")
				return
			}
		}
	}()
	var val interface{}
	res := &kafkaMessage{}

	select {
	case <-ctx.Done():
		val = <-done
	case val = <-done:
	}
	switch i := val.(type) {
	case error:
		return nil, i
	case *kafka.Message:
		res.m = i
	}
	// 拦截器
	go r.invokePreInterceptor(ctx, res)
	return res, nil
}

func (r *kafkaReader) ReadMessageWithOption(ctx context.Context, info mq.QueueMetaOption) (mq.Message, error) {
	panic("implement me")
}

func (r *kafkaReader) AddReaderInterceptor(interceptor mq.ReaderInterceptor) {
	if r.readInterceptors == nil {
		r.readInterceptors = make([]*mq.ReaderInterceptor, 0)
	}
	r.readInterceptors = append(r.readInterceptors, &interceptor)
}
func (r *kafkaReader) invokePreInterceptor(ctx context.Context, messages ...mq.Message) {
	for _, msg := range messages {
		for _, inter := range r.readInterceptors {
			(*inter).PreRead(ctx, msg)
		}
	}
}
func (r *kafkaReader) invokeAfterInterceptor(ctx context.Context, messages ...mq.Message) {
	for _, msg := range messages {
		for _, inter := range r.readInterceptors {
			(*inter).AfterRead(ctx, msg)
		}
	}
}

func (r *kafkaReader) RemoveReaderInterceptor(interceptor mq.ReaderInterceptor) {
	idx := -1
	for i, inter := range r.readInterceptors {
		// TODO: 可以优化成类型相等
		if inter == &interceptor {
			idx = i
			break
		}
	}
	r.readInterceptors = append(r.readInterceptors[:idx], r.readInterceptors[idx+1:]...)
}
func (r *kafkaReader) GetConfig() *mq.ReaderConfig {
	return r.config
}
func (r *kafkaReader) SetConfig(config *mq.ReaderConfig) {
	r.config = config
}
func (r *kafkaReader) Ack(ctx context.Context, messages ...mq.Message) {
	partitions := make([]kafka.TopicPartition, len(messages))
	for i, msg := range messages {
		topic := msg.Topic()
		offset := msg.SeqID()
		partitions[i] = kafka.TopicPartition{
			Topic:     &topic,
			Partition: int32(msg.Partition()),
			Offset:    kafka.Offset(offset),
		}
	}
	if !r.config.AutoCommit {
		log.Debug("commit offset %v ~ %v", partitions[0].Offset, partitions[len(partitions)-1].Offset)
		_, err := r.consumer.CommitOffsets(partitions)
		if err != nil {
			log.Error("fail to commit ,err:%v", err)
		}
	}
	go r.invokeAfterInterceptor(ctx, messages...)
}
func (r *kafkaReader) Close() {
	// panic("implement me")
}
