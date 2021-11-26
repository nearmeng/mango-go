// Package redis db redis support plugin.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	redisApi "github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/db"
)

// DBName 名字.
const DBName = "redis"

func init() {
	plugin.RegisterPluginFactory(&factory{})
}

// factory tcaplus工厂.
type factory struct{}

// Name 插件名字.
func (f *factory) Name() string {
	return DBName
}

// Type DB插件类型.
func (f *factory) Type() string {
	return "db"
}

// Destory tcaplus插件Destory方法.
func (f *factory) Destroy(interface{}) error {
	return nil
}

// Setup redis插件Init方法.
func (f *factory) Setup(c map[string]interface{}) (interface{}, error) {
	var cfg dbCfg
	if err := mapstructure.Decode(c, &cfg); err != nil {
		return nil, err
	}

	ins, err := DBOpen(&cfg)
	if err != nil {
		return nil, err
	}

	// 后面或许需要连接多个同类型DB, 比如tcaplus1,, tcaplus2
	return ins, nil
}

// Reload reload方法.
func (f *factory) Reload(interface{}, map[string]interface{}) error {
	return nil
}

// dbCfg Redis配置.
type dbCfg struct {
	Addr        string `mapstructure:"addr"`
	PoolSize    int    `mapstructure:"poolsize"`
	ConnTimeout uint32 `mapstructure:"conntimeout"`
	Password    string `mapstructure:"password"`
}

// DB 实现IDatabase接口.
type DB struct {
	client *redisApi.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// 类型断言.
var (
	_ db.IDBSimpleExecutor = (*DB)(nil)
)

// Open 初始化一个tcaplusDB.
func (t *DB) Open(config interface{}) error {
	cfg, ok := config.(*dbCfg)
	if !ok {
		return fmt.Errorf("config type not TcaplusDBCfg")
	}

	if cfg.PoolSize == 0 {
		return errors.New("pool must > 0")
	}

	defaultTimeout := time.Second * time.Duration(cfg.ConnTimeout)
	opt := &redisApi.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,

		// TODO: change timeout
		DialTimeout:  defaultTimeout,
		ReadTimeout:  defaultTimeout,
		WriteTimeout: defaultTimeout,
		//nolint
		PoolTimeout: 30 * time.Second,
		IdleTimeout: time.Minute,

		MaxRetries: -1,
		PoolSize:   cfg.PoolSize,

		//nolint
		IdleCheckFrequency: 100 * time.Millisecond,
	}
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.client = redisApi.NewClient(opt)
	if t.client == nil {
		return errors.New("new redis client nil")
	}
	return nil
}

// DBOpen 传入参数，初始化一个tcaplusDB.
//  @param config
//  @return db.IDatabase 返回的是通用DB接口
//  @return error
func DBOpen(config *dbCfg) (db.IDatabase, error) {
	d := &DB{}

	if e := d.Open(config); e != nil {
		return nil, e
	}
	return d, nil
}

// NewRequest 获得一个自定义请求（高级用法，暂未实现）.
func (t *DB) NewRequest() db.IDBRequest {
	return nil
}

// Exec tcaplus不需要实现.
func (t *DB) Exec(_ ...interface{}) (db.IDBResult, error) {
	return nil, fmt.Errorf("not support customize request")
}

// ExecAsync tcaplus不需要实现.
func (t *DB) ExecAsync(ret func(db.IDBResult, error), _ ...interface{}) {
	ret(nil, fmt.Errorf("not support customize request"))
}
