package confapi

import (
	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/keep"
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	v3 "git.code.oa.com/rainbow/proto/api/configv3"
	"github.com/pkg/errors"
)

// handlerFullMode 处理全量模式的数据
func (c *ConfAPI) handlerFullMode(getOpts types.GetOptions, key string,
	rsp *v3.RspGetDatas) (keep.Group, string, []*v3.Item, error) {
	var g keep.Group
	ret := rsp.GetRetCode()
	err := types.Code2Error(ret)

	//  成功
	if ret == 0 {
		ckv := rsp.GetItems()
		if ckv == nil {
			return g, "", ckv, nil
		}
		r := c.getBasicResult(getOpts, key, ckv)
		if r == nil {
			log.Errorf("%s, getBasicResult null", getOpts.SimpleString())
			return g, "", ckv, nil
		}
		//  单key
		if key != "" {
			if !getOpts.NoUpdateCache {
				c.handlerChange(r, keep.UpdateTypeGetKey, ckv)
			}
		} else {
			if !getOpts.NoUpdateCache {
				c.handlerChange(r, keep.UpdateTypeGetGroup, ckv)
			}
		}
		g, val, err := c.handlerConfigKeyValue(ckv, key)
		return g, val, ckv, err

	}

	// 有错误发生, 且错误不为逻辑错误
	if err != types.ErrNoConfig && err != types.ErrNoAnyVersion &&
		err != types.ErrNotFoundKey {
		return nil, "", nil, errors.Errorf("rsp.GetRetCode=%d, GetRetMsg=%s", ret, rsp.GetRetMsg())
	}

	// 错误为逻辑错误---值不存在
	// 不需要更新缓存
	if getOpts.NoUpdateCache {
		return nil, "", nil, errors.Wrap(err, getOpts.SimpleString())
	}

	exist := c.cache.DeleteGroup(getOpts, key)
	if !exist {
		return nil, "", nil, errors.Wrap(err, getOpts.SimpleString())
	}
	// 文件落地
	if c.opts.IsUsingFileCache {
		c.wfactory.NotifyResult(&watch.BasicResult{
			AppID:     getOpts.AppID,
			GroupName: getOpts.Group,
			EnvName:   getOpts.EnvName,
			Key:       key,
		})
	}
	return nil, "", nil, errors.Wrap(err, getOpts.SimpleString())
}
