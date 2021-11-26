// Package db support interface.
package db

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

// Record db plugin support record type.
type Record = proto.Message

// RecordResultTable TODO: not implement.
type RecordResultTable struct {
	code    int32
	Records []Record
}

// IDatabase DB接口类.
type IDatabase interface {
	IDBSimpleExecutor
	IDBAsyncExecutor

	// Open 根据config初始化DB.
	//  @param config
	Open(config interface{}) error

	// NewRequest 获得一个请求.
	NewRequest() IDBRequest

	// OnTick 执行异步读取数据库回包的调用.
	OnTick()

	// Exec 执行特殊的DB操作.
	//  @param args 自定义参数
	//  @return IDBResult
	//  @return error
	Exec(args ...interface{}) (IDBResult, error)

	// ExecAsync 执行异步DB操作.
	// @param ret 返回回调
	ExecAsync(ret func(IDBResult, error), args ...interface{})
}

// IDBSimpleExecutor 实现了一套简易的调用接口，理论上可覆盖大部分场景
// 与IDBExecutor接口主要的区别是Simple系列接口会直接将DB返回的数据覆盖于传入的model
// 并且无法获取诸如错误码、matchcount之类的额外信息，不过此类额外信息也只有特殊逻辑才需要使用
// 如果有特殊需求需要使用IDBExecutor接口.
type IDBSimpleExecutor interface {

	// SimpleGet 获取数据，支持指定fields.
	//  @param model 传入的数据模型，需要带上对应的key值，调用后model会自动带出拉取到的数据
	//  @param fields 如果为nil，表示全量拉取
	//  @return err
	SimpleGet(record Record, fields []string) (err error)

	// SimpleBatchGet 批量获取数据.
	//  @param model 需要传入slice，获取的数据也自动赋值上去
	//  @return error
	SimpleBatchGet(record []Record) error

	// SimpleUpdate 更新数据，如果不存在会失败，支持指定fields.
	//  @param model 传入的数据模型
	//  @param fields 如果为nil，表示全量更新
	//  @return err
	SimpleUpdate(record Record, fields []string) (err error)

	// SimpleInsert 插入新数据，如果已存在会失败.
	//  @param model
	//  @return error
	SimpleInsert(record Record) error

	// SimpleReplace 更新数据（如果没有就创建）.
	//  @param model 传入的数据模型
	//  @return err
	SimpleReplace(record Record) (err error)

	// SimpleDelete 删除指定key的数据.
	//  @param model 传入的数据模型，需要带上对应的key值
	//  @param resultFlag 指定0表示不需要返回数据，3表示从model传出删除的数据
	//  @return error
	SimpleDelete(record Record, resultFlag int) error

	// SimpleIncrease 自增指定整形字段.
	//  @param model 传入的数据模型，会返回最新的值
	//  @param fields 指定字段集合，需要在model中有赋值
	//  @return err
	SimpleIncrease(record Record, fields []string) (err error)
}

type AsyncResult func([]Record, error)

// IDBAsyncExecutor AsyncExecutor 的异步版本.
type IDBAsyncExecutor interface {
	// AsyncGet 获取数据，支持指定fields.
	//  @param model 传入的数据模型，需要带上对应的key值，调用后model会自动带出拉取到的数据
	//  @param fields 如果为nil，表示全量拉取
	//  @return err
	AsyncGet(ret AsyncResult, record Record, fields []string) (err error)

	// AsyncBatchGet 批量获取数据.
	//  @param model 需要传入slice，获取的数据也自动赋值上去
	//  @return error
	AsyncBatchGet(ret AsyncResult, record []Record) error

	// AsyncUpdate 更新数据，如果不存在会失败，支持指定fields.
	//  @param model 传入的数据模型
	//  @param fields 如果为nil，表示全量更新
	//  @return err
	AsyncUpdate(ret AsyncResult, record Record, fields []string) (err error)

	// AsyncInsert 插入新数据，如果已存在会失败.
	//  @param model
	//  @return error
	AsyncInsert(ret AsyncResult, record Record) error

	// AsyncReplace 更新数据（如果没有就创建）.
	//  @param model 传入的数据模型
	//  @return err
	AsyncReplace(ret AsyncResult, record Record) (err error)

	// AsyncDelete 删除指定key的数据.
	//  @param model 传入的数据模型，需要带上对应的key值
	//  @param resultFlag 指定0表示不需要返回数据，3表示从model传出删除的数据
	//  @return error
	AsyncDelete(ret AsyncResult, record Record, resultFlag int) error

	// AsyncIncrease 自增指定整形字段.
	//  @param model 传入的数据模型，会返回最新的值
	//  @param fields 指定字段集合，需要在model中有赋值
	//  @return err
	AsyncIncrease(ret AsyncResult, record Record, fields []string) (err error)
}

// IDBRequest 通用DB请求描述，用于构造一个请求和相关操作
// 为了兼容其他DB设计的，目前暂时用不到这套接口.
type IDBRequest interface {
	// Table 指定操作的表名.
	//  @param t
	//  @return DBRequest
	Table(t string) IDBRequest

	// Model 指定数据模型.
	//  @param m
	//  @return IDBRequest
	Model(m interface{}) IDBRequest

	// Limit 某些操作需要用到的分页参数.
	//  @param l
	//  @return IDBRequest
	Limit(l int) IDBRequest

	// Offset 某些操作需要用到的分页参数.
	//  @param off
	//  @return IDBRequest
	Offset(off int) IDBRequest

	// Keys 指定key.
	//  @param keys
	//  @return IDBRequest
	Keys(keys []string) IDBRequest

	// Fields 指定请求要操作的域.
	//  @param f
	//  @return IDBRequest
	Fields(f []string) IDBRequest

	// Option 自定义选项，每种DB都不同.
	//  @param key
	//  @param value
	//  @return DBRequest
	Option(key string, value interface{}) IDBRequest

	// DBVersion 指定DB乐观锁version.
	//  @param version
	//  @return IDBRequest
	DBVersion(version int) IDBRequest

	// ResultFlag 设置回包标记，没有此功能的DB直接return.
	//  @param resultFlag
	//  @return IDBRequest
	ResultFlag(resultFlag int) IDBRequest
}

// IDBResult DB请求结果相关接口.
type IDBResult interface {

	// ResultCode 获取数据库返回的错误码.
	//  @return int
	ResultCode() int

	// FetchData 一条一条获取结果.
	//  @param m 传入数据模型接收数据
	//  @return index db数据索引号
	//  @return version db数据版本号
	//  @return error
	FetchData(m interface{}) (index, version int, err error)

	// RecordCount 获取结果总数.
	//  @return int
	RecordCount() int

	// MatchCount 获取partkey匹配的数量.
	//  @return int
	MatchCount() int
}

var ErrRecordNotExist = errors.New("RecordNotExistErr")
