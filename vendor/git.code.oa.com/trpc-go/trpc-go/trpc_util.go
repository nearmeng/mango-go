package trpc

import (
	"context"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
	"git.code.oa.com/trpc-go/trpc-go/log"
)

// PanicBufLen panic调用栈日志buffer大小，默认1024
var PanicBufLen = 1024

// ----------------------- trpc 通用工具类函数 ------------------------------------ //

// Message 从ctx获取请求通用数据
func Message(ctx context.Context) codec.Msg {
	return codec.Message(ctx)
}

// BackgroundContext 携带空msg的background context，常用于用户自己创建任务函数的场景
func BackgroundContext() context.Context {
	cfg := GlobalConfig()
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithCalleeContainerName(cfg.Global.ContainerName)
	msg.WithNamespace(cfg.Global.Namespace)
	msg.WithEnvName(cfg.Global.EnvName)
	if cfg.Global.EnableSet == "Y" {
		msg.WithSetName(cfg.Global.FullSetName)
	}
	if len(cfg.Server.Service) > 0 {
		msg.WithCalleeServiceName(cfg.Server.Service[0].Name)
	} else {
		msg.WithCalleeApp(cfg.Server.App)
		msg.WithCalleeServer(cfg.Server.Server)
	}
	return ctx
}

// GetMetaData 从请求里面获取key的透传字段
func GetMetaData(ctx context.Context, key string) []byte {
	msg := codec.Message(ctx)
	if len(msg.ServerMetaData()) > 0 {
		return msg.ServerMetaData()[key]
	}
	return nil
}

// SetMetaData 设置透传字段返回给上游, 非并发安全
func SetMetaData(ctx context.Context, key string, val []byte) {
	msg := codec.Message(ctx)
	if len(msg.ServerMetaData()) > 0 {
		msg.ServerMetaData()[key] = val
		return
	}
	md := make(map[string][]byte)
	md[key] = val
	msg.WithServerMetaData(md)
}

// Request 获取trpc业务协议请求包头，不存在则返回空的包头结构体
func Request(ctx context.Context) *RequestProtocol {
	msg := codec.Message(ctx)
	request, ok := msg.ServerReqHead().(*RequestProtocol)
	if !ok {
		return &RequestProtocol{}
	}
	return request
}

// Response 获取trpc业务协议响应包头，不存在则返回空的包头结构体
func Response(ctx context.Context) *ResponseProtocol {
	msg := codec.Message(ctx)
	response, ok := msg.ServerRspHead().(*ResponseProtocol)
	if !ok {
		return &ResponseProtocol{}
	}
	return response
}

// CloneContext 复制context得到一个保留value不cancel的context. 用于handler异步处理时脱离原有的超时控制并保留原有的上下文信息.
// trpc handler函数return之后, ctx会cancel掉, 会将ctx中的Msg放回池中, 关联的模调信息和logger会释放掉.
// 业务handler中异步处理时, 需要在启动goroutine前调用此方法复制context, 脱离原有的超时控制, 保留msg里的信息用于模调监控,
// 保留logger context用于打印关联的日志, 保留context中的其它value如tracing context等.
func CloneContext(ctx context.Context) context.Context {
	oldMsg := codec.Message(ctx)
	newCtx, newMsg := codec.WithNewMessage(detach(ctx))
	codec.CopyMsg(newMsg, oldMsg)
	return newCtx
}

type detachedContext struct{ parent context.Context }

func detach(ctx context.Context) context.Context { return detachedContext{ctx} }

// Deadline implements context.Deadline
func (v detachedContext) Deadline() (time.Time, bool) { return time.Time{}, false }

// Done implements context.Done
func (v detachedContext) Done() <-chan struct{} { return nil }

// Err implements context.Err
func (v detachedContext) Err() error { return nil }

// Value implements context.Value
func (v detachedContext) Value(key interface{}) interface{} { return v.parent.Value(key) }

// GoAndWait 封装更安全的多并发调用, 启动goroutine并等待所有处理流程完成，自动recover
// 返回值error返回的是多并发协程里面第一个返回的不为nil的error，主要用于关键路径判断，当多并发协程里面有一个是关键路径且有失败则返回err，其他非关键路径并发全部返回nil
func GoAndWait(handlers ...func() error) error {
	var (
		wg   sync.WaitGroup
		once sync.Once
		err  error
	)
	for _, f := range handlers {
		wg.Add(1)
		go func(handler func() error) {
			defer func() {
				if e := recover(); e != nil {
					buf := make([]byte, PanicBufLen)
					buf = buf[:runtime.Stack(buf, false)]
					log.Errorf("[PANIC]%v\n%s\n", e, buf)
					report.PanicNum.Incr()
					once.Do(func() {
						err = errs.New(errs.RetServerSystemErr, "panic found in call handlers")
					})
				}
				wg.Done()
			}()
			if e := handler(); e != nil {
				once.Do(func() {
					err = e
				})
			}
		}(f)
	}
	wg.Wait()
	return err
}

// Go 封装更方便易用的异步调用方法，用于rpc handler内部启动goroutine执行异步逻辑，自动recover，上报监控日志。
// trpc handler函数return之后, ctx会cancel掉, 会将ctx中的Msg放回池中, 关联的模调信息和logger会释放掉
// 内部会复制一个新的ctx供handler使用，当前的方法返回error永远是nil，后续可能加入协程池等goroutine生命周期管理等手段，当资源不够时会返回err
// 启动异步任务时，也需要设置好timeout，控制好异步任务允许执行的最长时间，而不是放任goroutine无节制运行，防止协程泄露
func Go(ctx context.Context, timeout time.Duration, handler func(context.Context)) error {
	oldMsg := codec.Message(ctx)
	newCtx, newMsg := codec.WithNewMessage(detach(ctx))
	codec.CopyMsg(newMsg, oldMsg)
	newCtx, cancel := context.WithTimeout(newCtx, timeout)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				buf := make([]byte, PanicBufLen)
				buf = buf[:runtime.Stack(buf, false)]
				log.Errorf("[PANIC]%v\n%s\n", e, buf)
				report.PanicNum.Incr()
			}
			cancel()
		}()
		handler(newCtx)
	}()
	return nil
}

// ExpandEnv 寻找s中的 ${var} 并替换为环境变量的值，没有则替换为空，不解析 $var
//
// os.ExpandEnv会同时处理${var}和$var，配置文件中可能包含一些含特殊字符$的配置项，
// 如redisClient、mysqlClient的连接密码。
func ExpandEnv(s string) string {
	var buf []byte
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+2 < len(s) && s[j+1] == '{' { // 只匹配${var} 不匹配$var
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getEnvName(s[j+1:])
			if name == "" && w > 0 {
				// 非法匹配，去掉$
			} else if name == "" {
				buf = append(buf, s[j]) // 保留$
			} else {
				buf = append(buf, os.Getenv(name)...)
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		return s
	}
	return string(buf) + s[i:]
}

// getEnvName 获取环境变量名，即${var}里面的var内容，返回var内容及其长度
func getEnvName(s string) (string, int) {
	// 匹配右括号 }
	// 输入已经保证第一个字符是{，并且至少两个字符以上
	for i := 1; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\n' || s[i] == '"' { // "xx${xxx"
			return "", 0 // 遇到上面这些字符认为没有匹配中，保留$
		}
		if s[i] == '}' {
			if i == 1 { // ${}
				return "", 2 // 去掉${}
			}
			return s[1:i], i + 1
		}
	}
	return "", 0 // 没有右括号，保留$
}

// --------------- IP Config相关-----------------

// nicIP 记录网卡名对应的IP地址（包括ipv4和ipv6地址）
type nicIP struct {
	nic  string
	ipv4 []string
	ipv6 []string
}

// netInterfaceIP 记录本地所有网络接口对应的IP地址
type netInterfaceIP struct {
	once sync.Once
	ips  map[string]*nicIP
}

// enumAllIP 枚举本地网卡对应的ip地址
func (p *netInterfaceIP) enumAllIP() map[string]*nicIP {
	p.once.Do(func() {
		p.ips = make(map[string]*nicIP)
		interfaces, err := net.Interfaces()
		if err != nil {
			return
		}
		for _, i := range interfaces {
			p.addInterface(i)
		}
	})
	return p.ips
}

func (p *netInterfaceIP) addInterface(i net.Interface) {
	addrs, err := i.Addrs()
	if err != nil {
		return
	}
	for _, v := range addrs {
		ipNet, ok := v.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() != nil {
			p.addIPv4(i.Name, ipNet.IP.String())
		} else if ipNet.IP.To16() != nil {
			p.addIPv6(i.Name, ipNet.IP.String())
		}
	}
}

// addIPv4 append ipv4 address
func (p *netInterfaceIP) addIPv4(nic string, ip4 string) {
	ips := p.getNicIP(nic)
	ips.ipv4 = append(ips.ipv4, ip4)
}

// addIPv6 append ipv6 address
func (p *netInterfaceIP) addIPv6(nic string, ip6 string) {
	ips := p.getNicIP(nic)
	ips.ipv6 = append(ips.ipv6, ip6)
}

// getNicIP 获取网卡名对应的IP地址组
func (p *netInterfaceIP) getNicIP(nic string) *nicIP {
	if _, ok := p.ips[nic]; !ok {
		p.ips[nic] = &nicIP{nic: nic}
	}
	return p.ips[nic]
}

// getIPByNic 根据网卡名称返回ip地址
// 优先返回ipv4地址，如果没有ipv4地址，则返回ipv6地址
func (p *netInterfaceIP) getIPByNic(nic string) string {
	p.enumAllIP()
	if len(p.ips) <= 0 {
		return ""
	}
	if _, ok := p.ips[nic]; !ok {
		return ""
	}
	ip := p.ips[nic]
	if len(ip.ipv4) > 0 {
		return ip.ipv4[0]
	}
	if len(ip.ipv6) > 0 {
		return ip.ipv6[0]
	}
	return ""
}

// localIP 记录本地的网卡与IP的对应关系
var localIP = &netInterfaceIP{}

// GetIP 根据网卡名称返回IP地址
func GetIP(nic string) string {
	ip := localIP.getIPByNic(nic)
	return ip
}

// Deduplicate 将a和b中的字符串按顺序连在一起，且去重
func Deduplicate(a, b []string) []string {
	r := make([]string, 0, len(a)+len(b))
	m := make(map[string]bool)
	for _, s := range append(a, b...) {
		if _, ok := m[s]; !ok {
			m[s] = true
			r = append(r, s)
		}
	}
	return r
}
