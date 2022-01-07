package rainbow

import (
	"encoding/json"

	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	pb "git.code.oa.com/rainbow/proto/api/configv3"
	"git.code.oa.com/trpc-go/trpc-go/config"
)

// GroupWatchStream 基于SDK实现的 group watch stream
type GroupWatchStream struct {
	*WatchStream
}

// NewGroupStream 创建 group 数据格式的 stream watch
func NewGroupStream(cfg *Config) (Stream, error) {
	rainbow, err := globalSDKBuilder.BuildSDK(cfg.InitOptions())
	if err != nil {
		return nil, err
	}

	ws := &GroupWatchStream{newWatchStream(cfg, rainbow)}

	cbFunc := func(oldVal watch.Result, newVal []*pb.Item) error {
		v, err := ws.api.GetGroup(ws.opts...)
		if err != nil {
			return err
		}
		value, _ := json.Marshal(v)
		rsp := Response{
			key:   ws.cfg.Group,
			value: string(value),
		}
		rsp.event = config.EventTypePut
		ws.cbChan <- rsp

		return nil
	}

	// 设置回调函数
	ws.cb = cbFunc

	getFunc := func(key string) (*Response, error) {
		v, err := ws.api.GetGroup(ws.opts...)
		if err != nil {
			return nil, err
		}
		value, _ := json.Marshal(v)
		rsp := &Response{
			key:   key,
			value: string(value),
			event: config.EventTypePut,
		}

		return rsp, nil
	}
	// 设置拉取函数
	ws.get = getFunc

	return ws, nil
}
