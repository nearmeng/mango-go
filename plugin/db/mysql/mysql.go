// Package mysql db mysql support plugin.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/db"
	"github.com/spf13/viper"
)

// DBName 名字.
const (
	DBName = "mysql"
)

func init() {
	plugin.RegisterPluginFactory(&factory{})
}

// factory tcaplus工厂.
type factory struct{}

// Type DB插件类型.
func (f *factory) Type() string {
	return "db"
}

// Name 插件名字.
func (f *factory) Name() string {
	return DBName
}

// Setup redis插件Init方法.
func (f *factory) Setup(v *viper.Viper) (interface{}, error) {
	var cfg dbCfg
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	_ = mysql.SetLogger(nil)

	ins, err := DBOpen(&cfg)
	if err != nil {
		return nil, err
	}

	// 后面或许需要连接多个同类型DB, 比如tcaplus1,, tcaplus2
	return ins, nil
}

// Destory tcaplus插件Destory方法.
func (f *factory) Destroy(interface{}) error {
	return nil
}

// Reload reload方法.
func (f *factory) Reload(interface{}, map[string]interface{}) error {
	return nil
}

func (f *factory) Mainloop(interface{}) {

}

// dbCfg Redis配置.
type dbCfg struct {
	DataSource  string `mapstructure:"datasource"`
	IdleConns   int    `mapstructure:"idleconns"`
	MaxLifeTime uint32 `mapstructure:"maxlifetime"`
}

// DB 实现IDatabase接口.
type DB struct {
	sql    *sql.DB
	ctx    context.Context
	cancel context.CancelFunc
}

// 类型断言.
var (
	_ db.IDBSimpleExecutor = (*DB)(nil)
)

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

// Open 初始化一个tcaplusDB.
func (t *DB) Open(config interface{}) error {
	cfg, ok := config.(*dbCfg)
	if !ok {
		return fmt.Errorf("config type not TcaplusDBCfg")
	}

	dbsql, err := sql.Open("mysql", cfg.DataSource)
	if err != nil {
		return fmt.Errorf("open mysql datasource=%s, err:%w", cfg.DataSource, err)
	}
	dbsql.SetMaxIdleConns(cfg.IdleConns)
	dbsql.SetConnMaxLifetime(time.Duration(cfg.MaxLifeTime) * time.Second)

	if err = dbsql.Ping(); err != nil {
		return fmt.Errorf("failed to ping mysql datasource=%s", cfg.DataSource)
	}
	t.sql = dbsql
	t.ctx, t.cancel = context.WithCancel(context.Background())
	return nil
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
