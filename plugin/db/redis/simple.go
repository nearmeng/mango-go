package redis

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nearmeng/mango-go/plugin/db"
	"github.com/nearmeng/mango-go/plugin/db/pbsupport"
	"google.golang.org/protobuf/proto"
)

type Record = proto.Message

const (
	_redisOk = "OK"
)

// SimpleGet 获取数据，支持指定fields.
//  @param model 传入的数据模型，需要带上对应的key值，调用后model会自动带出拉取到的数据
//  @param fields 如果为nil，表示全量拉取
//  @return err
func (t *DB) SimpleGet(record Record, fields []string) error {
	//sw := metrics.StartStopwatchWithGroup("bingo.RedisCmd", "bingodb")
	//defer sw.RecordWithDim([]*metrics.Dimension{
	//{Name: "cmd", Value: "Get"},
	//})
	k := BuildKey(record)
	if len(fields) == 0 {
		ret, err := t.client.HGetAll(t.ctx, k).Result()
		if err != nil {
			return fmt.Errorf("redis ret err:%w", err)
		}
		if len(ret) == 0 {
			return db.ErrRecordNotExist
		}
		return pbsupport.UnmarshalFromMap(record, ret)
	}

	ret, err := t.client.HMGet(t.ctx, k, fields...).Result()
	if err != nil || len(ret) != len(fields) {
		return fmt.Errorf("redis ret err:%w", err)
	}
	m := map[string]string{}
	for i, f := range fields {
		s, ok := ret[i].(string)
		if !ok {
			return fmt.Errorf("result type field=%s", f)
		}
		m[f] = s
	}
	return pbsupport.UnmarshalFromMap(record, m)
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
func (t *DB) SimpleUpdate(record Record, fields []string) (err error) {
	//sw := metrics.StartStopwatchWithGroup("bingo.RedisCmd", "bingodb")
	//defer sw.RecordWithDim([]*metrics.Dimension{
	//{Name: "cmd", Value: "Update"},
	//})
	m, err := pbsupport.MarshalToMap(record, fields)
	if err != nil {
		return errors.New("marshal map failed")
	}

	const (
		script string = `
			local k = KEYS[1]
			local ret = redis.pcall('exists', k)
			if ret ~= 1 then
				return 'NotExist'
			end
			return redis.pcall('hmset',k,unpack(ARGV))
		`
	)
	args := []interface{}{}
	for k, v := range m {
		args = append(args, k, v)
	}
	k := BuildKey(record)
	ret, err := t.client.Eval(t.ctx, script, []string{k}, args...).Result()
	if err != nil {
		return fmt.Errorf("redis ret err:%w", err)
	}
	r, ok := ret.(string)
	if !ok {
		return fmt.Errorf("redis result not string")
	}
	if r != _redisOk {
		return fmt.Errorf("update record ret=%s", r)
	}
	return nil
}

// SimpleInsert 插入新数据，如果已存在会失败.
//  @param model
//  @return error
func (t *DB) SimpleInsert(record Record) error {
	//sw := metrics.StartStopwatchWithGroup("bingo.RedisCmd", "bingodb")
	//defer sw.RecordWithDim([]*metrics.Dimension{
	//{Name: "cmd", Value: "Insert"},
	//})
	m, err := pbsupport.MarshalToMap(record, nil)
	if err != nil {
		return errors.New("marshal map failed")
	}

	const (
		script string = `
			local k = KEYS[1]
			local ret = redis.pcall('exists', k)
			if ret == 1 then
				return 'AlreadyExists'
			end
			return redis.pcall('hmset',k,unpack(ARGV))
		`
	)
	insertargs := []interface{}{}
	for k, v := range m {
		insertargs = append(insertargs, k, v)
	}
	k := BuildKey(record)
	result, err := t.client.Eval(t.ctx, script, []string{k}, insertargs...).Result()
	if err != nil {
		return fmt.Errorf("redis ret err:%w", err)
	}
	r, ok := result.(string)
	if !ok {
		return fmt.Errorf("redis result not string")
	}
	if r != "OK" {
		return fmt.Errorf("insert record ret=%s", r)
	}
	return nil
}

// SimpleReplace 更新数据（如果没有就创建）.
//  @param model 传入的数据模型
//  @return err
func (t *DB) SimpleReplace(record Record) (err error) {
	//sw := metrics.StartStopwatchWithGroup("bingo.RedisCmd", "bingodb")
	//defer sw.RecordWithDim([]*metrics.Dimension{
	//{Name: "cmd", Value: "Replace"},
	//})
	m, err := pbsupport.MarshalToMap(record, nil)
	if err != nil {
		err = errors.New("marshal map failed")
		return
	}
	ret, err := t.client.HMSet(t.ctx, BuildKey(record), m).Result()
	if err != nil {
		err = fmt.Errorf("redis ret err:%w", err)
		return
	}
	if !ret {
		err = errors.New("redis ret false")
		return
	}
	return
}

// SimpleDelete 删除指定key的数据.
//  @param model 传入的数据模型，需要带上对应的key值
//  @param resultFlag 指定0表示不需要返回数据，3表示从model传出删除的数据
//  @return error
func (t *DB) SimpleDelete(record Record, resultFlag int) error {
	//sw := metrics.StartStopwatchWithGroup("bingo.RedisCmd", "bingodb")
	//defer sw.RecordWithDim([]*metrics.Dimension{
	//{Name: "cmd", Value: "Delete"},
	//})
	// nolint
	if resultFlag == 3 {
		t.SimpleGet(record, nil)
	}
	_, err := t.client.Del(t.ctx, BuildKey(record)).Result()
	if err != nil {
		return fmt.Errorf("redis ret err:%w", err)
	}
	return nil
}

// SimpleIncrease 自增指定整形字段.
//  @param model 传入的数据模型，会返回最新的值
//  @param fields 指定字段集合，需要在model中有赋值
//  @return err
func (t *DB) SimpleIncrease(record Record, fields []string) error {
	//sw := metrics.StartStopwatchWithGroup("bingo.RedisCmd", "bingodb")
	//defer sw.RecordWithDim([]*metrics.Dimension{
	//{Name: "cmd", Value: "Increase"},
	//})
	rf := record.ProtoReflect()
	meta := GetDBProtoMeta(rf.Descriptor())
	if !meta.IncreaseAble(fields) {
		return fmt.Errorf("cannot increase fields=%s", strings.Join(fields, ","))
	}
	if len(fields) == 1 {
		_, e1 := t.client.HIncrBy(t.ctx, BuildKey(record), fields[0], 1).Result()
		if e1 != nil {
			return fmt.Errorf("redis ret err:%w", e1)
		}
		return nil
	}

	pip := t.client.Pipeline()
	k := BuildKey(record)
	for _, f := range fields {
		pip.HIncrBy(t.ctx, k, f, 1)
	}
	_, err := pip.Exec(t.ctx)
	return err
}
