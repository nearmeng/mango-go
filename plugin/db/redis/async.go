package redis

import "github.com/nearmeng/mango-go/plugin/db"

// OnTick 执行异步读取数据库回包的调用.
func (t *DB) OnTick() {
}

// AsyncBatchGet 批量获取数据
//  @param model 需要传入slice []proto.Message
//  @return error
func (t *DB) AsyncBatchGet(ret db.AsyncResult, record []Record) error {
	return nil
}

// AsyncGet 获取数据，支持指定fields.
//  @param model 传入的数据模型，需要带上对应的key值，调用后model会自动带出拉取到的数据
//  @param fields 如果为nil，表示全量拉取
//  @return err
func (t *DB) AsyncGet(ret db.AsyncResult, record Record, fields []string) error {
	return nil
}

// AsyncIncrease 自增指定整形字段.
//  @param model 传入的数据模型，会返回最新的值
//  @param fields 指定字段集合，需要在model中有赋值
//  @return err
func (t *DB) AsyncIncrease(ret db.AsyncResult, record Record, fields []string) error {
	return nil
}

// AsyncUpdate 更新数据，如果不存在会失败，支持指定fields.
//  @param model 传入的数据模型
//  @param fields 如果为nil，表示全量更新
//  @return err
func (t *DB) AsyncUpdate(ret db.AsyncResult, record Record, fields []string) error {
	return nil
}

// AsyncReplace 更新数据（如果没有就创建）.
//  @param model 传入的数据模型
//  @return err
func (t *DB) AsyncReplace(ret db.AsyncResult, record Record) error {
	return nil
}

// AsyncInsert 插入新数据，如果已存在会失败.
//  @param model
//  @return error
func (t *DB) AsyncInsert(ret db.AsyncResult, record Record) error {
	return nil
}

// AsyncDelete 删除指定key的数据.
//  @param model 传入的数据模型，需要带上对应的key值
//  @param dbVersion 乐观锁版本号，如果不需要乐观锁，填0就行
//  @param resultFlag 指定0表示不需要返回数据，3表示从model传出删除的数据
//  @return error
func (t *DB) AsyncDelete(ret db.AsyncResult, record Record, resultFlag int) error {
	return nil
}
