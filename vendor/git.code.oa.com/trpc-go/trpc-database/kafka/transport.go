package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/naming/selector"
	"git.code.oa.com/trpc-go/trpc-go/transport"
	dsn "git.code.oa.com/trpc-go/trpc-selector-dsn"
	"github.com/Shopify/sarama"
)

func init() {
	selector.Register("kafka", dsn.DefaultSelector)
	transport.RegisterServerTransport("kafka", DefaultServerTransport)
	transport.RegisterClientTransport("kafka", DefaultClientTransport)
}

// DefaultServerTransport ServerTransport默认实现
var DefaultServerTransport = NewServerTransport()

// NewServerTransport new出来server transport实现
func NewServerTransport(opt ...transport.ServerTransportOption) transport.ServerTransport {
	// option 默认值
	kafkaOpts := &transport.ServerTransportOptions{}
	for _, o := range opt {
		o(kafkaOpts)
	}
	return &ServerTransport{opts: kafkaOpts}
}

// ServerTransport kafka consumer transport
type ServerTransport struct {
	opts *transport.ServerTransportOptions
}

// ListenAndServe 启动监听，如果监听失败则返回错误
func (s *ServerTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) (err error) {
	kafkalsopts := &transport.ListenServeOptions{}
	for _, opt := range opts {
		opt(kafkalsopts)
	}

	kafkaUserConfig, err := parseAddress(kafkalsopts.Address)
	if err != nil {
		return err
	}

	config := sarama.NewConfig()
	config.Version = kafkaUserConfig.version
	config.ClientID = kafkaUserConfig.group

	config.Metadata.Full = false                //禁止拉取所有元数据
	config.Metadata.Retry.Max = 1               //元数据更新重次次数
	config.Metadata.Retry.Backoff = time.Second //元数据更新等待时间

	config.Net.MaxOpenRequests = kafkaUserConfig.netMaxOpenRequests
	config.Net.DialTimeout = kafkaUserConfig.netDailTimeout
	config.Net.ReadTimeout = kafkaUserConfig.netReadTimeout
	config.Net.WriteTimeout = kafkaUserConfig.netWriteTimeout

	config.Consumer.MaxProcessingTime = kafkaUserConfig.maxProcessingTime
	config.Consumer.Fetch.Default = int32(kafkaUserConfig.fetchDefault)
	config.Consumer.Fetch.Max = int32(kafkaUserConfig.fetchMax)
	config.Consumer.Offsets.Initial = kafkaUserConfig.initial
	config.Consumer.Offsets.AutoCommit.Interval = 3 * time.Second //定时多久一次提交消费进度
	config.Consumer.Group.Rebalance.Strategy = kafkaUserConfig.strategy
	config.Consumer.Group.Rebalance.Timeout = kafkaUserConfig.groupRebalanceTimeout
	config.Consumer.Group.Rebalance.Retry.Max = kafkaUserConfig.groupRebalanceRetryMax
	config.Consumer.Group.Session.Timeout = kafkaUserConfig.groupSessionTimeout
	config.Consumer.MaxWaitTime = kafkaUserConfig.maxWaitTime
	kafkaUserConfig.scramClient.config(config)

	// 连接broker，失败会返回错误
	consumerGroup, err := sarama.NewConsumerGroup(kafkaUserConfig.brokers, kafkaUserConfig.group, config)
	if err != nil {
		return err
	}

	go func() {
		bc := &batchConsumer{opts: kafkalsopts, ctx: ctx,
			maxNum: kafkaUserConfig.batchConsumeCount, flushInterval: kafkaUserConfig.batchFlush,
			maxRetry: kafkaUserConfig.maxRetry}
		c := &consumer{opts: kafkalsopts, ctx: ctx, maxRetry: kafkaUserConfig.maxRetry}
		for {
			if consumerGroup != nil {
				if kafkaUserConfig.batchConsumeCount > 0 {
					err = consumerGroup.Consume(ctx, kafkaUserConfig.topics, bc)
				} else {
					err = consumerGroup.Consume(ctx, kafkaUserConfig.topics, c)
				}
			}
			select {
			case <-ctx.Done():
				log.ErrorContextf(ctx, "kafka server transport: context done:%v, close", ctx.Err())
				return
			default:
			}

			time.Sleep(time.Second * 3)
			if err == nil {
				continue
			}

			log.ErrorContextf(ctx, "kafka server transport: consume fail:%v, reconnect", err)
			if consumerGroup != nil {
				consumerGroup.Close()
				consumerGroup = nil
			}

			//重新连接broker，失败会返回错误
			consumerGroup, err = sarama.NewConsumerGroup(kafkaUserConfig.brokers, kafkaUserConfig.group, config)
			if err != nil {
				log.ErrorContextf(ctx, "kafka server transport: consume reconnect fail:%v", err)
			}
		}
	}()

	return nil
}

// consumer 消费者结构
type consumer struct {
	opts     *transport.ListenServeOptions
	ctx      context.Context
	maxRetry int
	retryNum int
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (s *consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (s *consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (s *consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {

		select {
		case <-s.ctx.Done():
			return errors.New("consumer service close")
		default:
		}

		//生成新的空的通用消息结构数据，并保存到ctx里面
		ctx, msg := codec.WithNewMessage(context.Background())

		//填充被调方，自己
		msg.WithCompressType(codec.CompressTypeNoop) //不解压缩
		msg.WithServerReqHead(message)

		_, err := s.opts.Handler.Handle(ctx, nil)
		if err != nil || msg.ServerRspErr() != nil {
			msgInfo := ""
			if message != nil {
				msgInfo = fmt.Sprintf("%+v:%+v:%+v", message.Topic, message.Partition, message.Offset)
			}
			if s.maxRetry == 0 || s.retryNum < s.maxRetry {
				s.retryNum++
				log.ErrorContextf(ctx, "kafka consumer handle fail:%v %v, retry: %+v, msg: %+v",
					err, msg.ServerRspErr(), s.retryNum, msgInfo)
				return nil
			}
		}

		//确认消息消费成功
		s.retryNum = 0
		session.MarkMessage(message, "")
	}

	return nil
}

// batchConsumer 批量消费
type batchConsumer struct {
	opts          *transport.ListenServeOptions
	ctx           context.Context
	maxNum        int // 一批最大数量
	flushInterval time.Duration
	maxRetry      int // 失败最大重试次数
	retryNum      int // 当前重试次数
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (s *batchConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (s *batchConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 批量消费
// 当满足maxNum条消息时触发消费, 刷新间隔到了也会触发消费, 避免消息流量不均匀的情况下阻塞消费。
// 如果业务消费失败则整个批次重试, 不支持只重试失败的消息
func (s *batchConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgArray := make([]*sarama.ConsumerMessage, s.maxNum)
	ticker := time.NewTicker(s.flushInterval)
	idx := 0
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			select {
			case <-s.ctx.Done():
				return errors.New("consumer service close")
			default:
			}

			msgArray[idx] = message
			idx++
			// 满足一批
			if idx >= s.maxNum {
				ctx, msg := codec.WithNewMessage(context.Background())
				msg.WithCompressType(codec.CompressTypeNoop) //不解压缩
				msg.WithServerReqHead(msgArray)
				_, err := s.opts.Handler.Handle(ctx, nil)
				if err != nil || msg.ServerRspErr() != nil {
					if s.maxRetry == 0 || s.retryNum < s.maxRetry {
						s.retryNum++
						log.ErrorContextf(ctx, "kafka consumer handle fail:%v %v, ready retry: %+v",
							err, msg.ServerRspErr(), s.retryNum)
						return nil
					}
				}
				//确认消息消费成功
				session.MarkMessage(message, "")
				idx = 0
				s.retryNum = 0
			}
		case <-ticker.C:
			if idx > 0 {
				ctx, msg := codec.WithNewMessage(context.Background())
				msg.WithCompressType(codec.CompressTypeNoop) //不解压缩
				msg.WithServerReqHead(msgArray[:idx])
				_, err := s.opts.Handler.Handle(ctx, nil)
				if err != nil || msg.ServerRspErr() != nil {
					if s.maxRetry == 0 || s.retryNum < s.maxRetry {
						s.retryNum++
						log.ErrorContextf(ctx, "kafka consumer handle fail:%v %v, ready retry: %+v",
							err, msg.ServerRspErr(), s.retryNum)
						return nil
					}
				}
				//确认消息消费成功
				session.MarkMessage(msgArray[idx-1], "")
				idx = 0
				s.retryNum = 0
			}
		}
	}
}

// ClientTransport client端kafka transport
type ClientTransport struct {
	opts               *transport.ClientTransportOptions
	producers          map[string]*Producer
	producersLock      sync.RWMutex
	asyncProducers     map[string]*Producer
	asyncProducersLock sync.RWMutex
}

// DefaultClientTransport 默认client kafka transport
var DefaultClientTransport = NewClientTransport()

// NewClientTransport 创建kafka transport
func NewClientTransport(opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}

	// 将传入的func option写到opts字段中
	for _, o := range opt {
		o(opts)
	}

	return &ClientTransport{
		opts:           opts,
		producers:      map[string]*Producer{},
		asyncProducers: map[string]*Producer{},
	}
}

// RoundTrip 收发kafka包, 回包kafka response放到ctx里面，这里不需要返回rspbuf
func (ct *ClientTransport) RoundTrip(ctx context.Context, _ []byte,
	callOpts ...transport.RoundTripOption) ([]byte, error) {
	msg := codec.Message(ctx)
	req, ok := msg.ClientReqHead().(*Request)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"kafka client transport: ReqHead should be type of *kafka.Request")
	}
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"kafka client transport: RspHead should be type of *kafka.Response")
	}
	opts := &transport.RoundTripOptions{}
	for _, o := range callOpts {
		o(opts)
	}
	if req.Async {
		producer, err := ct.GetAsyncProducer(opts.Address)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr,
				"kafka client transport GetAsyncProducer: "+err.Error())
		}
		if req.Topic == "" { // 优先取参数传入的topic
			if producer.topic == "" {
				return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport empty topic")
			}
			req.Topic = producer.topic
		}

		select {
		case producer.asyncProducer.Input() <- &sarama.ProducerMessage{
			Topic:   req.Topic,
			Key:     sarama.ByteEncoder(req.Key),
			Value:   sarama.ByteEncoder(req.Value),
			Headers: req.Headers,
		}:
		case <-ctx.Done(): // 如果生产通道阻塞，则返回阻塞超时错误
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"kafka client transport select: async producer message channel block")
		}
	} else {
		producer, err := ct.GetProducer(opts.Address, msg.RequestTimeout())
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport GetProducer:"+err.Error())
		}
		if req.Topic == "" { // 优先取参数传入的topic
			if producer.topic == "" {
				return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport empty topic")
			}
			req.Topic = producer.topic
		}

		if producer.async { // 兼容老sendmessage逻辑
			req.Async = true
			select {
			case producer.asyncProducer.Input() <- &sarama.ProducerMessage{
				Topic:   req.Topic,
				Key:     sarama.ByteEncoder(req.Key),
				Value:   sarama.ByteEncoder(req.Value),
				Headers: req.Headers,
			}:
			case <-ctx.Done(): // 如果生产通道阻塞，则返回阻塞超时错误
				return nil, errs.NewFrameError(errs.RetClientTimeout,
					"kafka client transport select: async producer message channel block")
			}
		} else {
			message := &sarama.ProducerMessage{
				Topic:   req.Topic,
				Key:     sarama.ByteEncoder(req.Key),
				Value:   sarama.ByteEncoder(req.Value),
				Headers: req.Headers,
			}
			rsp.Partition, rsp.Offset, err = producer.syncProducer.SendMessage(message)
			if err != nil {
				return nil, errs.NewFrameError(errs.RetClientNetErr, "kafka client transport SendMessage: "+err.Error())
			}
		}
	}
	return nil, nil
}

// GetProducer 获取生产者
func (ct *ClientTransport) GetProducer(address string, timeout time.Duration) (*Producer, error) {
	ct.producersLock.RLock()
	producer, ok := ct.producers[address]
	ct.producersLock.RUnlock()
	if ok {
		return producer, nil
	}

	ct.producersLock.Lock()
	defer ct.producersLock.Unlock()

	producer, ok = ct.producers[address]
	if ok {
		return producer, nil
	}

	userConfig, err := parseAddress(address)
	if err != nil {
		return nil, err
	}

	if userConfig.async == 1 {
		config := sarama.NewConfig()

		config.Version = userConfig.version
		config.ClientID = userConfig.clientid

		config.Producer.Return.Successes = userConfig.returnSuccesses
		config.Producer.MaxMessageBytes = userConfig.maxMessageBytes
		config.Producer.Partitioner = userConfig.partitioner
		config.Producer.Compression = userConfig.compression // 请酌情启用数据压缩。经过压缩的数据，在MQ的处理过程中吞吐能力更强，成本更优

		if timeout > 0 {
			config.Metadata.Timeout = timeout
		}
		userConfig.scramClient.config(config)
		p, e := sarama.NewAsyncProducer(userConfig.brokers, config)
		if e != nil {
			return nil, e
		}
		go ct.AsyncProduce(p)
		producer = &Producer{
			async:         true,
			asyncProducer: p,
			topic:         userConfig.topic,
		}
	} else {
		config := sarama.NewConfig()

		config.Version = userConfig.version
		config.ClientID = userConfig.clientid // 生产者ID，从管理平台中生成

		config.Producer.RequiredAcks = userConfig.requiredAcks
		config.Producer.Return.Successes = userConfig.returnSuccesses
		config.Producer.MaxMessageBytes = userConfig.maxMessageBytes
		config.Producer.Partitioner = userConfig.partitioner
		config.Producer.Compression = userConfig.compression // 请酌情启用数据压缩。经过压缩的数据，在MQ的处理过程中吞吐能力更强，成本更优

		if timeout > 0 {
			config.Net.DialTimeout = timeout
			config.Net.ReadTimeout = timeout
			config.Net.WriteTimeout = timeout
			config.Producer.Timeout = timeout
			config.Metadata.Timeout = timeout
		}
		userConfig.scramClient.config(config)
		p, e := sarama.NewSyncProducer(userConfig.brokers, config)
		if e != nil {
			return nil, e
		}
		producer = &Producer{
			syncProducer: p,
			topic:        userConfig.topic,
		}
	}

	ct.producers[address] = producer
	return producer, nil
}

// Producer kafka producer
type Producer struct {
	topic         string
	async         bool
	asyncProducer sarama.AsyncProducer
	syncProducer  sarama.SyncProducer
}

// GetAsyncProducer 获取异步生产者，同时启动异步协程处理生产数据和消息
func (ct *ClientTransport) GetAsyncProducer(address string) (*Producer, error) {
	ct.asyncProducersLock.RLock()
	producer, ok := ct.asyncProducers[address]
	ct.asyncProducersLock.RUnlock()
	if ok {
		return producer, nil
	}

	ct.asyncProducersLock.Lock()
	defer ct.asyncProducersLock.Unlock()

	producer, ok = ct.asyncProducers[address]
	if ok {
		return producer, nil
	}

	userConfig, err := parseAddress(address)
	if err != nil {
		return nil, err
	}

	config := sarama.NewConfig()

	config.Version = userConfig.version
	config.ClientID = userConfig.clientid

	config.Producer.Return.Successes = userConfig.returnSuccesses
	config.Producer.Partitioner = userConfig.partitioner
	userConfig.scramClient.config(config)

	p, err := sarama.NewAsyncProducer(userConfig.brokers, config)
	if err != nil {
		return nil, err
	}
	go ct.AsyncProduce(p)
	producer = &Producer{
		async:         true,
		asyncProducer: p,
		topic:         userConfig.topic,
	}
	ct.asyncProducers[address] = producer

	return producer, nil
}

// AsyncProduce 异步生产并处理捕获的消息
func (ct *ClientTransport) AsyncProduce(producer sarama.AsyncProducer) {
	for {
		select {
		case msg := <-producer.Errors():
			log.Errorf("asyncProduce failed. topic:%s, key:%s, value:%s. err:%v", msg.Msg.Topic, msg.Msg.Key,
				msg.Msg.Value, msg.Err)
		case <-producer.Successes():
		}
	}
}
