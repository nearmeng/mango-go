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
	defaultDomainExpire = 60 * time.Second
	staticDomainReload  = 30 * time.Second
	staticDomainFiles   = []string{"/data/L5Backup/name2sid.backup", "/data/L5Backup/name2sid.cache.bin"}
	domainss            *domains
	anonymouss          *anonymous

	ErrKGetSidFaild = errors.New("get sid failed")
)

func init() {
	domainss = &domains{store: make(map[string]*Domain)}
	anonymouss = &anonymous{store: make(map[int32]map[int32]*Domain)}
	go domainss.interval()
}

type domains struct {
	store map[string]*Domain
	l     sync.Mutex
}

// Query 通过l5域名获取Domain结构体，这个实现认为sid对应的mod/cmd是不会过期的
func (m *domains) Query(sid string) (*Domain, error) {
	now := time.Now()
	m.l.Lock()
	domain, exists := m.store[sid]
	if exists && domain.networking {
		m.l.Unlock()
		if domain.mod == 0 {
			domain.sidLock.Lock() // 只是为了获取锁, 下面已经把它锁住
			domain.sidLock.Unlock()

			if domain.mod != 0 {
				return domain, nil
			}
			return nil, ErrKGetSidFaild
		}
		return domain, nil
	}

	domain = &Domain{
		name:       sid,
		mod:        0,
		cmd:        0,
		expire:     now.Add(defaultDomainExpire),
		balancer:   NewBalancer(defaultBalancer),
		inited:     false,
		networking: true,
	}
	m.store[sid] = domain

	domain.sidLock.Lock() // 在lock之前,不能让它在上面的某个地方lock住.所以先lock再unlock
	m.l.Unlock()
	defer domain.sidLock.Unlock()

	buf, err := dial(QOS_CMD_QUERY_SNAME, atomic.AddInt32(&seqno, 1), domain.mod, domain.cmd, int32(os.Getpid()), int32(len(domain.name)), domain.name)
	if err != nil {
		domain.networking = false
		return nil, err
	}
	domain.mod = int32(defaultEndian.Uint32(buf[0:4]))
	domain.cmd = int32(defaultEndian.Uint32(buf[4:8]))
	domain.networking = true

	// 开始周期更新
	err = domain.getFromAgentAndLocal()
	if err == nil {
		domain.inited = true
	}
	go func() {
		var timeDiff time.Duration
		for {
			// 保护一下更新时间间隔, 如果getFromAgentAndLocal发生错误, 也1s之后再更新
			// 避免cpu彪高
			timeDiff = domain.expire.Sub(time.Now())
			if timeDiff > 0 {
				time.Sleep(timeDiff)
			} else {
				time.Sleep(time.Second)
			}
			domain.getFromAgentAndLocal()
		}
	}()

	return domain, nil
}

// interval 每隔30s从/data/L5Backup/name2sid.backup加载 l5域名对应的 modid:cmdid 映射关系
func (m *domains) interval() {
	interval := time.NewTicker(staticDomainReload)
	var now time.Time
	for {
		select {
		case <-interval.C:
			var (
				err error
				fp  *os.File
			)
			now = time.Now()
			for _, v := range staticDomainFiles {
				if fp, err = os.Open(v); err != nil {
					// log.Printf("open file failed: %s", err.Error())
					continue
				}
				for {
					var (
						name string
						mod  int32
						cmd  int32
					)
					if n, fail := fmt.Fscanln(fp, &name, &mod, &cmd); n == 0 || fail != nil {
						break
					}
					m.l.Lock()
					_, exists := m.store[name]
					if !exists {
						m.store[name] = &Domain{
							name:       name,
							mod:        mod,
							cmd:        cmd,
							expire:     now.Add(defaultDomainExpire),
							balancer:   NewBalancer(defaultBalancer),
							inited:     false,
							networking: true,
						}
					}
					m.l.Unlock()
				}
				fp.Close()
			}
		}
	}
}

// Domain 每个l5 id对应的Domain
type Domain struct {
	name string
	mod  int32
	cmd  int32

	l sync.RWMutex
	// 获取route之后,要过多久再去获取下一次route,是从agent的上一次返回中下发的,所以这里把它记录下来
	expire   time.Time
	balancer Balancer

	// [MOD/CMD模式使用]下面的变量都是第一次初始化临时使用的
	inited     bool
	updateLock sync.Mutex // 为了不和Lock冲突

	// [sid模式下使用]
	sidLock    sync.Mutex
	networking bool
}

// Mod 返回 modid
func (d *Domain) Mod() int32 {
	return d.mod
}

// Cmd 返回 cmdid
func (d *Domain) Cmd() int32 {
	return d.cmd
}

// Name 返回 l5域名
func (d *Domain) Name() string {
	return d.name
}

// anonymous 保存所有 modid:cmdid 对应的所有Domain
type anonymous struct {
	store map[int32]map[int32]*Domain
	l     sync.RWMutex
}

// Get 获取每个 modid:cmdid 对应的 Domain结构体
func (a *anonymous) Get(mod int32, cmd int32) *Domain {
	a.l.RLock()
	m, exists := a.store[mod]
	a.l.RUnlock()

	if !exists {
		a.l.Lock()
		m, exists = a.store[mod]
		if !exists {
			m = make(map[int32]*Domain)
			a.store[mod] = m
			domain := NewDomain(mod, cmd)
			m[cmd] = domain
			a.l.Unlock()
			return domain
		}
		a.l.Unlock()
	}

	a.l.RLock()
	domain, exists := m[cmd]
	a.l.RUnlock()
	if exists {
		return domain
	}

	a.l.Lock()
	domain, exists = m[cmd]
	if exists {
		a.l.Unlock()
		return domain
	}
	domain = NewDomain(mod, cmd)
	m[cmd] = domain
	a.l.Unlock()

	return domain
}

// NewDomain 新建一个Domain，首次初始化从agent拉取server列表，并启动协程，定时3s更新server list
func NewDomain(mod int32, cmd int32) *Domain {
	now := time.Now()
	domain := &Domain{
		name:     "",
		mod:      mod,
		cmd:      cmd,
		expire:   now.Add(defaultDomainExpire),
		balancer: NewBalancer(defaultBalancer),
		inited:   false,
	}

	err := domain.getFromAgentAndLocal()
	if err == nil {
		domain.inited = true
	}
	go func() {
		var timeDiff time.Duration
		for {
			// 保护一下更新时间间隔, 如果getFromAgentAndLocal发生错误, 也1s之后再更新
			// 避免cpu彪高
			timeDiff = domain.expire.Sub(time.Now())
			if timeDiff > 0 {
				time.Sleep(timeDiff)
			} else {
				time.Sleep(time.Second)
			}
			domain.getFromAgentAndLocal()
		}
	}()

	return domain
}

// SetBalancer 设置每个Domain的负载器
func (d *Domain) SetBalancer(b Balancer) {
	d.l.Lock()
	d.balancer = b
	d.l.Unlock()
}
