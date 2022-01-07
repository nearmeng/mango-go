package confapi

import (
	"context"
	"fmt"

	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	config "git.code.oa.com/rainbow/proto/api/configv3"
)

// handleWatcher  监听当个key，或者一个group
func (c *ConfAPI) handleWatcher(w *watch.Watcher) (perr error) {
	oldVal := watch.Result{
		BasicResult: watch.BasicResult{AppID: w.AppID, Key: w.Key,
			GroupName: w.Group, Version: w.Version,
		},
	}
	opts := w.GetOptions
	gw, err := c.cache.GetGroup(opts)
	if err == nil {
		oldVal.Group = gw.Value
		oldVal.Version = gw.Version
		oldVal.VersionID = int64(gw.VersionID)
		oldVal.VersionName = gw.VersionName
		// 监听单个key
		if w.Key != "" {
			if v, ok := gw.Value[w.Key]; ok {
				oldVal.Value = v.(string)
			}
		}
	}
	if w.Version == "" {
		w.Version = gw.Version
	}
	// 请求远程
	var ctx = context.Background()
	req := &config.ReqPoll{
		Groups: []*config.WatchGroup{
			{
				ClientIds: c.buildClientInfos(w.GetOptions),
				AppId:     w.AppID,
				Group:     w.Group,
				VerUuid:   w.Version,
				EnvName:   w.GetOptions.EnvName,
				Key:       w.Key,
			},
		},
	}
	rsp, err := c.handler.Poll(ctx, req, &w.GetOptions)
	if err != nil {
		perr = fmt.Errorf("handler.Polling, err=%v, appID=%s group=%s key=%s",
			err, opts.AppID, opts.Group, w.Key)
		return perr
	}
	if rsp.GetRetCode() != 0 {
		if rsp.GetRetCode() == 708 {
			return nil
		}
		perr = fmt.Errorf("rsp.GetRetCode(), ret=%d, errmsg=%s,  appID=%s group=%s key=%s",
			rsp.GetRetCode(), rsp.GetRetMsg(), opts.AppID, opts.Group, w.Key)
		return perr
	}
	log.Debugf("%s", rsp.String())
	//TODO
	groups := rsp.GetGroups()
	if len(groups) == 0 {
		return fmt.Errorf("group empty  appID=%s group=%s key=%s",
			opts.AppID, opts.Group, w.Key)
	}
	//这里只取第一个
	/*wgroup := groups[0]
	if wgroup.Op == 0 {
		return nil
	}*/

	_, _, ckv, err := c.handlerPullConfig(opts, w.Key)
	if err != nil {
		log.Errorf("%#v \n %#v", ckv, err)
		return err
	}

	r := c.getBasicResult(opts, w.Key, ckv)
	if r == nil {
		perr = fmt.Errorf("getBasicResult null, appID=%s group=%s key=%s",
			opts.AppID, opts.Group, w.Key)
		return perr
	}
	w.Version = r.Version
	// 回调
	if w.CB != nil {
		w.CB(oldVal, ckv)
	}
	return nil
}
