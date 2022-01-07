package watch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/keep"
	v3 "git.code.oa.com/rainbow/proto/api/configv3"
)

// CallBack watch 回调
type CallBack func(oldVal Result, newVal []*v3.Item) error

// BasicResult 基本结果
type BasicResult struct {
	AppID       string
	GroupName   string
	Version     string
	VersionID   int64
	VersionName string
	Key         string // 监听单个key时不为空
	Value       string // 监听单个key时使用
	EnvName     string
}

// Result  watch结果
type Result struct {
	BasicResult
	Group keep.Group
}

// Watcher watcher
type Watcher struct {
	Key string
	types.GetOptions
	CB             CallBack
	RandomSleepDur time.Duration // watch的时候随机sleep的上限值
}

// CancelWatcher watcher with cacnel
type CancelWatcher struct {
	Watcher
	Cancel context.CancelFunc
}

// WatcherFactory watcher factory
type WatcherFactory struct {
	Factory      sync.Map          // key:AppID.Group or + Key, value:*watcher,
	ResultChan   chan *BasicResult // 用来通知backup更新
	BackupResult sync.Map          // key:同factory, value: *BasicResult
}

func (wf *WatcherFactory) watchKey(app, group, key, envname string) (string, error) {
	if app == "" || group == "" {
		return "", fmt.Errorf("watchKey invalid[%s:%s]", app, group)
	}
	// group watcher
	merge := app + "." + group
	if envname != "" {
		merge += "." + envname
	}
	// 单key watcher
	if key != "" {
		merge += "@" + key
	}
	return merge, nil
}

// AddCancelWatcher 添加watcher
func (wf *WatcherFactory) AddCancelWatcher(w *CancelWatcher) error {
	key, err := wf.watchKey(w.AppID, w.Group, w.Key, w.EnvName)
	if err != nil {
		return err
	}
	_, exists := wf.Factory.LoadOrStore(key, w)
	if exists {
		return types.ErrExistWatch
	}
	return nil
}

// IsExistsWatcher  是否存在watcher
func (wf *WatcherFactory) IsExistsWatcher(key string, opts types.GetOptions) bool {
	wkey, err := wf.watchKey(opts.AppID, opts.Group, "", opts.EnvName)
	if err != nil {
		return false
	}
	if _, ok := wf.Factory.Load(wkey); ok {
		return true
	}
	// 单key的watcher
	if key == "" {
		return false
	}
	wkey, _ = wf.watchKey(opts.AppID, opts.Group, key, opts.EnvName)
	if _, ok := wf.Factory.Load(wkey); ok {
		return true
	}

	return false
}

// DelCancelWatcher 获取watcher
func (wf *WatcherFactory) DelCancelWatcher(w *Watcher) error {
	key, err := wf.watchKey(w.AppID, w.Group, w.Key, w.EnvName)
	if err != nil {
		return err
	}
	val, ok := wf.Factory.Load(key)
	if !ok {
		return fmt.Errorf("key=%s watcher not exists", key)
	}
	cw := val.(*CancelWatcher)
	if cw.Cancel != nil {
		cw.Cancel()
	}
	wf.Factory.Delete(key)
	return nil
}

// NotifyResult 变更通知
func (wf *WatcherFactory) NotifyResult(r *BasicResult) error {
	_, err := wf.watchKey(r.AppID, r.GroupName, r.Key, r.EnvName)
	if err != nil {
		return err
	}
	// 放进chan
	select {
	case wf.ResultChan <- r:
	default:
		return fmt.Errorf("chan full[%v]", r)
	}
	return nil
}

// NeedBackup 是否需要备份
func (wf *WatcherFactory) NeedBackup(r *BasicResult) bool {
	/*key, err := wf.watchKey(r.AppID, r.GroupName, r.Key, r.EnvName)
	var todo bool
	if err != nil {
		return false
	}
	// 该key 是否需要更新
	todo = wf.thisKeyNeedBackup(key, r)
	if !todo {
		return false
	}
	wf.BackupResult.Store(key, r)
	if r.Key == "" {
		return todo
	}
	// key 不为空 比较下group 的版本
	key, _ = wf.watchKey(r.AppID, r.GroupName, "", r.EnvName)
	return wf.thisKeyNeedBackup(key, r)*/
	return true
}

// thisKeyNeedBackup 是否需要备份
/*func (wf *WatcherFactory) thisKeyNeedBackup(key string, r *BasicResult) bool {
	val, ok := wf.BackupResult.Load(key)
	if !ok {
		return true
	}
	old := val.(*BasicResult)
	if old.VersionID < r.VersionID {
		return true
	}
	return false
}*/
