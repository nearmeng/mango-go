package l5

import (
	"errors"
	"math/rand"
	"sync"
)

// 负载均衡器类型
const (
	CL5_LB_TYPE_WRR = iota
	CL5_LB_TYPE_STEP
	CL5_LB_TYPE_MOD
	CL5_LB_TYPE_CST_HASH
	CL5_LB_TYPE_RANDOM
)

var (
	defaultBalancer = CL5_LB_TYPE_WRR
	ErrNotBalancer  = errors.New("not set balancer")
	ErrNotFound     = errors.New("not found")
	ErrInsertFailed = errors.New("insert failed")
)

// Balancer 负载器定义
type Balancer interface {
	Get() (*Server, error)
	Set(*Server) error
	Remove(*Server) error
	Destroy() error
}

// weightedRoundRobin 带过期时间的权重轮询调度实现
type weightedRoundRobin struct {
	l          sync.RWMutex
	list       []*Server
	index      int
	max        int32
	gcd        int32
	currWeight int32
}

// Get 加权轮询取出server
func (w *weightedRoundRobin) Get() (*Server, error) {
	/*优先分配准备探测的故障机,采用普通的轮询算法*/
	// weight是负数,表示agent想要让你重试这个svr abs(weight)次,重试次数完成,我这里就把它踢掉了
	// 存在写操作,所以使用写锁
	w.l.Lock()
	// 最终还是选择全部加写锁,避免各种问题.
	// 使用读锁的做法代码还保留着,注释掉了,追求性能极限可以恢复注释,把这个deffer去掉
	defer w.l.Unlock()

	length := len(w.list)
	if length < 1 {
		//w.l.Unlock()
		return nil, ErrNotFound
	}

	if w.list[w.index].weight <= 0 {
		srv := w.list[w.index]
		srv.weight++
		if srv.weight >= 0 {
			w.list = append(w.list[0:w.index], w.list[w.index+1:]...)
			w.index = 0
		}
		//w.l.Unlock()
		return srv, nil
	}
	//w.l.Unlock()

	/*// 下面使用读lock, 需要重新获取一次长度
	w.l.RLock()
	defer w.l.RUnlock()

	length = len(w.list)
	if length < 1 {
		return nil, ErrNotFound
	}*/
	/*开始获取ip*/
	var srv *Server
	for {
		srv = w.list[w.index]
		if srv.weight > w.currWeight {
			w.index = (w.index + 1) % length
			if w.index == 0 {
				/*每分配完一轮就增加权重*/
				if w.currWeight >= w.max {
					w.currWeight = 0
				} else {
					w.currWeight += w.gcd
				}
			}
			return srv, nil
		} else {
			/*因为权重已经按照从大到小排序，故不用再查看last_index之后的数据，直接调整当前权重后从头开始*/
			if w.currWeight >= w.max {
				w.currWeight = 0
			} else {
				w.currWeight += w.gcd
			}
			w.index = 0
		}
	}
}

// Set 按weight从大到小排好序
func (w *weightedRoundRobin) Set(s *Server) error {
	w.l.Lock()
	defer w.l.Unlock()

	// weight是负数,表示agent想要让你重试这个svr abs(weight)次.
	if s.weight <= 0 {
		w.list = append([]*Server{s}, w.list...)
		w.index = 0
		return nil
	}

	length := len(w.list)
	if length == 0 {
		w.list = append(w.list, s)
		w.gcd = s.weight
		w.max = s.weight
		w.currWeight = 0
		w.index = 0
		return nil
	}

	for i := 0; i < length; i++ {
		if w.list[i].weight > 0 && s.weight > w.list[i].weight {
			w.list = append(w.list[0:i], append([]*Server{s}, w.list[i:]...)...)
		} else if i == length-1 {
			w.list = append(w.list, s)
		} else {
			continue
		}

		// 重置gcd, max, index, currWeight
		w.gcd = GreatestCommonDivider(w.gcd, s.weight)
		for j := range w.list {
			if w.list[j].weight > 0 {
				w.max = w.list[j].weight
				break
			}
		}
		w.currWeight = 0
		// 如果有需要探测的svr,就一定从需要探测的svr开始
		if w.list[0].weight > 0 {
			w.index = rand.Intn(len(w.list))
		} else {
			w.index = 0
		}
		return nil
	}

	return ErrInsertFailed
}

// TODO: 这个函数有bug,但是现在没人用.
// 正确的写法是直接rebuild,重设gcd,currWeight=0,rand index
func (w *weightedRoundRobin) Remove(srv *Server) error {
	w.l.Lock()
	defer w.l.Unlock()
	length := len(w.list)
	w.gcd = 1
	for i := length - 1; i >= 0; i-- {
		if w.list[i].ip == srv.ip && w.list[i].port == srv.port {
			if i == length-1 {
				w.list = w.list[0:i]
			} else {
				go w.list[i].report(true)
				w.list = append(w.list[0:i], w.list[i+1:]...)
			}
			length--
		} else {
			w.gcd = GreatestCommonDivider(w.gcd, w.list[i].weight)
			if w.index > 0 && w.index >= i {
				w.index--
			}
		}
	}
	return nil
}

// Destroy 上报老的统计数据到agent，清空server列表，重新初始化
func (w *weightedRoundRobin) Destroy() error {
	w.l.Lock()
	defer w.l.Unlock()
	w.gcd = 1
	for _, v := range w.list {
		go v.report(true)
	}
	w.list = make([]*Server, 0)
	w.index = 0
	w.max = 0
	return nil
}

// NewBalancer 新建一个负载器
func NewBalancer(typ int) Balancer {
	switch typ {
	case CL5_LB_TYPE_WRR:
		return &weightedRoundRobin{
			list:  []*Server{},
			index: 0,
			max:   0,
			gcd:   1,
		}
		//@todo
	}
	return nil
}
