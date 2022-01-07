package kafka

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
)

// UserConfig 从address中解析出的配置
type UserConfig struct {
	brokers                []string // 集群地址
	topics                 []string // 用于消费者
	topic                  string   // 用于生产者
	group                  string   // 消费者组
	async                  int      // 是否异步生产，0同步 1异步
	clientid               string   // 客户端ID
	compression            sarama.CompressionCodec
	version                sarama.KafkaVersion
	strategy               sarama.BalanceStrategy
	partitioner            func(topic string) sarama.Partitioner
	initial                int64 // 新消费者组第一次连到集群 消费的位置
	fetchDefault           int
	fetchMax               int
	maxWaitTime            time.Duration
	requiredAcks           sarama.RequiredAcks
	returnSuccesses        bool
	timeout                time.Duration
	maxMessageBytes        int
	batchConsumeCount      int           // 批量消费最大的消息数
	batchFlush             time.Duration // 批量消费生效
	scramClient            *LSCRAMClient // LSCRAM安全认证
	maxRetry               int           // 失败最大重试次数, 默认0: 一直重试
	netMaxOpenRequests     int           //最大请求数
	maxProcessingTime      time.Duration
	netDailTimeout         time.Duration
	netReadTimeout         time.Duration
	netWriteTimeout        time.Duration
	groupSessionTimeout    time.Duration
	groupRebalanceTimeout  time.Duration
	groupRebalanceRetryMax int
}

// parseAddress address格式 ip1:port1,ip2:port2?clientid=xx&topics=topic1,topic2&group=xxx&compression=gzip
func parseAddress(address string) (config *UserConfig, err error) {
	config = getDefaultConfig()

	tokens := strings.SplitN(address, "?", 2)
	if len(tokens) != 2 {
		return nil, fmt.Errorf("address format invalid: address: %v, tokens: %v", address, tokens)
	}
	config.brokers = strings.Split(tokens[0], ",")
	tokens = strings.Split(tokens[1], "&")
	if len(tokens) == 0 {
		return nil, fmt.Errorf("address format invalid: brokers: %v with empty params", config.brokers)
	}
	for _, val := range tokens {
		vals := strings.SplitN(val, "=", 2)
		if len(vals) != 2 {
			return nil, fmt.Errorf("address format invalid, key=value missing: %v", vals)
		}

		// 各配置字段值支持特殊字符
		vals[1], _ = url.QueryUnescape(vals[1])

		if err := getConfigFromToken(config, vals); err != nil {
			return nil, err
		}
	}
	return
}

func getConfigFromToken(config *UserConfig, vals []string) error {
	var err error
	switch vals[0] {
	case "clientid":
		config.clientid = vals[1]
	case "topics":
		config.topics = strings.Split(vals[1], ",")
	case "topic":
		config.topic = vals[1]
	case "group":
		config.group = vals[1]
	case "async":
		config.async, err = parseAsync(vals[1])
	case "compression":
		config.compression, err = parseCompression(vals[1])
	case "version":
		config.version, err = sarama.ParseKafkaVersion(vals[1])
	case "strategy":
		config.strategy, err = parseStrategy(vals[1])
	case "partitioner":
		config.partitioner, err = parsePartitioner(vals[1])
	case "fetchDefault":
		config.fetchDefault, err = strconv.Atoi(vals[1])
	case "fetchMax":
		config.fetchMax, err = strconv.Atoi(vals[1])
	case "maxMessageBytes":
		config.maxMessageBytes, err = strconv.Atoi(vals[1])
	case "initial":
		config.initial, err = parseInital(vals[1])
	case "maxWaitTime":
		config.maxWaitTime, err = parseDuration(vals[1])
	case "batch":
		config.batchConsumeCount, err = strconv.Atoi(vals[1])
	case "batchFlush":
		config.batchFlush, err = parseDuration(vals[1])
	case "requiredAcks":
		config.requiredAcks, err = parseRequireAcks(vals[1])
	case "maxRetry":
		config.maxRetry, err = strconv.Atoi(vals[1])
	case "netMaxOpenRequests":
		config.netMaxOpenRequests, err = strconv.Atoi(vals[1])
	case "maxProcessingTime":
		config.maxProcessingTime, err = parseDuration(vals[1])
	case "netDailTimeout":
		config.netDailTimeout, err = parseDuration(vals[1])
	case "netReadTimeout":
		config.netReadTimeout, err = parseDuration(vals[1])
	case "netWriteTimeout":
		config.netWriteTimeout, err = parseDuration(vals[1])
	case "groupSessionTimeout":
		config.groupSessionTimeout, err = parseDuration(vals[1])
	case "groupRebalanceTimeout":
		config.groupRebalanceTimeout, err = parseDuration(vals[1])
	case "groupRebalanceRetryMax":
		config.groupRebalanceRetryMax, err = strconv.Atoi(vals[1])
	case "user", "password", "mechanism":
		if config.scramClient == nil {
			config.scramClient = &LSCRAMClient{}
		}
		config.scramClient.Parse(vals)
	default:
		return fmt.Errorf("address format invalid, unknown keys: %v", vals[0])
	}
	if err != nil {
		return fmt.Errorf("address format invalid(%v) err:%v", vals[0], err)
	}
	return nil
}

func getDefaultConfig() *UserConfig {
	userConfig := &UserConfig{
		compression:            sarama.CompressionGZIP,       //CDMQ默认压缩
		version:                sarama.V1_1_1_0,              //CDMQ默认版本
		strategy:               sarama.BalanceStrategySticky, //CDMQ默认严格
		partitioner:            sarama.NewRandomPartitioner,  //CDMQ默认随机
		initial:                sarama.OffsetNewest,
		fetchDefault:           524288,      // 单次消费拉取请求中，单个分区最大返回消息大小。一次拉取请求可能返回多个分区的数据，这里限定单个分区的最大数据大小
		fetchMax:               1048576,     // 单次消费拉取请求中，单个分区最大返回消息大小。一次拉取请求可能返回多个分区的数据，这里限定单个分区的最大数据大小
		maxWaitTime:            time.Second, // 单次消费拉取请求最长等待时间。最长等待时间仅在没有最新数据时才会等待。此值应当设置较大点，减少空请求对服务端QPS的消耗。
		requiredAcks:           sarama.WaitForAll,
		returnSuccesses:        true,
		timeout:                time.Second, // 请求在服务端最长请求处理时间
		maxMessageBytes:        131072,      // CDMQ设置
		clientid:               "trpcgo",
		batchFlush:             time.Duration(2 * time.Second),
		scramClient:            nil,
		netMaxOpenRequests:     5,
		maxProcessingTime:      100 * time.Millisecond,
		netDailTimeout:         30 * time.Second,
		netReadTimeout:         30 * time.Second,
		netWriteTimeout:        30 * time.Second,
		groupSessionTimeout:    10 * time.Second,
		groupRebalanceTimeout:  60 * time.Second,
		groupRebalanceRetryMax: 4,
	}
	return userConfig
}

func parseAsync(val string) (int, error) {
	if val == "1" {
		return 1, nil
	}
	return 0, nil
}

func parseCompression(val string) (sarama.CompressionCodec, error) {
	switch val {
	case "none":
		return sarama.CompressionNone, nil
	case "gzip":
		return sarama.CompressionGZIP, nil
	case "snappy":
		return sarama.CompressionSnappy, nil
	case "lz4":
		return sarama.CompressionLZ4, nil
	case "zstd":
		return sarama.CompressionZSTD, nil
	default:
		return sarama.CompressionNone, errors.New("param not support")
	}
}

func parseStrategy(val string) (sarama.BalanceStrategy, error) {
	switch val {
	case "sticky":
		return sarama.BalanceStrategySticky, nil
	case "range":
		return sarama.BalanceStrategyRange, nil
	case "roundrobin":
		return sarama.BalanceStrategyRoundRobin, nil
	default:
		return nil, errors.New("param not support")
	}
}

func parsePartitioner(val string) (func(topic string) sarama.Partitioner, error) {
	switch val {
	case "random":
		return sarama.NewRandomPartitioner, nil
	case "roundrobin":
		return sarama.NewRoundRobinPartitioner, nil
	case "hash":
		return sarama.NewHashPartitioner, nil
	default:
		return nil, errors.New("param not support")
	}
}

func parseInital(val string) (int64, error) {
	switch val {
	case "newest":
		return sarama.OffsetNewest, nil
	case "oldest":
		return sarama.OffsetOldest, nil
	default:
		return 0, errors.New("param not support")
	}
}

func parseDuration(val string) (time.Duration, error) {
	maxWaitTime, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return time.Duration(maxWaitTime) * time.Millisecond, err
}

func parseRequireAcks(val string) (sarama.RequiredAcks, error) {
	ack, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	saramaAcks := sarama.RequiredAcks(ack)
	if saramaAcks != sarama.WaitForAll && saramaAcks != sarama.WaitForLocal && saramaAcks != sarama.NoResponse {
		return 0, fmt.Errorf("invalid requiredAcks: %s", val)
	}
	return saramaAcks, err
}
