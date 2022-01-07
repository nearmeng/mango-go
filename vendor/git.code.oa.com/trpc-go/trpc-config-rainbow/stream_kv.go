package rainbow

import (
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	"git.code.oa.com/trpc-go/trpc-go/config"

	pb "git.code.oa.com/rainbow/proto/api/configv3"
)

// KVWatchStream 基于SDK实现的 KV 结构配置 watch
type KVWatchStream struct {
	*WatchStream
}

// NewKVStream 创建 KVStream
func NewKVStream(cfg *Config) (Stream, error) {
	rainbow, err := globalSDKBuilder.BuildSDK(cfg.InitOptions())
	if err != nil {
		return nil, err
	}

	ws := &KVWatchStream{newWatchStream(cfg, rainbow)}
	// 设置回调函数
	ws.cb = generateCallbackFunc(ws.cbChan)
	getFunc := func(key string) (*Response, error) {
		defer func() {
			watcher := watch.Watcher{
				Key:        key,
				GetOptions: types.NewGetOptions(ws.opts...),
				CB:         ws.cb,
			}
			_ = ws.api.AddWatcher(watcher)
		}()
		v, err := ws.api.Get(key, ws.opts...)
		if err != nil {
			return nil, err
		}
		rsp := &Response{
			key:   key,
			value: v,
			event: config.EventTypePut,
		}
		return rsp, nil
	}
	// 设置拉取函数
	ws.get = getFunc

	return ws, nil
}

func generateCallbackFunc(cbChan chan Response) func(watch.Result, []*pb.Item) error {
	cbFunc := func(oldVal watch.Result, newVal []*pb.Item) error {
		for _, item := range newVal {
			for _, v := range item.GetKvs().GetKvs() {
				rsp := Response{
					key:   v.GetKey(),
					value: v.GetValue(),
				}
				rsp.event = getEventType(item)
				cbChan <- rsp
			}
		}
		return nil
	}

	return cbFunc
}
