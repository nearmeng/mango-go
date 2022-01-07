// Package attalog atta远程日志 trpc-日志插件里面的远程输出插件
package attalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"git.code.oa.com/atta/attaapi-go"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/metrics"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	pluginName = "atta"
	pluginType = "log"

	attaAPIDefaultSize     = 3     // atta obj默认大小, 单条3k, 单协程，单obj性能约21w/s。3个obj基本能达到agent瓶颈
	channelDefaultCapacity = 10000 // 管道默认容量, 按1条log 1k，约10M

	defaultSendInternalMs = 1000 // 发送间隔
)

// defaultLevelMap 鹰眼告警日志级别映射
var defaultLevelMap = map[string]string{
	"debug": "1",
	"info":  "2",
	"warn":  "3",
	"error": "4",
	"fatal": "5",
}

func init() {
	log.RegisterWriter(pluginName, &AttaPlugin{})
}

// Config atta log配置
type Config struct {
	AttaID          string   `yaml:"atta_id"`
	AttaToken       string   `yaml:"atta_token"`
	Field           []string // atta表结构字段 业务申请attaid时 自己定义的字段，必须一致
	TimeKey         string   `yaml:"time_key"`
	LevelKey        string   `yaml:"level_key"`
	AttaWarning     bool     `yaml:"atta_warning"`
	NameKey         string   `yaml:"name_key"`
	CallerKey       string   `yaml:"caller_key"`
	StacktraceKey   string   `yaml:"stacktrace_key"`
	MessageKey      string   `yaml:"message_key"`
	AutoEscape      bool     `yaml:"auto_escape"`
	AttaobjSize     int      `yaml:"attaobj_size"`
	ChannelBlock    bool     `yaml:"channel_block"`
	ChannelCapacity int      `yaml:"channel_capacity"`
	EnableBatch     bool     `yaml:"enable_batch"`  // log是否缓存批量发送
	SendInternal    int      `yaml:"send_internal"` // 缓存批量发送间隔，单位ms
}

// AttaPlugin atta log trpc 插件实现
type AttaPlugin struct {
}

// Type atta log trpc插件类型
func (p *AttaPlugin) Type() string {
	return pluginType
}

// Setup atta实例初始化log output core
func (p *AttaPlugin) Setup(name string, configDec plugin.Decoder) error {
	// 配置解析, 配置错误依旧返回err, 外部依赖错误降级处理
	decoder, conf, cfg, err := getDecoderAndConf(configDec)
	if err != nil {
		return err
	}

	initSucc := true
	err = fixAttaLogConfig(cfg)
	if err != nil {
		initSucc = false
	}

	// 初始化 attalog
	attaLogger := &AttaLogger{
		Field:      cfg.Field,
		MessageKey: cfg.MessageKey,
		AllowBlock: cfg.ChannelBlock,
		LogChannel: make(chan *buffer.Buffer, cfg.ChannelCapacity),
		InitSucc:   initSucc,
	}

	err = startComsumer(cfg, attaLogger)
	if err != nil {
		attaLogger.InitSucc = false
	}

	encoder := getEncoder(cfg, conf)

	l := zap.NewAtomicLevelAt(log.Levels[conf.Level])
	c := zapcore.NewCore(
		encoder,
		zapcore.AddSync(attaLogger),
		l,
	)

	decoder.Core = c
	decoder.ZapLevel = l
	return nil
}

// startComsumer 启动消费者
func startComsumer(cfg *Config, attaLogger *AttaLogger) error {
	if !attaLogger.InitSucc {
		return fmt.Errorf("atta logger init error")
	}

	// 初始化消费者
	for i := 0; i < cfg.AttaobjSize; i++ {
		t := newAttaAPI()
		ret := t.InitUDP()
		if ret != 0 { // 初始化失败设置InitSucc=false并退出for循环
			log.Errorf("trpc-log-atta: atta index %d, InitUDP error ret:%d", i, ret)
			return fmt.Errorf("attaapi init fail")
		}

		consumer := &logConsumer{
			Atta:      t,
			AttaID:    cfg.AttaID,
			AttaToken: cfg.AttaToken,

			EnableBatch:    cfg.EnableBatch,
			SendInternalMs: cfg.SendInternal,
		}

		consumer.start(attaLogger.LogChannel)
	}

	return nil
}

// newAttaAPI 新建atta api
func newAttaAPI() *attaapi.AttaApi {
	return &attaapi.AttaApi{}
}

// getDecoderAndConf 解析日志配置
func getDecoderAndConf(configDec plugin.Decoder) (*log.Decoder, *log.OutputConfig, *Config, error) {
	if configDec == nil {
		return nil, nil, nil, errors.New("attalog writer decoder empty")
	}
	decoder, ok := configDec.(*log.Decoder)
	if !ok {
		return nil, nil, nil, errors.New("attalog writer log decoder type invalid")
	}

	conf := &log.OutputConfig{}
	err := decoder.Decode(&conf)
	if err != nil {
		return nil, nil, nil, err
	}

	var cfg Config
	err = conf.RemoteConfig.Decode(&cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return decoder, conf, &cfg, nil
}

// getEncoder 生成zap atta encoder
func getEncoder(cfg *Config, conf *log.OutputConfig) *attaEncoder {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        cfg.TimeKey,
		LevelKey:       cfg.LevelKey,
		NameKey:        cfg.NameKey,
		CallerKey:      cfg.CallerKey,
		MessageKey:     cfg.MessageKey,
		StacktraceKey:  cfg.StacktraceKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     log.NewTimeEncoder(conf.FormatConfig.TimeFmt),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	if cfg.AttaWarning {
		encoderCfg.EncodeLevel = levelMapFunc
	} else {
		encoderCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	}
	encoder := newAttaEncoder(encoderCfg, cfg.Field, cfg.MessageKey, cfg.AutoEscape)
	return encoder
}

// fixAttaLogConfig 修复atta log默认值
func fixAttaLogConfig(cfg *Config) error {
	var err error
	if cfg.AttaobjSize < 1 {
		cfg.AttaobjSize = attaAPIDefaultSize
	}

	if cfg.ChannelCapacity < 1 {
		cfg.ChannelCapacity = channelDefaultCapacity
	}

	if cfg.SendInternal < 1 {
		cfg.SendInternal = defaultSendInternalMs
	}

	// 支持远程拉取atta字段列表
	if len(cfg.Field) == 0 {
		cfg.Field, err = getRemoteAttaFields(cfg.AttaID)
		if err != nil {
			log.Errorf("trpc-log-atta:getRemoteAttaFields err:%v", err)
			return err
		}
		log.Debugf("trpc-log-atta: attaID:%v, remote fields:%+v", cfg.AttaID, cfg.Field)
	}

	return nil
}

// AttaLogger atta logger
type AttaLogger struct {
	MessageKey string
	Field      []string
	AllowBlock bool
	LogChannel chan *buffer.Buffer
	InitSucc   bool
}

// Write 写atta日志
func (l *AttaLogger) Write(p []byte) (n int, err error) {
	// 没有初始化成功直接返回
	if !l.InitSucc {
		return len(p), nil
	}

	dst := bufferpool.Get()
	dst.Write(p)

	// 写管道
	if l.AllowBlock {
		// 阻塞，不建议开启：本地日志有且level小于attlog或者有业务逻辑，消费速度基本足够，有丢失可考虑增加atta obj数量
		l.LogChannel <- dst
	} else {
		// 非阻塞，可能丢数据，避免消费性能不足影响业务逻辑
		select {
		case l.LogChannel <- dst:
		default:
			metrics.Counter("AttaLoggerPutChannelFail").Incr() // 减少插件压力,继续trpc log大概率也会丢
		}
	}

	return len(p), nil
}

// attaDataAPIResp atta dataapi返回
type attaDataAPIResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Fielddata []struct {
			SFieldName string `json:"sFieldName"`
		} `json:"fielddata"`
	} `json:"data"`
}

// getRemoteAttaFields 拉取atta字段列表
func getRemoteAttaFields(attaID string) ([]string, error) {
	url := fmt.Sprintf("http://atta.wsd.com/cgi/dataapi?interfaceId=98&"+
		"token=216-df322f0a-5596-4671-a46c-06342c32a84c&sAttaId=%s", attaID)

	client := http.Client{
		Timeout: 1 * time.Second,
	}
	rsp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	var resp attaDataAPIResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Data.Fielddata) <= 2 {
		return nil, fmt.Errorf("trpc-log-atta:atta dataapi resp fieldData(len:%d) invalid", len(resp.Data.Fielddata))
	}

	var fields []string
	for _, fieldData := range resp.Data.Fielddata[2:] {
		fields = append(fields, fieldData.SFieldName)
	}
	return fields, nil
}

// levelMapFunc 日志级别映射
func levelMapFunc(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	s := l.String()
	if t, ok := defaultLevelMap[s]; ok {
		enc.AppendString(t)
	} else {
		enc.AppendString(s)
	}
}
