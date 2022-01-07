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

// AllCache 本地缓存
type AllCache struct {
	// [key=appid, value=*sync.Map[key=group name, value=*GroupWapper]]
	cache sync.Map
	// 文件备份锁
	backupLock sync.Mutex
}

// Group 存储一个group数据
type Group map[string]interface{}

// App 存储一个app数据,一个app多个group
type App map[string]GroupWrapper

// All 存储所有APP数据
type All map[string]App

// GroupWrapper group 包装一层
type GroupWrapper struct {
	Value       Group                `json:"value,omitempty"`
	Version     string               `json:"version,omitempty"`
	Mutex       sync.RWMutex         `json:"-"`
	VersionID   int64                `json:"versionId,omitempty"`
	VersionName string               `json:"versionName,omitempty"`
	StructType  int32                `json:"struct_type,omitempty"`
	Table       *config.KeyValueItem `json:"table,omitempty"`
}

// UpdateType 更新类型
type UpdateType int32

const (
	// UpdateTypeGetKey get单key触发更新
	UpdateTypeGetKey UpdateType = 0
	// UpdateTypeGetGroup get group 触发更新
	UpdateTypeGetGroup UpdateType = 1
	// UpdateTypePollKey poll单key触发update
	UpdateTypePollKey UpdateType = 2
	// UpdateTypePollGroup poll group
	UpdateTypePollGroup UpdateType = 3
)

// Get 从本地缓存获取值
func (ac *AllCache) Get(key string, opts types.GetOptions) (string, error) {
	var kv string
	// 找到所属app
	iv, ok := ac.cache.Load(opts.AppID)
	// 不存在
	if !ok {
		return kv, types.ErrNoAnyVersion
	}
	app := iv.(*sync.Map)
	// 找到所属group
	if iv, ok = app.Load(opts.Group); !ok {
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

// CoverGroup  覆盖更新
func (ac *AllCache) CoverGroup(opts types.GetOptions, items []*config.KeyValueItem) {
	var tryApp sync.Map
	iv, _ := ac.cache.LoadOrStore(opts.AppID, &tryApp)
	app := iv.(*sync.Map)
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
				app.Delete(opts.Group)
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
	iv, ok := app.LoadOrStore(opts.Group, &gw)
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
func (ac *AllCache) UpdateGroup(opts types.GetOptions, ut UpdateType, items []*config.KeyValueItem) {
	var tryApp sync.Map
	iv, _ := ac.cache.LoadOrStore(opts.AppID, &tryApp)
	app := iv.(*sync.Map)
	var gw = GroupWrapper{Value: make(Group)}

	iv, _ = app.LoadOrStore(opts.Group, &gw)
	group := iv.(*GroupWrapper)
	group.Mutex.Lock()
	defer group.Mutex.Unlock()
	itemsLen := len(items)

	for i := 0; i < itemsLen; i++ {
		kvItem := items[i]
		if kvItem == nil {
			continue
		}
		if group.VersionID >= kvItem.GetVersionId() {
			return
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
		// 单key更新, 不更新版本号(最后一个 item 才会更新 version)
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

// CopyGroup 深度copy group
func CopyGroup(src Group) Group {
	dst := make(Group)
	for k, v := range src {
		switch v.(type) {
		case *config.KeyValueItem:
			kvi, _ := v.(*config.KeyValueItem)
			dst[k] = CopyKeyValueItem(kvi)
		default:
			dst[k] = v
		}
	}
	return dst
}

// CopyGroupWithTable  copy group with table
func CopyGroupWithTable(src *GroupWrapper) GroupWrapper {
	var gw GroupWrapper
	gw = *src
	gw.Value = CopyGroup(src.Value)
	gw.Table = CopyKeyValueItem(src.Table)
	return gw
}

func operatorKeyValueItem(origin, src *config.KeyValueItem) *config.KeyValueItem {
	if src.EventType == config.EventTypeAll {
		return CopyKeyValueItem(src)
	}

	if origin == nil {
		origin = &config.KeyValueItem{}
	}
	if origin.Rows == nil {
		origin.Rows = make([]string, 0)
	}
	if src == nil || src.Rows == nil {
		return origin
	}

	if src.EventType == config.EventTypeAdd {
		origin.Rows = append(origin.Rows, src.Rows...)
		return origin
	}

	type ColumnData struct {
		AutoID string `json:"_auto_id, omitempty"`
	}

	columns := make(map[string]string)
	var col ColumnData
	for _, v := range src.Rows {
		err := json.Unmarshal([]byte(v), &col)
		if err == nil {
			columns[col.AutoID] = v
		}
	}

	dst := make([]string, 0)
	if src.EventType == config.EventTypeDelete {
		for _, v := range origin.Rows {
			err := json.Unmarshal([]byte(v), &col)
			if err != nil {
				continue
			}

			if _, ok := columns[col.AutoID]; ok {
				continue
			}
			dst = append(dst, v)
		}
		origin.Rows = dst
	}

	if src.EventType == config.EventTypeUpdate {
		for i, v := range origin.Rows {
			err := json.Unmarshal([]byte(v), &col)
			if err != nil {
				continue
			}

			if cv, ok := columns[col.AutoID]; ok {
				origin.Rows[i] = cv
			}
		}
	}
	return origin
}

// CopyKeyValueItem  KeyValueItem
func CopyKeyValueItem(src *config.KeyValueItem) *config.KeyValueItem {
	if src == nil {
		return nil
	}
	dst := &config.KeyValueItem{
		Group:       src.Group,
		Version:     src.Version,
		EventType:   src.EventType,
		VersionId:   src.VersionId,
		VersionName: src.VersionName,
		StructType:  src.StructType,
		RowsEnd:     src.RowsEnd,
	}
	if src.KeyValues != nil && len(src.KeyValues) >= 0 {
		dst.KeyValues = make([]*config.KeyValue, len(src.KeyValues))
		for i, v := range src.KeyValues {
			dst.KeyValues[i] = &config.KeyValue{Key: v.Key, Value: v.Value}
		}
	}
	if src.Rows != nil && len(src.Rows) >= 0 {
		dst.Rows = make([]string, len(src.Rows))
		for i, v := range src.Rows {
			dst.Rows[i] = v
		}
	}
	if src.ColumnTypes != nil && len(src.ColumnTypes) >= 0 {
		dst.ColumnTypes = make([]*config.TableColumnType, len(src.ColumnTypes))
		for i, v := range src.ColumnTypes {
			dst.ColumnTypes[i] = &config.TableColumnType{Column: v.Column, DataType: v.DataType}
		}
	}

	return dst
}

// GetGroup 从本地缓存获取group
func (ac *AllCache) GetGroup(opts types.GetOptions) (GroupWrapper, error) {
	var gw GroupWrapper
	iv, ok := ac.cache.Load(opts.AppID)
	if !ok {
		return gw, fmt.Errorf("appID=%s app  not exists", opts.AppID)
	}
	app := iv.(*sync.Map)
	if iv, ok = app.Load(opts.Group); !ok {
		return gw, fmt.Errorf("appID=%s, group=%s group not exists", opts.AppID, opts.Group)
	}
	group := iv.(*GroupWrapper)
	group.Mutex.RLock()
	gw = CopyGroupWithTable(group)
	defer group.Mutex.RUnlock()
	if group.Version == "" {
		return gw, fmt.Errorf("GetGroup=> no group Version need to update")
	}

	return gw, nil
}

// FilterValue  event 判断
func FilterValue(kv config.KeyValue) error {
	return nil
}

// Backup 文件备份
func (ac *AllCache) Backup(b string) error {
	backup := make(All)
	ac.cache.Range(func(key, val interface{}) bool {
		k, _ := key.(string)
		a, _ := val.(*sync.Map)

		app := make(App)
		a.Range(func(key, val interface{}) bool {
			k, _ := key.(string)
			g, _ := val.(*GroupWrapper)
			app[k] = (*g)
			// fmt.Printf("\n[Backup] %#v\n", *g)
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
	ac.backupLock.Lock()
	defer ac.backupLock.Unlock()
	return ioutil.WriteFile(b, data, 0644)
}

// LoadBackup 加载备份文件, 单group
func (ac *AllCache) LoadBackup(b string, opts types.GetOptions) (*GroupWrapper, error) {
	ac.backupLock.Lock()
	data, err := ioutil.ReadFile(b)
	ac.backupLock.Unlock()

	if err != nil {
		return nil, err
	}
	backup := make(All)
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
	// app := val.(App)
	group, ok := app[opts.Group]
	if !ok {
		return nil, types.ErrNoAnyVersion
	}
	return &group, nil
}

// LoadGroup2Cache 将信息LoadOrStore to cache
func (ac *AllCache) LoadGroup2Cache(opts types.GetOptions, gw *GroupWrapper) error {
	var tryApp sync.Map
	iv, _ := ac.cache.LoadOrStore(opts.AppID, &tryApp)
	app := iv.(*sync.Map)
	if gw.Value == nil {
		gw.Value = make(Group)
	}
	app.LoadOrStore(opts.Group, gw)

	return nil
}
