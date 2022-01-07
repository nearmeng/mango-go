package l5

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	staticServerFiles = []string{"/data/L5Backup/current_route.backup", "/data/L5Backup/current_route_v2.backup"}
	// defaultServerExpire = 3 * time.Second           // 定时3s从agent更新所有 ip port weight列表[已去除,从agent获取的时间间隔改从agent获取]
	ErrKL5Overload = errors.New("l5 overload") // l5过载 踢光后端机器

	seqno = int32(0)
)

// getFromAgentAndLocal 先从agent中获取 ip port weight, agent取失败则从静态文件中获取
func (d *Domain) getFromAgentAndLocal() error {
	var buf []byte
	var list []*Server
	var err error
	var tmpIP uint32
	var tmpPort uint16
	buf, err = dial(QOS_CMD_BATCH_GET_ROUTE_WEIGHT, atomic.AddInt32(&seqno, 1), d.mod, d.cmd, int32(os.Getpid()), int32(gVersion))
	if err == nil && len(buf) >= 16 {
		now := time.Now()
		size := len(buf) - 16
		list = make([]*Server, size/14)
		for k := range list {
			tmpIP = defaultEndian.Uint32(buf[16+k*14 : 20+k*14])
			tmpPort = defaultEndian.Uint16(buf[20+k*14 : 22+k*14])
			list[k] = &Server{
				domain:  d,
				ip:      tmpIP,
				port:    tmpPort,
				weight:  int32(defaultEndian.Uint32(buf[22+k*14 : 26+k*14])),
				total:   int32(defaultEndian.Uint32(buf[26+k*14 : 30+k*14])),
				strIP:   Ip2StringLittle(tmpIP),
				intPort: int(tmpPort),
				stat: Stat{
					LastReport: now,
				},
			}

			// 后续都使用weight,不适用total
			// total的原意是"分配的调用次数 < 0代表需要探测 =0 代表机器被剔除 >0代表机器可调用数"
			if list[k].total <= 0 {
				list[k].weight = list[k].total
			}
		}
		// agent告诉我什么时候超时
		var expireMilliseconds = defaultEndian.Uint32(buf[12:16])
		if expireMilliseconds < 1000 {
			expireMilliseconds = 1000
		}
		d.expire = now.Add(time.Millisecond * time.Duration(expireMilliseconds))
		if err = d.Set(list); err != nil {
			return err
		}

		return nil
	}
	fmt.Printf("getFromAgentAndLocal err:%s, buf len:%d\n", err, len(buf))

	// 从agent取失败则从静态文件中获取
	var fp *os.File
	list = nil

	var (
		mod  int32
		cmd  int32
		ip   string
		port uint16
		n    int
		fail error
	)
	for _, v := range staticServerFiles {
		if fp, err = os.Open(v); err != nil {
			continue
		}
		now := time.Now()
		for {
			if n, fail = fmt.Fscanln(fp, &mod, &cmd, &ip, &port); n == 0 || fail != nil {
				break
			}
			if d.mod != mod || d.cmd != cmd {
				continue
			}
			list = append(list, &Server{
				domain:  d,
				ip:      String2IpLittle(ip),
				port:    HostInt16ToLittle(port),
				weight:  100, //default weight: 100
				total:   0,
				strIP:   ip,
				intPort: int(port),
				stat: Stat{
					LastReport: now,
				},
			})
		}
		fp.Close()
	}

	if err = d.Set(list); err != nil {
		return err
	}
	d.expire = time.Now().Add(defaultDomainExpire)
	return nil
}

// Get 从balancer中获取路由server
func (d *Domain) Get() (*Server, error) {
	d.l.RLock() // 定时更新server列表时，会清空老serverlist，这里需要加锁
	if d.balancer == nil {
		d.l.RUnlock()
		return nil, ErrNotBalancer
	}
	srv, err := d.balancer.Get()
	d.l.RUnlock()

	if err == nil {
		return srv.allocate(), nil
	}

	if err == ErrNotFound {
		if d.inited {
			return nil, ErrNotFound
		}

		d.updateLock.Lock()
		if d.inited {
			d.updateLock.Unlock()
			return nil, ErrKL5Overload
		}
		err = d.getFromAgentAndLocal()
		if err == nil {
			d.inited = true
		}
		d.updateLock.Unlock()

		if err != nil {
			return nil, err
		}

		d.l.RLock() // protect balancer
		if d.balancer == nil {
			d.l.RUnlock()
			return nil, ErrNotBalancer
		}
		srv, err = d.balancer.Get()
		d.l.RUnlock()

		if err != nil {
			return nil, err
		}

		return srv.allocate(), nil
	}

	return nil, err
}

// Set 设置每个l5 id 对应的server列表
func (d *Domain) Set(list []*Server) error {
	d.l.Lock()
	defer d.l.Unlock()

	if err := d.Destroy(); err != nil {
		return err
	}
	if d.balancer == nil {
		return ErrNotBalancer
	}
	for _, v := range list {
		if err := d.balancer.Set(v); err != nil {
			return err
		}
	}
	return nil
}

// Destroy 重新初始化 balancer
func (d *Domain) Destroy() error {
	if d.balancer == nil {
		return ErrNotBalancer
	}
	return d.balancer.Destroy()
}

// Server 路由server信息
type Server struct {
	domain *Domain
	ip     uint32 // Little
	port   uint16 // Little
	weight int32
	total  int32
	stat   Stat

	strIP   string
	intPort int

	l sync.RWMutex
}

// Ip 字符串ip地址
func (s *Server) Ip() string {
	return s.strIP
}

// Port 端口
func (s *Server) Port() int {
	return s.intPort
}

// LittleIp 小端数字ip
func (s *Server) LittleIp() uint32 {
	return s.ip
}

// LittlePort 小端端口
func (s *Server) LittlePort() uint16 {
	return s.port
}
