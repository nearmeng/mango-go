package confapi

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/types"
	chttp "git.code.oa.com/rainbow/golang-sdk/v3/http"
	"git.code.oa.com/rainbow/golang-sdk/v3/keep"
	"git.code.oa.com/rainbow/golang-sdk/v3/trpc"
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	config "git.code.oa.com/rainbow/proto/api/configv3"
)

// ConfAPI 配置调用接口
type ConfAPI struct {
	opts     types.InitOptions
	cache    keep.FullCache
	handler  types.RequestV3 // 远程请求
	wfactory watch.WatcherFactory
	quit     bool // 退出
}

// 全局的一个实例
var (
	global *ConfAPI
	mu     sync.Mutex
)

// New new api 全局一个实例
func New(opts ...types.AssignInitOption) (*ConfAPI, error) {
	if global != nil {
		return global, nil
	}
	mu.Lock()
	defer mu.Unlock()

	if global == nil {
		api, err := NewAgain(opts...)
		if err != nil {
			return nil, err
		}
		global = api
	}
	return global, nil
}

// NewAgain 创建一个新的实例，不推荐使用，推荐用上面的New全局的一个实例
func NewAgain(opts ...types.AssignInitOption) (*ConfAPI, error) {
	api := &ConfAPI{
		opts: types.NewInitOptions(opts...),
	}
	err := api.initialize()
	if err != nil {
		return nil, err
	}
	// 预加载
	api.cache.SetOpts(api.opts)
	api.PreLoad()
	api.wfactory.ResultChan = make(chan *watch.BasicResult, 10)
	api.handlerNotifyResult()
	// api.Heartbeat(types.WithIP(types.GetLocalIPStr()))

	return api, err
}

// Get 获取某个配置项
func (c *ConfAPI) Get(key string, opts ...types.AssignGetOption) (string, error) {
	getOpts := types.NewGetOptions(opts...)
	c.fillDefaultArgs(&getOpts)

	// 内存缓存
	if c.opts.IsUsingLocalCache && !getOpts.Remote {
		val, err := c.cache.Get(key, getOpts)
		if err == nil {
			return val, nil
		}
		if c.wfactory.IsExistsWatcher(key, getOpts) {
			return getOpts.DefaultValue, err
		}
	}
	// 配置服务
	_, val, _, err := c.handlerPullConfig(getOpts, key)
	if err == nil {
		return val, nil
	}
	// fmt.Println(getOpts)
	return getOpts.DefaultValue, err
}

// GetAndWatch get并且watch 这个key
func (c *ConfAPI) GetAndWatch(key string, opts ...types.AssignGetOption) (string, error) {
	w := watch.Watcher{Key: key}
	err := c.AddWatcher(w, opts...)
	if err != nil && err != types.ErrExistWatch {
		return "", err
	}
	val, err := c.Get(key, opts...)
	return val, err
}

// GetRaw 直接从远端获取某个配置项，返回裸数据
func (c *ConfAPI) GetRaw(key string, opts ...types.AssignGetOption) (*config.Item, error) {
	getOpts := types.NewGetOptions(opts...)
	c.fillDefaultArgs(&getOpts)
	_, _, kv, err := c.handlerPullConfig(getOpts, key)
	if err == nil && len(kv) > 0 {
		return kv[0], nil
	}
	return nil, err
}

// GetNumber 获取某个配置项，并转成number
func (c *ConfAPI) GetNumber(key string, opts ...types.AssignGetOption) (int64, error) {
	val, err := c.Get(key, opts...)
	if err != nil {
		return int64(0), err
	}
	v, err := strconv.ParseInt(val, 10, 64)
	return v, err
}

// GetAny 获取某个配置项，并转成入参interface{}
func (c *ConfAPI) GetAny(key string, p interface{}, opts ...types.AssignGetOption) (interface{}, error) {
	val, err := c.Get(key, opts...)
	if err != nil {
		return p, err
	}
	decoder := json.NewDecoder(strings.NewReader(val))
	decoder.UseNumber()
	err = decoder.Decode(p)
	return p, err
}

// GetGroup 获取某个group配置项
func (c *ConfAPI) GetGroup(opts ...types.AssignGetOption) (keep.Group, error) {
	var g keep.Group
	getOpts := types.NewGetOptions(opts...)
	c.fillDefaultArgs(&getOpts)

	// 内存缓存
	if c.opts.IsUsingLocalCache && !getOpts.Remote {
		val, err := c.cache.GetGroup(getOpts)
		if err == nil {
			return val.Value, nil
		}
		if c.wfactory.IsExistsWatcher("", getOpts) {
			return val.Value, err
		}
	}
	// 配置服务
	group, _, _, err := c.handlerPullConfig(getOpts, "")
	if err == nil {
		return group, nil
	}
	return g, err
}

// GetGroupAndWatch get并且watch 这个group
func (c *ConfAPI) GetGroupAndWatch(opts ...types.AssignGetOption) (keep.Group, error) {
	var g keep.Group
	w := watch.Watcher{Key: ""}
	err := c.AddWatcher(w, opts...)
	if err != nil && err != types.ErrExistWatch {
		return g, err
	}
	val, err := c.GetGroup(opts...)
	return val, err
}

// GetGroupRaw 直接从远端获取某个group配置项, 返回裸数据
func (c *ConfAPI) GetGroupRaw(opts ...types.AssignGetOption) ([]*config.Item, error) {
	getOpts := types.NewGetOptions(opts...)
	c.fillDefaultArgs(&getOpts)

	// 配置服务
	_, _, kv, err := c.handlerPullConfig(getOpts, "")
	if err == nil {
		return kv, nil
	}
	return kv, err
}

func filterTable(table *config.TableList,
	opts types.GetOptions) (*config.TableList, error) {
	if table.GetRows() == nil {
		return table, nil
	}
	// table.RowsEnd = false
	total := int32(len(table.GetRows()))
	if total < opts.Start {
		table.Rows = table.Rows[0:0]
		return table, types.ErrInvalidArg
	}
	end := opts.Start + opts.Offset
	if end >= total || end <= 0 {
		end = total
		//table.RowsEnd = true
	}
	table.Rows = table.Rows[opts.Start:end]
	return table, nil
}

// GetTable 获取表数据
func (c *ConfAPI) GetTable(opts ...types.AssignGetOption) (*config.TableList, error) {
	getOpts := types.NewGetOptions(opts...)
	c.fillDefaultArgs(&getOpts)

	// 内存缓存
	if c.opts.IsUsingLocalCache && !getOpts.Remote {
		val, err := c.cache.GetGroup(getOpts)
		if err == nil {
			return filterTable(val.TBL, getOpts)
		}
		if c.wfactory.IsExistsWatcher("", getOpts) {
			return val.TBL, err
		}
	}
	// 配置服务
	_, _, tb, err := c.handlerPullConfig(getOpts, "")
	if err != nil {
		return nil, err
	}
	kvs := tb
	if len(kvs) > 0 {
		return kvs[0].GetTables(), nil
	}
	return nil, fmt.Errorf("empty ")
}

// GetTableAny 传入指定列，返回指定列[]
func (c *ConfAPI) GetTableAny(cols interface{}, opts ...types.AssignGetOption) (
	*config.TableList, []interface{}, error) {
	tkv, err := c.GetTable(opts...)
	if err != nil || tkv.GetRows() == nil {
		return nil, nil, err
	}
	result := make([]interface{}, 0, len(tkv.GetRows()))
	for _, v := range tkv.GetRows() {
		decoder := json.NewDecoder(strings.NewReader(v))
		decoder.UseNumber()
		err = decoder.Decode(cols)
		if err != nil {
			return nil, nil, err
		}
		ref := reflect.New(reflect.TypeOf(cols).Elem())
		val := reflect.ValueOf(cols).Elem()
		nval := ref.Elem()
		for i := 0; i < val.NumField(); i++ {
			nval.Field(i).Set(val.Field(i))
		}
		result = append(result, ref.Interface())
	}
	return tkv, result, nil
}

// GetFile 获取文件只有描述信息
func (c *ConfAPI) GetFile(opts ...types.AssignGetOption) (*config.FileDataList, error) {
	getOpts := types.NewGetOptions(opts...)
	c.fillDefaultArgs(&getOpts)

	// 内存缓存
	if c.opts.IsUsingLocalCache && !getOpts.Remote {
		val, err := c.cache.GetGroup(getOpts)
		if err == nil {
			return val.FDL, nil
		}
		if c.wfactory.IsExistsWatcher("", getOpts) {
			return val.FDL, err
		}
	}
	// 配置服务
	_, _, tb, err := c.handlerPullConfig(getOpts, "")
	if err != nil {
		return nil, err
	}
	kvs := tb
	if len(kvs) > 0 {
		return kvs[0].GetFiles(), nil
	}
	return nil, fmt.Errorf("empty ")
}

// GetFileData 获取文件数据
func (c *ConfAPI) GetFileData(opts ...types.AssignGetOption) (keep.FileMetaList, error) {
	getOpts := types.NewGetOptions(opts...)
	c.fillDefaultArgs(&getOpts)

	// 内存缓存
	if c.opts.IsUsingLocalCache && !getOpts.Remote {
		val, err := c.cache.GetGroup(getOpts)
		if err == nil {
			return val.FDML, nil
		}
		if c.wfactory.IsExistsWatcher("", getOpts) {
			return val.FDML, err
		}
	}
	// 配置服务
	_, _, tb, err := c.handlerPullConfig(getOpts, "")
	if err != nil {
		return nil, err
	}
	kvs := tb
	if len(kvs) > 0 {
		fmdl := c.cache.LoadFileMeta(getOpts, kvs[0].GetFiles())
		return fmdl, nil
	}
	return nil, fmt.Errorf("empty ")

}

func heuristicSleep(maxST, st time.Duration) time.Duration {
	st *= 2
	if st > maxST {
		st = maxST
	}
	return st
}

// AddRandomSleepWatcher  polling的时候随机sleep微视使用
func (c *ConfAPI) AddRandomSleepWatcher(w watch.Watcher, opts ...types.AssignGetOption) error {
	var ctx = context.Background()
	ctx, cancel := context.WithCancel(ctx)
	wp := watch.CancelWatcher{
		Watcher: w,
		Cancel:  cancel,
	}
	wp.Watcher.Reassign(opts...)
	c.fillDefaultArgs(&wp.Watcher.GetOptions)
	// 判断是否已经存在
	if ok := c.wfactory.IsExistsWatcher(w.Key, wp.Watcher.GetOptions); ok {
		return types.ErrExistWatch
	}
	// watch前拉取一次
	_, _, _, err := c.handlerPullConfig(wp.Watcher.GetOptions, w.Key)
	if err != nil {
		log.Warnf("handlerPullConfig failed %s", err.Error())
	}
	err = c.wfactory.AddCancelWatcher(&wp)
	if err != nil {
		return err
	}
	go func() {
		endLoop := false
		for !c.quit && !endLoop {
			select {
			case <-ctx.Done():
				endLoop = true
				return
			default:
				st := types.Int63n(int64(w.RandomSleepDur))
				time.Sleep(time.Duration(st))
				err := c.handleWatcher(&wp.Watcher)
				if err != nil {
					log.Warningf(" watch failed %s", err.Error())
				}
			}
		}
	}()
	return nil
}

// AddWatcher 添加watcher
func (c *ConfAPI) AddWatcher(w watch.Watcher, opts ...types.AssignGetOption) error {
	var ctx = context.Background()
	maxST := 60 * time.Second
	st := 10 * time.Millisecond
	ctx, cancel := context.WithCancel(ctx)
	wp := watch.CancelWatcher{
		Watcher: w,
		Cancel:  cancel,
	}
	wp.Watcher.Reassign(opts...)
	c.fillDefaultArgs(&wp.Watcher.GetOptions)
	// 判断是否已经存在
	if ok := c.wfactory.IsExistsWatcher(w.Key, wp.Watcher.GetOptions); ok {
		return types.ErrExistWatch
	}
	// watch前拉取一次
	_, _, _, err := c.handlerPullConfig(wp.Watcher.GetOptions, w.Key)
	if err != nil {
		log.Warnf("handlerPullConfig failed %s", err.Error())
	}
	err = c.wfactory.AddCancelWatcher(&wp)
	if err != nil {
		return err
	}
	go func() {
		endLoop := false
		for !c.quit && !endLoop {
			select {
			case <-ctx.Done():
				endLoop = true
				return
			default:
				err := c.handleWatcher(&wp.Watcher)
				if err != nil {
					log.Warningf(" watch failed %s", err.Error())
					time.Sleep(st)
					st = heuristicSleep(maxST, st)
				} else {
					st = 10 * time.Millisecond
				}
			}
		}
	}()
	return nil
}

// Heartbeat 心跳
/*func (c *ConfAPI) Heartbeat(opts ...types.AssignGetOption) {
	go func() {
		var ctx = context.Background()
		req := &config.ReqHeartbeat{
			TerminalId:   version.TerminalID,
			TerminalType: version.Header,
			VersionName:  version.Version,
		}
		opts := types.NewGetOptions(opts...)
		t := time.NewTicker(c.opts.Heartbeat)
		defer t.Stop()

		for !c.quit {
			select {
			case <-t.C:
				_, err := c.handler.Heartbeat(ctx, req, &opts)
				if err != nil {
					log.Errorf("%#v\n", err)
				}
			}
		}
	}()
}*/

// DelWatcher 删除watcher
func (c *ConfAPI) DelWatcher(w watch.Watcher) error {
	return c.wfactory.DelCancelWatcher(&w)
}

// initialize 初始化
func (c *ConfAPI) initialize() error {
	if c.opts.RemoteProto == "trpc" {
		c.handler = &trpc.Requestor{}
		return c.handler.Init(c.opts)
	}
	c.handler = &chttp.Requestor{}
	return c.handler.Init(c.opts)
}

// Quit  退出
func (c *ConfAPI) Quit() {
	c.quit = true
}
