// Package bingologger bingo default logger.
/*
1. Supports synchronous write back and asynchronous write back
2. Support for splitting across days and according to file size
*/
package bingologger

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	_ = os.RemoveAll("./logs")
	_ = os.MkdirAll("./logs", os.ModePerm)
	r := m.Run()
	_ = os.RemoveAll("./logs")
	os.Exit(r)
}

func Test_factory_Type(t *testing.T) {
	tests := []struct {
		name string
		f    *factory
		want string
	}{
		{"success", &factory{}, "log"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &factory{}
			assert.Equal(t, tt.want, f.Type())
		})
	}
}

func Test_factory_Name(t *testing.T) {
	tests := []struct {
		name string
		f    *factory
		want string
	}{
		{"success", &factory{}, "bingologger"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &factory{}
			assert.Equal(t, tt.want, f.Name())
		})
	}
}

type logConfig struct {
	Log map[string]interface{} `toml:"Log"`
}

var asyncCfgStr = `
		[log]
		tag="default"
		path="./logs/testlog.log"
		level=1
		filesplitmb=1000
		isasync=true
		asynccachesize=10000
		asyncwritemillsec=10
		`

var syncCfgStr = `
		[log]
		tag="default"
		path="./logs/testlog.log"
		level=1
		filesplitmb=1000
		`

func Test_factory_Setup(t *testing.T) {
	type args struct {
		c map[string]interface{}
	}

	t.Run("invalid", func(t *testing.T) {
		cfgStr := `
		[log]
		tag="default"
		path="./logs/testlog.log"
		level=1
		`

		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(cfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		/*
			f := &factory{}
			got, err := f.Setup(cfg.Log)
			assert.Error(t, err)
			assert.Nil(t, got)
		*/
	})

	t.Run("success", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		/*
			f := &factory{}
			got, err := f.Setup(cfg.Log)
			assert.NoError(t, err)
			assert.NotNil(t, got)
		*/
	})
}

func Test_factory_Destory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		/*
				f := &factory{}
				got, err := f.Setup(cfg.Log)
				assert.NoError(t, err)
				assert.NotNil(t, got)

			// destory
			assert.NoError(t, f.Destroy(got))
		*/
	})
}

func Test_factory_Reload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		/*
			f := &factory{}
			got, err := f.Setup(cfg.Log)
			assert.NoError(t, err)
			assert.NotNil(t, got)

			assert.NoError(t, f.Reload(got, cfg.Log))
		*/
	})
}

func TestNewBingoLogger(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		var cfg2 LogCfg
		assert.NoError(t, mapstructure.Decode(cfg.Log, &cfg2))

		got, err := NewBingoLogger(cfg2)
		assert.NoError(t, err)
		assert.NotNil(t, got)
	})
}

func TestBingoLogger_Output(t *testing.T) {
	t.Run("success1", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		var cfg2 LogCfg
		assert.NoError(t, mapstructure.Decode(cfg.Log, &cfg2))

		got, err := NewBingoLogger(cfg2)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		got.Output(2, "xxx", nil, "ddd=%d", 1)
	})

	t.Run("success2", func(t *testing.T) {
		logger := &BingoLogger{}
		logger.Output(2, "xxx", nil, "ddd=%d", 1)
	})
}

func TestBingoLogger_Write(t *testing.T) {
	t.Run("success1", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		var cfg2 LogCfg
		assert.NoError(t, mapstructure.Decode(cfg.Log, &cfg2))

		got, err := NewBingoLogger(cfg2)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		content := "xxxxxxxxxxxxx"
		n, err := got.Write([]byte(content))
		assert.Len(t, content, n)
		assert.NoError(t, err)
	})

	t.Run("success2", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(syncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		var cfg2 LogCfg
		assert.NoError(t, mapstructure.Decode(cfg.Log, &cfg2))

		got, err := NewBingoLogger(cfg2)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		content := "xxxxxxxxxxxxxxxx"
		n, err := got.Write([]byte(content))
		assert.Len(t, content, n)
		assert.NoError(t, err)
	})
}

func TestBingoLogger_Sync(t *testing.T) {
	t.Run("success1", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(asyncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		var cfg2 LogCfg
		assert.NoError(t, mapstructure.Decode(cfg.Log, &cfg2))

		got, err := NewBingoLogger(cfg2)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		assert.NoError(t, got.Sync())
	})

	t.Run("success2", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(syncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		var cfg2 LogCfg
		assert.NoError(t, mapstructure.Decode(cfg.Log, &cfg2))

		got, err := NewBingoLogger(cfg2)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.NoError(t, got.Sync())
	})
}

func TestBingoLogger_updateOldFileFd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		v := viper.New()
		var cfg logConfig
		v.SetConfigType("toml")
		assert.NoError(t, v.ReadConfig(bytes.NewBuffer([]byte(syncCfgStr))))
		assert.NoError(t, v.Unmarshal(&cfg))

		var cfg2 LogCfg
		assert.NoError(t, mapstructure.Decode(cfg.Log, &cfg2))

		got, err := NewBingoLogger(cfg2)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		assert.NoError(t, got.updateFileFd())

		got.fileName = fmt.Sprintf("./logs/notexistfile_%d.log", time.Now().UnixNano())
		assert.NoError(t, got.updateOldFileFd())

		assert.NoError(t, got.updateFileFd())

		got.fileCreateTime = got.fileCreateTime.Add(time.Hour * 25)
		assert.NoError(t, got.updateOldFileFd())

		got.fileSplitMB = 0
		assert.NoError(t, got.updateFileFd())
		assert.NoError(t, got.updateOldFileFd())
	})
}

func TestBingoLogger_openLogFile(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name      string
		args      args
		assertion assert.ErrorAssertionFunc
	}{
		{"success", args{"./logs/testfile.log"}, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &BingoLogger{}
			l.fileName = tt.args.filePath
			tt.assertion(t, l.openLogFile(tt.args.filePath))
		})
	}
}

func TestBingoLogger_moveLogFile(t *testing.T) {
	filePath := "./logs/testfile.log"
	t.Run("success", func(t *testing.T) {
		l := &BingoLogger{}
		l.fileName = filePath
		_, _ = os.Create("./logs/testfile.log")
		assert.NoError(t, l.moveLogFile(filePath, time.Now()))
	})

	t.Run("filenotfound", func(t *testing.T) {
		l := &BingoLogger{}
		l.fileName = filePath
		assert.Error(t, l.moveLogFile(filePath, time.Now()))
	})
}

func TestBingoLogger_isLogFileExist(t *testing.T) {
	type args struct {
		filePath string
	}

	_, _ = os.Create("./logs/testfile.log")

	tests := []struct {
		name      string
		args      args
		want      bool
		assertion assert.ErrorAssertionFunc
	}{
		{"exist", args{"./logs/testfile.log"}, true, assert.NoError},
		{"notexist", args{fmt.Sprintf("./logs/testfile_%d.log", time.Now().UnixNano())}, false, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &BingoLogger{}
			got, err := l.isLogFileExist(tt.args.filePath)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
