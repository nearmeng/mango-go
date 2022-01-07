// +build !windows

package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/metrics"
)

var pid = os.Getpid()

func init() {
	// 启动runtime监控
	go func() {
		time.Sleep(time.Second * 3) // 等待框架启动完成
		for {
			// 每次都在每分钟的第30s左右上报数据，防止在分钟间隔出现0或者2倍的问题
			time.Sleep((time.Duration)(90-time.Now().Second()) * time.Second)
			RuntimeMetrics()
		}
	}()
}

// RuntimeMetrics runtime监控 每隔一分钟定时上报runtime详细信息
func RuntimeMetrics() {

	profiles := pprof.Profiles()
	for _, p := range profiles {
		switch p.Name() {
		case "goroutine":
			metrics.Gauge("trpc.GoroutineNum").Set(float64(p.Count()))
		case "threadcreate":
			metrics.Gauge("trpc.ThreadNum").Set(float64(p.Count()))
		default:
		}
	}

	metrics.Gauge("trpc.GOMAXPROCSNum").Set(float64(runtime.GOMAXPROCS(0)))
	metrics.Gauge("trpc.CPUCoreNum").Set(float64(runtime.NumCPU()))

	getMemStats()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	getFDs(ctx)
	getPidCount(ctx)
	getTcpSocket()
}

func getMemStats() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	var pauseNs uint64
	var pause100us, pause500us, pause1ms, pause10ms, pause50ms, pause100ms, pause500ms, pause1s, pause1sp int
	for _, ns := range mem.PauseNs {
		pauseNs += ns
		if ns < 100e3 {
			pause100us++
		} else if ns < 500e3 {
			pause500us++
		} else if ns < 1e6 {
			pause1ms++
		} else if ns < 10e6 {
			pause10ms++
		} else if ns < 50e6 {
			pause50ms++
		} else if ns < 100e6 {
			pause100ms++
		} else if ns < 500e6 {
			pause500ms++
		} else if ns < 1e9 {
			pause1s++
		} else {
			pause1sp++
		}
	}
	pauseNs /= uint64(len(mem.PauseNs))
	metrics.Gauge("trpc.PauseNsLt100usTimes").Set(float64(pause100us))
	metrics.Gauge("trpc.PauseNs100_500usTimes").Set(float64(pause500us))
	metrics.Gauge("trpc.PauseNs500us_1msTimes").Set(float64(pause1ms))
	metrics.Gauge("trpc.PauseNs1_10msTimes").Set(float64(pause10ms))
	metrics.Gauge("trpc.PauseNs10_50msTimes").Set(float64(pause50ms))
	metrics.Gauge("trpc.PauseNs50_100msTimes").Set(float64(pause100ms))
	metrics.Gauge("trpc.PauseNs100_500msTimes").Set(float64(pause500ms))
	metrics.Gauge("trpc.PauseNs500ms_1sTimes").Set(float64(pause1s))
	metrics.Gauge("trpc.PauseNsBt1sTimes").Set(float64(pause1sp))

	metrics.Gauge("trpc.AllocMem_MB").Set(float64(mem.Alloc) / 1024 / 1024)
	metrics.Gauge("trpc.SysMem_MB").Set(float64(mem.Sys) / 1024 / 1024)
	metrics.Gauge("trpc.NextGCMem_MB").Set(float64(mem.NextGC) / 1024 / 1024)
	metrics.Gauge("trpc.PauseNs_us").Set(float64(pauseNs / 1024))
	metrics.Gauge("trpc.GCCPUFraction_ppb").Set(mem.GCCPUFraction * 1000)
}

func getFDs(ctx context.Context) {
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err == nil {
		metrics.Gauge("trpc.MaxFdNum").Set(float64(limit.Cur))
	}

	out, err := exec.CommandContext(ctx, "bash", "-c", fmt.Sprintf("ls /proc/%d/fd | wc -l", pid)).Output()
	if err != nil {
		return
	}
	num, err := strconv.Atoi(strings.Trim(string(out), " \n\t"))
	if err != nil {
		return
	}
	metrics.Gauge("trpc.CurrentFdNum").Set(float64(num))
}

func getPidCount(ctx context.Context) {
	shell := fmt.Sprintf("ps -eLF|wc -l")
	out, err := exec.CommandContext(ctx, "bash", "-c", shell).Output()
	if err != nil {
		return
	}
	pidNum, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return
	}
	metrics.Gauge("trpc.PidNum").Set(pidNum)
}

func getTcpSocket() {
	///proc/net/sockstat
	st, err := os.Open("/proc/net/sockstat")
	if err != nil {
		return
	}
	data := make([]byte, 50)
	c, err := st.Read(data)
	if err != nil || c == 0 {
		return
	}
	stats := string(data[:func() int {
		for i, s := range data {
			if s == '\n' {
				return i
			}
		}
		return 0
	}()])
	sum, err := strconv.ParseFloat(strings.Split(stats, " ")[2], 64)
	if err != nil {
		return
	}
	metrics.Gauge("trpc.TcpNum").Set(sum)
}
