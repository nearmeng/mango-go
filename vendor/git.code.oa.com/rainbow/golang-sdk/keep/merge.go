package keep

import (
	"git.code.oa.com/rainbow/golang-sdk/config"
	"git.code.oa.com/rainbow/golang-sdk/types"
)

var (
	defaultEnv = "Default"
)

// MergeCache 默认环境和其他的环境的cache
type MergeCache struct {
	othersEnv FullCache
}

func (mc *MergeCache) mergePath(b string) string {
	return b
}

// Get 从本地缓存取值
func (mc *MergeCache) Get(key string, opts types.GetOptions) (string, error) {
	if opts.EnvName == "" {
		opts.EnvName = defaultEnv
	}
	return mc.othersEnv.Get(key, opts)
}

// CoverGroup 覆盖更新
func (mc *MergeCache) CoverGroup(opts types.GetOptions, items []*config.KeyValueItem) {
	if opts.EnvName == "" {
		opts.EnvName = defaultEnv
	}
	mc.othersEnv.CoverGroup(opts, items)
}

// UpdateGroup 覆盖更新
func (mc *MergeCache) UpdateGroup(opts types.GetOptions, ut UpdateType, items []*config.KeyValueItem) {
	if opts.EnvName == "" {
		opts.EnvName = defaultEnv
	}
	mc.othersEnv.UpdateGroup(opts, ut, items)
}

// GetGroup 从本地缓存获取group
func (mc *MergeCache) GetGroup(opts types.GetOptions) (GroupWrapper, error) {
	if opts.EnvName == "" {
		opts.EnvName = defaultEnv
	}
	return mc.othersEnv.GetGroup(opts)
}

// Backup 备份
func (mc *MergeCache) Backup(env, b string) error {
	return mc.othersEnv.Backup(mc.mergePath(b))
}

// LoadBackup 加载备份文件, 单group
func (mc *MergeCache) LoadBackup(b string, opts types.GetOptions) (*GroupWrapper, error) {
	if opts.EnvName == "" {
		opts.EnvName = defaultEnv
	}
	return mc.othersEnv.LoadBackup(mc.mergePath(b), opts)
}

// LoadGroup2Cache 将信息LoadOrStore to cache
func (mc *MergeCache) LoadGroup2Cache(opts types.GetOptions, gw *GroupWrapper) error {
	if opts.EnvName == "" {
		opts.EnvName = defaultEnv
	}
	return mc.othersEnv.LoadGroup2Cache(opts, gw)
}
