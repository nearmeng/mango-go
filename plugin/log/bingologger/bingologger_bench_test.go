package bingologger

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// BenchmarkOutputAsync async log.
// nolint
func BenchmarkOutputAsync(b *testing.B) {
	// create logger
	v := viper.New()
	var cfg logConfig
	v.SetConfigType("toml")
	_ = v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr)))
	_ = v.Unmarshal(&cfg)
	var cfg2 LogCfg
	_ = mapstructure.Decode(cfg.Log, &cfg2)
	logger, _ := NewBingoLogger(cfg2)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := "123vdsgsfdsfsf"
		uid := i
		err1 := fmt.Errorf("bingoLoggerComplex uid=%d", uid)
		err2 := fmt.Errorf("bingoLoggerComplex msgid=%s err:%w", "xxxxxxxxxxx.req", err1)
		logger.Output(1, "Info", nil, "log 初始化成功 uid=%d msgid=%s err:%s", uid, s, err2.Error())
	}
}

// BenchmarkOutputAsync sync log.
// nolint
func BenchmarkOutputSync(b *testing.B) {
	// create logger
	v := viper.New()
	var cfg logConfig
	v.SetConfigType("toml")
	v.ReadConfig(bytes.NewBuffer([]byte(syncCfgStr)))
	v.Unmarshal(&cfg)
	var cfg2 LogCfg
	mapstructure.Decode(cfg.Log, &cfg2)
	logger, _ := NewBingoLogger(cfg2)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := "123vdsgsfdsfsf"
		uid := i
		err1 := fmt.Errorf("bingoLoggerComplex uid=%d", uid)
		err2 := fmt.Errorf("bingoLoggerComplex msgid=%s err:%w", "xxxxxxxxxxx.req", err1)
		logger.Output(1, "Info", nil, "log 初始化成功 uid=%d msgid=%s err:%s", uid, s, err2.Error())
	}
}
