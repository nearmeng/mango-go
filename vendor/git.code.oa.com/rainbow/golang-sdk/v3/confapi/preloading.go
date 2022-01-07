package confapi

import (
	"context"
	"fmt"

	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/keep"
	config "git.code.oa.com/rainbow/proto/api/configv3"
)

// PreLoad preload
func (c *ConfAPI) PreLoad() error {
	groupLen := len(c.opts.Groups)
	// 没group 不需要预加载
	if c.opts.AppID == "" || groupLen == 0 || !c.opts.IsUsingLocalCache {
		return nil
	}
	// 依次加载group
	for i := 0; i < groupLen; i++ {
		if c.opts.Groups[i] == "" {
			continue
		}
		gopts := types.GetOptions{
			AppID:   c.opts.AppID,
			Group:   c.opts.Groups[i],
			EnvName: c.opts.EnvName,
		}
		_, _, _, err := c.handlerPullConfig(gopts, "")
		if err == nil {
			continue
		}
		log.Warnf("[%s:%s]preload from remote failed,preload from local file [%s]", gopts.AppID, gopts.Group, err.Error())
		// 降级加载
		if c.opts.IsUsingFileCache {
			gw, err := c.cache.LoadBackup(c.opts.FileCachePath, gopts)
			if err != nil {
				log.Errorf("[%s:%s]preload from local file failed [%s]", gopts.AppID, gopts.Group, err.Error())
				continue
			}
			c.cache.LoadGroup2Cache(gopts, gw)
		}
	}
	return nil
}

func (c *ConfAPI) fillDefaultArgs(opts *types.GetOptions) {
	// 没有传用初始化时的值
	if opts.AppID == "" {
		opts.AppID = c.opts.AppID
	}
	if opts.Group == "" && len(c.opts.Groups) > 0 {
		opts.Group = c.opts.Groups[0]
	}
	if opts.EnvName == "" {
		opts.EnvName = c.opts.EnvName
	}
}

func (c *ConfAPI) handlerPullConfig(getOpts types.GetOptions, key string) (
	keep.Group, string, []*config.Item, error) {
	var ctx = context.Background()

	c.fillDefaultArgs(&getOpts)
	// 参数检查一下
	if getOpts.AppID == "" || getOpts.Group == "" {
		err := fmt.Errorf("invalid argument, appID=%s, group=%s", getOpts.AppID, getOpts.Group)
		return nil, "", nil, err
	}
	req := &config.ReqGetDatas{
		AppId:   getOpts.AppID,
		Group:   getOpts.Group,
		EnvName: getOpts.EnvName,
		Opts: &config.Options{
			ClientVerUuid: getOpts.Version,
			Key:           key,
			LastId:        getOpts.Start,
			PageSize:      getOpts.Offset,
		},
	}
	req.ClientIds = c.buildClientInfos(getOpts)
	rsp, err := c.handler.Getdatas(ctx, req, &getOpts)
	if err != nil {
		return nil, "", nil, fmt.Errorf("handler.PullConfig failed, err=%s", err.Error())
	}

	log.Debugf("%s", rsp.String())

	return c.handlerFullMode(getOpts, key, rsp)
}

func (c *ConfAPI) handlerConfigKeyValue(ckv []*config.Item,
	key string) (keep.Group, string, error) {
	var g = make(keep.Group)
	var val string

	if ckv == nil {
		return g, "", nil
	}
	items := ckv
	itemsLen := len(ckv)
	for i := 0; i < itemsLen; i++ {
		kvItem := items[i]
		if kvItem == nil {
			continue
		}

		if kvItem.StructType != int32(config.Item_KV) {
			continue
		}
		kvs := kvItem.GetKvs()
		if kvs == nil {
			continue
		}
		for _, v := range kvs.GetKvs() {
			g[v.GetKey()] = v.GetValue()
			if key != "" && key == v.GetKey() {
				val = v.GetValue()
			}
		}
	}
	return g, val, nil
}
