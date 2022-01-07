package runtime

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/metrics"
	"git.code.oa.com/trpc-go/trpc-go/plugin"

	trpc "git.code.oa.com/trpc-go/trpc-go"
)

// InStatDomainName 内网统计域名
var InStatDomainName = "stat.trpc.oa.com"

// OutStatDomainName 外网统计域名
var OutStatDomainName = "stat.trpc.tencent.com"

// ReportInterval 上报间隔
var ReportInterval = time.Hour * 24

func init() {
	// 启动统计数据监控，每隔24小时上报一次
	go StatReport(trpc.Version(), "")
}

// StatReport 每隔24小时上报统计数据
func StatReport(version string, plug string) {

	time.Sleep(time.Second * 3)
	plugin.WaitForDone(time.Minute) // 等待框架启动完成

	for {
		err := Stat(InStatDomainName, version, plug)
		if err != nil {
			metrics.Counter("InStatReportFail").Incr()
			err = Stat(OutStatDomainName, version, plug)
			if err != nil {
				metrics.Counter("OutStatReportFail").Incr()
			}
		}

		time.Sleep(ReportInterval)
	}
}

// Stat 数据上报接口
func Stat(domain string, version string, plugin string) error {

	// http://stat.trpc.tencent.com/report?app=xx&server=xx&ip=10.100.1.2&version=v0.1.0-rc.1&lang=go&container=xx
	body := fmt.Sprintf(`{"app":"%s", "server":"%s", "ip":"%s", "container":"%s", "lang":"go", "version":"%s", "plugin":"%s"}`,
		trpc.GlobalConfig().Server.App,
		trpc.GlobalConfig().Server.Server,
		trpc.GlobalConfig().Global.LocalIP,
		trpc.GlobalConfig().Global.ContainerName,
		version,
		plugin,
	)
	rsp, err := http.Post(fmt.Sprintf("http://%s/api/trpc/stat/add", domain), "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	rsp.Body.Close()

	return nil
}
