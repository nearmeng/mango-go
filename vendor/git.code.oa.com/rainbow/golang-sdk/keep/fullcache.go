package keep

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"git.code.oa.com/rainbow/golang-sdk/config"
	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/types"
)

// FullCache 全量缓存，多APP，多环境，多group
type FullCache struct {
	// [key=appid, value=*sync.Map[key=env name, value=*sync.Map[key=group name, value=*GroupWapper]]]
	cache sync.Map
	// 文件备份锁
	backupLock sync.Mutex
}

// FullEnv  存储环境下面的所有group
type FullEnv map[string]GroupWrapper

// FullApp 存储App的所有环境
type FullApp map[string]FullEnv

// Full 存储所有APP数据
type Full map[string]FullApp

// Get 从本地缓存取值
func (fc *FullCache) Get(key string, opts types.GetOptions) (string, error) {
	var kv string
	// 找到所属app
	iv, ok := fc.cache.Load(opts.AppID)
	// 不存在
	if !ok {
		return kv, types.ErrNoAnyVersion
	}
	app := iv.(*sync.Map)
	// 找到所属环境
	if iv, ok = app.Load(opts.EnvName); !ok {
		return kv, types.ErrNoAnyVersion
	}
	env := iv.(*sync.Map)

	// 找到所属group
	if iv, ok = env.Load(opts.Group); !ok {
		return kv, types.ErrNoAnyVersion
	}

	// 找到对应key
	group := iv.(*GroupWrapper)
	group.Mutex.RLock()
	defer group.Mutex.RUnlock()
	if iv, ok = group.Value[key]; !ok {
		return kv, types.ErrNoAnyVersion
	}
	kv = iv.(string)
	return kv, nil
}

// CoverGroup 覆盖更新
func (fc *FullCache) CoverGroup(opts types.GetOptions, items []*config.KeyValueItem) {
	var tryApp sync.Map
	var tryEnv sync.Map
	iv, _ := fc.cache.LoadOrStore(opts.AppID, &tryApp)
	appTrue := iv.(*sync.Map)
	iv, _ = appTrue.LoadOrStore(opts.EnvName, &tryEnv)
	envTrue := iv.(*sync.Map)

	var gw = GroupWrapper{Value: make(Group)}
	itemsLen := len(items)
	for i := 0; i < itemsLen; i++ {
		kvItem := items[i]
		if kvItem == nil {
			continue
		}
		// 表类型,全量数据才缓存
		if kvItem.StructType == 1 {
			if opts.Start != 0 {
				continue
			}
			if opts.Offset > 0 && !kvItem.RowsEnd {
				continue
			}
		}
		gw.Version = kvItem.GetVersion()
		gw.VersionID = kvItem.GetVersionId()
		gw.VersionName = kvItem.GetVersionName()
		gw.StructType = kvItem.GetStructType()

		// table 类型
		if gw.StructType == 1 {
			gw.Table = CopyKeyValueItem(kvItem)
			if kvItem.GetEventType() == config.EventTypeDelete {
				envTrue.Delete(opts.Group)
			}
			continue
		}
		kvs := kvItem.GetKeyValues()
		if kvs == nil {
			continue
		}
		for j := 0; j < len(kvs); j++ {
			gw.Value[kvs[j].GetKey()] = kvs[j].GetValue()
		}
	}
	iv, ok := envTrue.LoadOrStore(opts.Group, &gw)
	if ok {
		group := iv.(*GroupWrapper)
		group.Mutex.Lock()
		defer group.Mutex.Unlock()
		if group.VersionID >= gw.VersionID {
			return
		}
		group.Version = gw.Version
		group.VersionID = gw.VersionID
		group.VersionName = gw.VersionName
		group.Value = gw.Value
		group.StructType = gw.StructType
	}
}

// UpdateGroup 更新group
func (fc *FullCache) UpdateGroup(opts types.GetOptions, ut UpdateType, items []*config.KeyValueItem) {
	var tryApp sync.Map
	var tryEnv sync.Map
	iv, _ := fc.cache.LoadOrStore(opts.AppID, &tryApp)
	appTrue := iv.(*sync.Map)
	iv, _ = appTrue.LoadOrStore(opts.EnvName, &tryEnv)
	envTrue := iv.(*sync.Map)

	var gw = GroupWrapper{Value: make(Group)}
	iv, _ = envTrue.LoadOrStore(opts.Group, &gw)

	group := iv.(*GroupWrapper)
	group.Mutex.Lock()
	defer group.Mutex.Unlock()
	itemsLen := len(items)

	for i := 0; i < itemsLen; i++ {
		kvItem := items[i]
		if kvItem == nil {
			continue
		}
		/*if group.VersionID >= kvItem.GetVersionId() {
			return
		}*/
		// 表类型,全量数据才缓存
		if kvItem.StructType == 1 {
			if opts.Start != 0 {
				continue
			}
			if opts.Offset > 0 && !kvItem.RowsEnd {
				continue
			}
		}
		// 单key更新, 不更新版本号 (最后一个 item 才会更新 version)
		if i == itemsLen-1 && (ut == UpdateTypeGetGroup || ut == UpdateTypePollGroup) {
			group.Version = kvItem.GetVersion()
			group.VersionID = kvItem.GetVersionId()
			group.VersionName = kvItem.GetVersionName()
			group.StructType = kvItem.GetStructType()
		}

		if group.StructType == 1 {
			group.Table = operatorKeyValueItem(group.Table, kvItem)
			continue
		}

		et := kvItem.GetEventType()
		kvs := kvItem.GetKeyValues()
		if kvs == nil {
			continue
		}
		for j := 0; j < len(kvs); j++ {
			if et != config.EventTypeDelete {
				group.Value[kvs[j].GetKey()] = kvs[j].GetValue()
				continue
			}
			if et == config.EventTypeDelete {
				delete(group.Value, kvs[j].GetKey())
			}
		}
	}
}

// GetGroup 从本地缓存获取group
func (fc *FullCache) GetGroup(opts types.GetOptions) (GroupWrapper, error) {
	var gw GroupWrapper
	iv, ok := fc.cache.Load(opts.AppID)
	if !ok {
		return gw, fmt.Errorf("appID=%s app  not exists", opts.AppID)
	}
	app := iv.(*sync.Map)

	if iv, ok = app.Load(opts.EnvName); !ok {
		return gw, fmt.Errorf("appID=%s, env=%s, group=%s env not exists",
			opts.AppID, opts.EnvName, opts.Group)
	}
	envTrue := iv.(*sync.Map)
	if iv, ok = envTrue.Load(opts.Group); !ok {
		return gw, fmt.Errorf("appID=%s, env=%s, group=%s group not exists",
			opts.AppID, opts.EnvName, opts.Group)
	}

	group := iv.(*GroupWrapper)
	group.Mutex.RLock()
	gw = CopyGroupWithTable(group)
	// fmt.Printf("%#v\n", group)
	defer group.Mutex.RUnlock()
	if group.Version == "" {
		return gw, fmt.Errorf("GetGroup=> no group Version need to uptate")
	}
	return gw, nil
}

// Backup 备份
func (fc *FullCache) Backup(b string) error {
	backup := make(Full)
	fc.cache.Range(func(key, val interface{}) bool {
		k, _ := key.(string)
		a, _ := val.(*sync.Map)
		app := make(FullApp)
		a.Range(func(key, val interface{}) bool {
			k, _ := key.(string)
			ev, _ := val.(*sync.Map)
			env := make(FullEnv)

			ev.Range(func(key, val interface{}) bool {
				k, _ := key.(string)
				g, _ := val.(*GroupWrapper)
				g.Mutex.Lock()
				gw := CopyGroupWithTable(g)
				g.Mutex.Unlock()
				env[k] = gw
				return true
			})
			app[k] = env
			return true
		})
		backup[k] = app
		return true
	})
	data, err := json.Marshal(backup)
	if err != nil {
		return err
	}
	// fmt.Printf("\n[back json] %s\n\n", string(data))
	fc.backupLock.Lock()
	defer fc.backupLock.Unlock()
	return ioutil.WriteFile(b, data, 0644)
}

// LoadBackup 加载备份文件, 单group
func (fc *FullCache) LoadBackup(b string, opts types.GetOptions) (*GroupWrapper, error) {
	fc.backupLock.Lock()
	data, err := ioutil.ReadFile(b)
	fc.backupLock.Unlock()

	if err != nil {
		return nil, err
	}
	backup := make(Full)
	err = json.Unmarshal(data, &backup)
	if err != nil {
		log.Errorf("%v", err)
		return nil, err
	}
	// 找到App
	app, ok := backup[opts.AppID]
	if !ok {
		return nil, types.ErrNoAnyVersion
	}
	// 找到环境
	env, ok := app[opts.EnvName]
	if !ok {
		return nil, types.ErrNoAnyVersion
	}
	group, ok := env[opts.Group]
	if !ok {
		return nil, types.ErrNoAnyVersion
	}
	return &group, nil
}

// LoadGroup2Cache 将信息LoadOrStore to cache
func (fc *FullCache) LoadGroup2Cache(opts types.GetOptions, gw *GroupWrapper) error {
	var tryApp sync.Map
	var tryEnv sync.Map
	iv, _ := fc.cache.LoadOrStore(opts.AppID, &tryApp)
	app := iv.(*sync.Map)
	iv, _ = app.LoadOrStore(opts.EnvName, &tryEnv)
	env := iv.(*sync.Map)
	env.LoadOrStore(opts.Group, gw)
	return nil
}
