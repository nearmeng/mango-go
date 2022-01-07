package rainbow

import (
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	pb "git.code.oa.com/rainbow/proto/api/configv3"
	"git.code.oa.com/trpc-go/trpc-go/config"
)

// TableWatchStream 基于SDK实现的 table watch stream
type TableWatchStream struct {
	*WatchStream
}

// NewTableStream 创建 table 数据格式的 stream watch
func NewTableStream(cfg *Config) (Stream, error) {
	rainbow, err := globalSDKBuilder.BuildSDK(cfg.InitOptions())
	if err != nil {
		return nil, err
	}

	ws := &TableWatchStream{newWatchStream(cfg, rainbow)}

	cbFunc := func(oldVal watch.Result, newVal []*pb.Item) error {
		v, err := ws.api.GetTable(ws.opts...)
		if err != nil {
			return err
		}
		rsp := Response{
			key:   ws.cfg.Group,
			value: spliceString(v.Rows),
		}
		rsp.event = config.EventTypePut
		ws.cbChan <- rsp

		return nil
	}

	// 设置回调函数
	ws.cb = cbFunc

	getFunc := func(key string) (*Response, error) {
		v, err := ws.api.GetTable(ws.opts...)
		if err != nil {
			return nil, err
		}

		rsp := &Response{
			key:   key,
			value: spliceString(v.Rows),
			event: config.EventTypePut,
		}

		return rsp, nil
	}
	// 设置拉取函数
	ws.get = getFunc

	return ws, nil
}
