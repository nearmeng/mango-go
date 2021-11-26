package pulsar

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/mq"
)

type pulsarWriter struct {
	producer         pulsar.Producer
	config           *mq.WriterConfig
	client           pulsar.Client
	m                *sync.Mutex
	writeInterceptor []*mq.WriterInterceptor
}

// AddWriterInterceptor 只有在接收消息前修改interceptor才能生效
func (w *pulsarWriter) AddWriterInterceptor(interceptor mq.WriterInterceptor) {
	if w.writeInterceptor == nil {
		w.writeInterceptor = make([]*mq.WriterInterceptor, 0)
	}
	w.writeInterceptor = append(w.writeInterceptor, &interceptor)
}

// RemoveWriterInterceptor 只有在接收消息前修改interceptor才能生效
func (w *pulsarWriter) RemoveWriterInterceptor(interceptor mq.WriterInterceptor) {
	idx := -1
	for i, inter := range w.writeInterceptor {
		if inter == &interceptor {
			idx = i
			break
		}
	}
	if idx != -1 {
		w.writeInterceptor = append(w.writeInterceptor[:idx], w.writeInterceptor[idx+1:]...)
	}
}

func (w *pulsarWriter) constructMessage(ctx context.Context, msg mq.Message) *pulsar.ProducerMessage {
	producerMessage := &pulsar.ProducerMessage{
		Payload:    msg.PayLoad(),
		Key:        msg.Key(),
		Properties: msg.MetaInfo(),
	}
	// 指定消息的SequenceID
	if msg.SeqID() > 0 {
		seqID := msg.SeqID()
		producerMessage.SequenceID = &seqID
		log.Debug("seqId:%v", msg.SeqID)
	}
	return producerMessage
}

func (w *pulsarWriter) WriteMessage(ctx context.Context, msg mq.Message) (int64, error) {

	if ok, err := w.checkProducer(ctx); !ok {
		return 0, err
	}
	producerMessage := w.constructMessage(ctx, msg)

	done := make(chan interface{}, 1)
	defer close(done)
	go func() {
		msgId, err := w.producer.Send(ctx, producerMessage)
		if err != nil {
			done <- err
		} else if msgId == nil {
			done <- 0
		} else {
			res := strings.Split(fmt.Sprintf("%s", msgId), ":")
			if len(res) < 2 {
				done <- 0
			} else {
				offset, _ := strconv.ParseInt(res[1], 10, 64)
				done <- offset
			}
		}
	}()
	var val interface{}
	select {
	// 提前结束
	case <-ctx.Done():
		{
			// 等待下一层调用取消
			<-done
			return 0, ctx.Err()
		}
	// 正常返回
	case val = <-done:
		{
			switch i := val.(type) {
			case int64:
				return i, nil
			case error:
				return 0, i
			}
		}

	}

	return 0, errors.New("unexpect error")
}
func (w *pulsarWriter) WriteMessageAsync(ctx context.Context, msg mq.Message, callBack mq.CallBackFunc) error {
	if ok, err := w.checkProducer(ctx); !ok {
		return err
	}
	producerMessage := w.constructMessage(ctx, msg)
	pulsarCallback := pulsarCallBackWrapper(callBack)
	w.producer.SendAsync(ctx, producerMessage, pulsarCallback)
	return nil
}
func pulsarCallBackWrapper(callBack mq.CallBackFunc) func(id pulsar.MessageID, message *pulsar.ProducerMessage, err error) {
	return func(id pulsar.MessageID, message *pulsar.ProducerMessage, err error) {
		if callBack == nil {
			return
		}
		//log.Debug("message: %+v", message)
		var offset int64 = 0
		if message.SequenceID != nil {
			offset = *message.SequenceID
		}
		//log.Debug("seqId:%v", offset)
		callBack(offset, &producerMessage{message}, err)
	}
}
func (w *pulsarWriter) checkProducer(ctx context.Context) (bool, error) {
	if w.producer == nil {
		w.m.Lock()
		defer w.m.Unlock()
		if w.producer == nil {
			if err := w.init(ctx); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}
func (w *pulsarWriter) GetConfig() *mq.WriterConfig {
	return w.config
}
func (w *pulsarWriter) SetConfig(config *mq.WriterConfig) {
	w.config = config
}

type writeInterceptor struct {
	inter *mq.WriterInterceptor
}

func (w writeInterceptor) BeforeSend(producer pulsar.Producer, message *pulsar.ProducerMessage) {
	(*w.inter).PreWrite(context.Background(), &producerMessage{m: message})
}

func (w writeInterceptor) OnSendAcknowledgement(producer pulsar.Producer, message *pulsar.ProducerMessage, msgID pulsar.MessageID) {
	(*w.inter).AfterWrite(context.Background(), &producerMessage{m: message})
}

func (w *pulsarWriter) init(ctx context.Context) error {
	// 懒加载
	log.Info("init pulsar writer")
	cfg := producerOption(ctx, w.config)
	cfg.Interceptors = make([]pulsar.ProducerInterceptor, 0)
	for _, inter := range w.writeInterceptor {
		cfg.Interceptors = append(cfg.Interceptors, &writeInterceptor{inter: inter})
	}
	pulsarProducer, err := w.client.CreateProducer(*cfg)
	if err != nil {
		log.Error("fail to create pulsar writer,err:%v", err)
		return err
	}
	w.producer = pulsarProducer
	return nil
}
func (w *pulsarWriter) Close() {
	if w.producer != nil {
		_ = w.producer.Flush()
		w.producer.Close()
	}
}
