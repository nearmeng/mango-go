package rainbow

import (
	"sync"

	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	"git.code.oa.com/trpc-go/trpc-go/config"

	pb "git.code.oa.com/rainbow/proto/api/configv3"
)

// Stream 基于SDK实现的watch
type Stream interface {
	Run()
	Get(key string) (*Response, error)
	Check() error
	AddWatcher(*WatchReq)
	DataType() string
}

// WatchStream stream基础结构
type WatchStream struct {
	cfg *Config

	api    SDK
	opts   []types.AssignGetOption
	outc   []chan Response
	submit chan *WatchReq

	// cb watch callback 函数
	cb     func(watch.Result, []*pb.Item) error
	cbChan chan Response
	// get 不同数据结构对应的 get 方法
	get func(string) (*Response, error)

	wg      sync.WaitGroup
	rw      sync.RWMutex
	done    chan error
	running bool
}

func getEventType(item *pb.Item) config.EventType {
	event := pb.Item_Event(item.GetEventType())
	for _, i := range []pb.Item_Event{
		pb.Item_UPDATE, pb.Item_ADD, pb.Item_ALL,
	} {
		if event == i {
			return config.EventTypePut
		}
	}
	if event == pb.Item_DELETE {
		return config.EventTypeDel
	}
	return config.EventTypeNull
}

func newWatchStream(cfg *Config, api SDK) *WatchStream {
	return &WatchStream{
		cfg:    cfg,
		api:    api,
		outc:   make([]chan Response, 0),
		submit: make(chan *WatchReq),
		opts:   cfg.GetOptions(),

		wg:      sync.WaitGroup{},
		done:    make(chan error),
		rw:      sync.RWMutex{},
		running: false,

		cbChan: make(chan Response, 1),
	}
}

// Run 开启监听任务
func (w *WatchStream) Run() {
	close(w.done)
	w.loop()
}

func prepareNotify(notify chan struct{}) {
	select {
	case notify <- struct{}{}:
	default:
	}

}

func notifying(buf []Response, outc []chan Response) []Response {

	for len(buf) > 0 && len(outc) > 0 {
		// 给每个订阅者发消息
		for _, consumer := range outc {
			consumer <- buf[0]
		}
		buf = buf[1:]
	}
	return buf
}

func (w *WatchStream) loop() {
	notify := make(chan struct{}, 1)
	buf := make([]Response, 0)
	for {
		select {
		case resp := <-w.cbChan:
			buf = append(buf, resp)
			prepareNotify(notify)

		case <-notify:
			buf = notifying(buf, w.outc)

		case req := <-w.submit:
			w.outc = append(w.outc, req.recv)
			close(req.done)
			prepareNotify(notify)
		}
	}

}

// Get 从 steam 中获取最新的配置
func (w *WatchStream) Get(key string) (*Response, error) {
	return w.get(key)
}

// Check 检查是否正在运行
func (w *WatchStream) Check() error {
	w.rw.RLock()
	if w.running {
		w.rw.RUnlock()
		return nil
	}
	w.rw.RUnlock()

	w.rw.Lock()
	defer w.rw.Unlock()
	if w.running {
		return nil
	}

	go w.Run()

	if err := <-w.done; err != nil {
		return err
	}

	w.running = true
	return nil
}

// AddWatcher 添加监视器
func (w *WatchStream) AddWatcher(req *WatchReq) {
	ignore := false
	// 新的sdk要过滤掉空key的监听
	if (w.DataType() == "kv" || w.DataType() == "") && req.key == "" {
		ignore = true
	}
	if !ignore {
		watcher := watch.Watcher{
			Key:        req.key,
			GetOptions: types.NewGetOptions(w.opts...),
			CB:         w.cb,
		}
		w.api.AddWatcher(watcher)
	}

	w.submit <- req
}

// DataType 查询数据结构
func (w *WatchStream) DataType() string {
	return w.cfg.Type
}
