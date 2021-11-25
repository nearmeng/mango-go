package signal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nearmeng/mango-go/plugin/log"
)

// 之前版本注册的信号
// syscall.SIGTERM  程序退出
// syscall.SIGUSR1 灰度更新
// syscall.SIGUSR2  reload svr conf
// syscall.SIGUSR3  reload databin

const (
	SIGUSR3 = syscall.Signal(35)
)

// Handler 信号处理函数.
type Handler func()

var _sigHandlers = make(map[os.Signal]Handler)

// RegisterSignalHandler 注册多个信号的处理函数.
func RegisterSignalHandler(sigs []os.Signal, handler Handler) {
	for _, sig := range sigs {
		_sigHandlers[sig] = handler
	}
}

// StartSignal 启动信号监听协程.
func StartSignal() {
	// 创建监听退出chan
	c := make(chan os.Signal, 10) //nolint:gomnd
	// 监听指定信号 ctrl+c kill

	if len(_sigHandlers) == 0 {
		log.Error("Signal handlers empty.")
		return
	}

	sigs := make([]os.Signal, 0, len(_sigHandlers))
	for sig := range _sigHandlers {
		sigs = append(sigs, sig)
	}

	signal.Notify(c, sigs...)
	log.Info("Listen signals=%v", sigs)

	go func() {
		for s := range c {
			log.Info("Recv signal=%s", s.String())
			if handler, ok := _sigHandlers[s]; ok {
				handler()
			} else {
				log.Info("Recv signal=%s no handler", s.String())
			}
		}
	}()
}
