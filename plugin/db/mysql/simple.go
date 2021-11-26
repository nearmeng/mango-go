package mysql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/nearmeng/mango-go/plugin/db"
	"github.com/nearmeng/mango-go/plugin/db/pbsupport"
	"google.golang.org/protobuf/proto"
)

type Record = proto.Message

// SimpleGet 获取数据，支持指定fields.
//  @param model 传入的数据模型，需要带上对应的key值，调用后model会自动带出拉取到的数据
//  @param fields 如果为nil，表示全量拉取
//  @return err
func (t *DB) SimpleGet(record Record, fields []string) error {
	rf := record.ProtoReflect()
	meta := GetDBProtoMeta(rf.Descriptor())
	if meta == nil {
		return fmt.Errorf("get meta nil fullname=%s", rf.Descriptor().FullName())
	}
	sqlPkg := meta.SelectFieldsPkg(rf, fields)
	rows, err := t.sql.QueryContext(t.ctx, sqlPkg.str, sqlPkg.params...)
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// 没有数据则返回失败
	if !rows.Next() {
		return db.ErrRecordNotExist
	}

	values := make([]sql.NullString, len(cols))
	args := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		args[i] = &values[i]
	}
	if e := rows.Scan(args...); e != nil {
		return fmt.Errorf("result scan err:%w", e)
	}
	retMap := map[string]string{}
	for i := 0; i < len(cols); i++ {
		retMap[cols[i]] = values[i].String
	}
	return pbsupport.UnmarshalFromMap(record, retMap)
}

// SimpleBatchGet 批量获取数据
//  @param model 需要传入slice []proto.Message
//  @return error
// limited support.
func (t *DB) SimpleBatchGet(record []Record) error {
	return nil
}

// SimpleUpdate 更新数据，如果不存在会失败，支持指定fields.
//  @param model 传入的数据模型
//  @param fields 如果为nil，表示全量更新
//  @return err
func (t *DB) SimpleUpdate(record Record, fields []string) error {
	rf := record.ProtoReflect()
	m, err := pbsupport.MarshalToMap(record, fields)
	if err != nil {
		return errors.New("marshal map failed")
	}
	meta := GetDBProtoMeta(rf.Descriptor())
	if meta == nil {
		return fmt.Errorf("get meta nil fullname:%s", rf.Descriptor().FullName())
	}
	sqlPkg := meta.UpdatePkg(rf, m)
	if sqlPkg == nil {
		return fmt.Errorf("build sql fullname:%s", rf.Descriptor().FullName())
	}
	r, e := t.sql.ExecContext(t.ctx, sqlPkg.str, sqlPkg.params...)
	if e != nil {
		return fmt.Errorf("mysql exec err:%w", e)
	}
	n, e := r.RowsAffected()
	if e != nil {
		return fmt.Errorf("mysql row affect err:%w", e)
	}
	if n == 0 {
		return errors.New("affect 0")
	}
	return nil
}

// SimpleInsert 插入新数据，如果已存在会失败.
//  @param model
//  @return error
func (t *DB) SimpleInsert(record Record) error {
	rf := record.ProtoReflect()
	m, err := pbsupport.MarshalToMap(record, nil)
	if err != nil {
		return errors.New("marshal map failed")
	}
	meta := GetDBProtoMeta(rf.Descriptor())
	if meta == nil {
		return fmt.Errorf("get meta nil fullname:%s", rf.Descriptor().FullName())
	}
	sqlPkg := meta.InsertPkg(m)
	if sqlPkg == nil {
		return fmt.Errorf("build sql fullname:%s", rf.Descriptor().FullName())
	}
	_, e := t.sql.ExecContext(t.ctx, sqlPkg.str, sqlPkg.params...)
	if e != nil {
		return fmt.Errorf("mysql exec err:%w", e)
	}
	return nil
}

// SimpleReplace 更新数据（如果没有就创建）.
//  @param model 传入的数据模型
//  @return err
func (t *DB) SimpleReplace(record Record) error {
	rf := record.ProtoReflect()
	m, err := pbsupport.MarshalToMap(record, nil)
	if err != nil {
		return errors.New("marshal map failed")
	}
	meta := GetDBProtoMeta(rf.Descriptor())
	if meta == nil {
		return fmt.Errorf("get meta nil fullname:%s", rf.Descriptor().FullName())
	}
	sqlPkg := meta.ReplacePkg(m)
	_, e := t.sql.ExecContext(t.ctx, sqlPkg.str, sqlPkg.params...)
	if e != nil {
		return fmt.Errorf("mysql exec err:%w", e)
	}
	return nil
}

// SimpleDelete 删除指定key的数据.
//  @param model 传入的数据模型，需要带上对应的key值
//  @param resultFlag 指定0表示不需要返回数据，3表示从model传出删除的数据
//  @return error
func (t *DB) SimpleDelete(record Record, resultFlag int) error {
	// nolint
	if resultFlag == 3 {
		t.SimpleGet(record, nil)
	}
	rf := record.ProtoReflect()
	meta := GetDBProtoMeta(rf.Descriptor())
	sqlPkg := meta.DeleteSQLPkg(rf)

	_, e := t.sql.ExecContext(t.ctx, sqlPkg.str, sqlPkg.params...)
	if e != nil {
		return fmt.Errorf("mysql exec err:%w", e)
	}
	return nil
}

// SimpleIncrease 自增指定整形字段.
//  @param model 传入的数据模型，会返回最新的值
//  @param fields 指定字段集合，需要在model中有赋值
//  @return err
func (t *DB) SimpleIncrease(record Record, fields []string) error {
	rf := record.ProtoReflect()
	meta := GetDBProtoMeta(rf.Descriptor())
	if meta == nil {
		return fmt.Errorf("get meta nil fullname:%s", rf.Descriptor().FullName())
	}
	sqlPkg := meta.IncreasePkg(rf, fields)
	if sqlPkg == nil {
		return fmt.Errorf("increase sql build fullname:%s", rf.Descriptor().FullName())
	}
	_, e := t.sql.ExecContext(t.ctx, sqlPkg.str, sqlPkg.params...)
	if e != nil {
		return fmt.Errorf("mysql exec err:%w", e)
	}
	return nil
}
