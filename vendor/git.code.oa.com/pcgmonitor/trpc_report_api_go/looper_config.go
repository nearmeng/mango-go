package pcgmonitor

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/config"
)

var (
	errConfigNoUpdate = fmt.Errorf("config not update")
)

// configLooper 远程配置周期更新
type configLooper struct {
	active chan int
	inst   *Instance
}

// Action 循环具体动作
func (l *configLooper) Action() error {
	return l.updateConfig(l.getReqBody())
}

// Interval 循环间隔
func (l *configLooper) Interval() time.Duration {
	// 做个保护，避免太频繁拉取
	interval := 1 * time.Minute
	t := l.inst.remoteConfig()
	// 根据http返回的interval来设置下一次拉取配置的间隔时间
	if t.attaInfo.GetConfigInterval >= 1*60*1000 {
		interval = time.Millisecond * time.Duration(t.attaInfo.GetConfigInterval)
	}
	return interval
}

// Trigger 外部是否主动促发
func (l *configLooper) Trigger() chan int {
	return l.active
}

// remoteConfig 调用Http接口拉取远程配置
func (l *configLooper) updateConfig(reqBody *config.ReqBody) error {
	remoteConf, err := l.getRemoteConf(reqBody)
	if err != nil && err != errConfigNoUpdate {
		log.Printf("获取中控信息失败:%s", err.Error())
		return err
	}

	if err == errConfigNoUpdate {
		return nil
	}

	// 更新数据, copy and write, 无锁
	configNew := &remoteConfig{}
	configNew.prefixInfo = &prefixInfo{
		prefix:               remoteConf.Data.Prefix,
		prefixMetrics:        remoteConf.Data.Prefix + "m",
		prefixActiveModCall:  remoteConf.Data.Prefix + "a",
		prefixPassiveModCall: remoteConf.Data.Prefix + "p",
		prefixCustom:         remoteConf.Data.Prefix + "c",
	}
	configNew.attaInfo = remoteConf.Data
	configNew.configVersion = remoteConf.ConfigVersion
	configNew.details = make(map[string]*config.Detail)
	for _, d := range remoteConf.Detail {
		configNew.details[d.Logname] = d
	}

	l.inst.withRemoteConfig(configNew)
	if len(remoteConf.HawkLogConfigs.Detail) == 0 {
		return nil
	}
	// 更新日志相关配置
	hawkLogConfNew := &hawkLogConfig{}
	hawkLogConfNew.hawkLogConfig = make(map[string]*config.HawkLogConfig)
	for _, item := range remoteConf.HawkLogConfigs.Detail {
		// 需要单独发送到其他attaid的日志级别
		separateSendLogMap := make(map[int]*config.LogLevelAttaInfo, len(item.Subs))
		for _, logAttaInfo := range item.Subs {
			separateSendLogMap[logAttaInfo.Level] = logAttaInfo
		}
		item.SubMaps = separateSendLogMap
		hawkLogConfNew.hawkLogConfig[item.LogName] = item
	}
	l.inst.withHawkLogConfig(hawkLogConfNew)
	return nil
}

// getRemoteConf 拉取远程配置
func (l *configLooper) getRemoteConf(reqBody *config.ReqBody) (*config.RspInfo, error) {
	err := l.isInstValid()
	if err != nil {
		log.Printf("trpc_report_api_go:isInstValid err:%v", err)
		return nil, err
	}

	curConfig := l.inst.remoteConfig()
	configVersion := curConfig.configVersion

	url, err := l.geneConfURL(configVersion)
	if err != nil {
		log.Printf("trpc_report_api_go:geneConfURL error:%v", err)
		return nil, err
	}
	code, timeMs := 0, 0
	defer l.inst.selfMonitorLooper.addGetConfigSelfMonitor(&configVersion, &code, &timeMs) // 中控自监控上报

	var result *config.RspInfo
	result, code, timeMs, err = l.getRemoteConfBiz(url, reqBody)
	if err != nil {
		return nil, err
	}

	// 配置无更新，直接return
	if result.Code == 1 {
		// 拉配置自监控上报
		// 配置无更新是不会返回configVersion的，直接取之前的configVersion来上报
		code = result.Code
		return nil, errConfigNoUpdate
	}

	if result.Data == nil {
		code = -100
		return nil, fmt.Errorf("config result data null")
	}

	configVersion = result.ConfigVersion
	code = result.Code

	return result, nil
}

// getRemoteConfBiz 实际拉取远程配置
func (l *configLooper) getRemoteConfBiz(url string, reqBody *config.ReqBody) (*config.RspInfo, int, int, error) {
	var code, timeMs int

	begin := time.Now()
	client := http.Client{Timeout: 2 * time.Second}
	var postBody io.Reader
	logBodyJson, err := json.Marshal(reqBody)

	if len(logBodyJson) > 0 {
		postBody = strings.NewReader(string(logBodyJson))
	} else {
		postBody = nil
	}
	rsp, err := client.Post(url, "application/json", postBody)
	timeMs = int(time.Since(begin) / time.Millisecond)
	if err != nil {
		l.inst.reportLog(levelError, l.inst.debugLogOpen, "trpc_report_api_go:get conf error[%v], "+
			"costMs[%d], url[%s]", err, timeMs, url)
		code = -200
		return nil, code, timeMs, err
	}
	defer rsp.Body.Close()

	code = rsp.StatusCode
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		l.inst.reportLog(levelError, l.inst.debugLogOpen, "trpc_report_api_go:get conf"+
			" read resp error[%v], url[%s]", err, url)
		code = -1
		return nil, code, timeMs, err
	}

	l.inst.reportLog(levelDebug, l.inst.debugLogOpen, "trpc_report_api_go:get conf "+
		"succ, url[%s], res[%s]", url, string(body))

	// 解析字符串为Json
	var result config.RspInfo
	err = json.Unmarshal(body, &result)
	if err != nil {
		l.inst.reportLog(levelError, l.inst.debugLogOpen, "trpc_report_api_go:conf unmarshal "+
			"json error[%v], url[%s]", err, url)
		code = -2
		return nil, code, timeMs, err
	}

	return &result, code, timeMs, nil
}

// geneConfURL 远程配置地址
func (l *configLooper) geneConfURL(configVersion int) (string, error) {
	// 北极星寻址
	address, err := l.inst.polaris.GetAddr("Production", "64939329:131073")
	if err != nil {
		return "", err
	}

	appServerName := l.inst.frameSvrInfo.App + "." + l.inst.frameSvrInfo.Server
	frameCode := l.inst.frameSvrInfo.FrameCode
	if l.inst.svrInfoType != svrInfoFrame {
		appServerName = l.inst.commSvrInfo.CommName
		frameCode = commReportFrameCode
	}
	url := fmt.Sprintf("http://%s/APIConf/getConfigInfo?appServerName=%s&frameCode=%s&configVersion=%d",
		address, appServerName, frameCode, configVersion)
	return url, nil
}

// isInstValid 实例参数检查
func (l *configLooper) isInstValid() error {
	if l.inst == nil {
		return fmt.Errorf("instance is null")
	}

	if (l.inst.svrInfoType == svrInfoFrame && l.inst.frameSvrInfo.App == "") ||
		(l.inst.svrInfoType == svrInfoComm && l.inst.commSvrInfo.CommName == "") {
		return fmt.Errorf("instance is invalid")
	}

	return nil
}

// getReqBody 获取中控请求体
func (l *configLooper) getReqBody() *config.ReqBody {
	sdkConfList := make([]*config.HawkLogSdk, 0)
	logConf := l.inst.getHawkLogConfig()
	for _, hawkLogConfig := range logConf.hawkLogConfig {
		sdkConf := &config.HawkLogSdk{
			Version:       hawkLogConfig.Version,
			Name:          hawkLogConfig.LogName,
			DimensionsNum: hawkLogConfig.DimensionsNum,
		}
		sdkConfList = append(sdkConfList, sdkConf)
	}
	return &config.ReqBody{
		ConfigInfoLogBody: &config.HawkLogBody{
			Params: sdkConfList,
		},
	}
}

// initNewLog 初始化新的日志
func (l *configLooper) initNewLog() {
	sdkConfList := make([]*config.HawkLogSdk, 0, len(l.inst.hawkLogNames))
	for _, logName := range l.inst.hawkLogNames {
		sdkConf := &config.HawkLogSdk{
			Name:          logName,
			Version:       0,
			DimensionsNum: 0,
		}
		sdkConfList = append(sdkConfList, sdkConf)
	}
	// 拉取远端配置，有兜底配置，忽略失败
	_ = l.updateConfig(&config.ReqBody{
		ConfigInfoLogBody: &config.HawkLogBody{
			Params: sdkConfList,
		},
	})
}
