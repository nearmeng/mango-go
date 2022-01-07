package confapi

import (
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/keep"
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	config "git.code.oa.com/rainbow/proto/api/configv3"
)

func existKey(k string, m map[string]string) bool {
	if m == nil {
		return false
	}
	if _, ok := m[k]; ok {
		return true
	}
	return false
}

func (c *ConfAPI) buildClientInfos(opts types.GetOptions) map[string]string {
	conf := make(map[string]string)
	ip := opts.IP
	if ip == "" {
		ip = types.GetLocalIPStr()
	}

	if !existKey("ip", opts.ClientInfo) {
		conf["ip"] = ip
	}
	if opts.Uin != "" {
		if !existKey("uin", opts.ClientInfo) {
			conf["uin"] = opts.Uin
		}
	}
	if opts.ClientInfo != nil {
		for k, v := range opts.ClientInfo {
			conf[k] = v
		}
	}
	return conf
}

// getBasicResult 获取基本Result
func (c *ConfAPI) getBasicResult(opts types.GetOptions, key string,
	items []*config.Item) *watch.BasicResult {
	if len(items) <= 0 {
		return nil
	}
	result := &watch.BasicResult{
		AppID:       opts.AppID,
		GroupName:   opts.Group,
		Version:     items[0].GetVerUuid(),
		VersionID:   int64(items[0].GetVerId()),
		VersionName: items[0].GetVerName(),
		Key:         key,
		EnvName:     opts.EnvName,
	}
	return result
}

// handlerChange 处理变更
func (c *ConfAPI) handlerChange(result *watch.BasicResult, ut keep.UpdateType, items []*config.Item) error {
	// 更新缓存
	if c.opts.IsUsingLocalCache {
		opts := types.GetOptions{
			AppID:   result.AppID,
			Group:   result.GroupName,
			EnvName: result.EnvName,
		}
		updateItems := make([]*config.Item, 0)
		if ut == keep.UpdateTypeGetGroup || ut == keep.UpdateTypePollGroup {
			for i := 0; i < len(items); i++ {
				kvItem := items[i]
				if kvItem == nil {
					continue
				}
				if kvItem.EventType == int32(config.Item_ALL) {
					nItems := []*config.Item{kvItem}
					c.cache.CoverGroup(opts, nItems)
				} else {
					updateItems = append(updateItems, kvItem)
				}
			}
		} else {
			updateItems = items
		}
		c.cache.UpdateGroup(opts, ut, updateItems)
	}

	// 文件落地
	if c.opts.IsUsingFileCache {
		c.wfactory.NotifyResult(result)
	}
	return nil
}

// handlerNotifyResult 处理文件通知结果
func (c *ConfAPI) handlerNotifyResult() {
	go func() {
		for !c.quit {
			result := <-c.wfactory.ResultChan
			if result == nil {
				continue
			}
			c.handlerBackup(result)
		}
	}()
}

// handlerBackup 处理文件备份
func (c *ConfAPI) handlerBackup(r *watch.BasicResult) {
	todo := c.wfactory.NeedBackup(r)
	if !todo {
		return
	}
	c.cache.Backup(c.opts.FileCachePath)
}
