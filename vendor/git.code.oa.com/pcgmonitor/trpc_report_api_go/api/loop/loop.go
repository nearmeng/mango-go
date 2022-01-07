// Package loop 循环
package loop

import (
	"log"
	"runtime"
	"time"
)

// Looper 循环上报interface
type Looper interface {
	Action() error           // 具体动作
	Interval() time.Duration // 周期
	Trigger() chan int       // 支持外部主动促发
}

// Start 启动loopReporter，单独起goroutinue执行
func Start(looper Looper, interval time.Duration) {
	defer func() {
		// 捕获panic，避免影响外部业务
		if err := recover(); err != nil {
			buf := make([]byte, 1024)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("trpc_report_api_go: [PANIC]%v\n%s\n", err, buf)
			go Start(looper, interval)
		}
	}()

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		err := logic(looper, timer)
		if err != nil {
			log.Printf("trpc_report_api_go: logic err:%v", err)
		}
	}
}

// logic 周期执行逻辑
func logic(looper Looper, timer *time.Timer) error {
	select {
	case <-timer.C:
		// 周期执行
		timer.Reset(looper.Interval())
		return looper.Action()
	case <-looper.Trigger():
		// 外部促发执行
		return looper.Action()
	}
}
