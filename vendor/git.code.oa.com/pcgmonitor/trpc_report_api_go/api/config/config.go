// Package config 远端配置，配置文档见https://iwiki.woa.com/pages/viewpage.action?pageId=98470173
package config

import (
	"fmt"
)

// Data 服务粒度配置
type Data struct {
	Prefix            string `json:"PREFIX"`              // 前缀
	AttaID            string `json:"ATTA_ID"`             // 业务attaID
	AttaToken         string `json:"ATTA_TOKEN"`          // 业务attaToken
	TmMasterName      string `json:"TM_MASTER_NAME"`      // 拉取配置自监控名
	TmReportName      string `json:"TM_REPORT_NAME"`      // 上报自监控名
	TmAttaName        string `json:"TM_ATTA_NAME"`        // 发送ATTA自监控名
	TmHawkLogName     string `json:"TM_LOG_NAME"`         // 007日志自监控名
	TmAttaID          string `json:"TM_ATTA_ID"`          // 自监控AttaId
	TmAttaToken       string `json:"TM_ATTA_TOKEN"`       // 自监控AttaToken
	SendNum           int    `json:"SEND_NUM"`            // 汇聚前的条数限制
	SendInterval      int    `json:"SEND_INTERVAL"`       // 发送的最长间隔
	SendPerSize       int    `json:"SEND_PER_SIZE"`       // 汇聚后的条数限制
	GetConfigInterval int    `json:"GET_CONFIG_INTERVAL"` // 拉配置的时间间隔
	TmInterval        int    `json:"TM_INTERVAL"`         // 自监控上报的时间间隔
	LogAttaID         string `json:"LOG_ATTA_ID"`         // 远程日志attaID
	LogAttaToken      string `json:"LOG_ATTA_TOKEN"`      // 远程日志attaToken
	LogOpen           bool   `json:"LOG_OPEN"`            // 远程日志开关
	BizAttaID         string `json:"BUSINESS_ATTA_ID"`    // 业务数据告警attaID
	BizAttaToken      string `json:"BUSINESS_ATTA_TOKEN"` // 业务数据告警attaToken
	SamplingRate      int    `json:"SAMPLING_RATE"`       // 采样率
}

// TimeInterval 时间区间
type TimeInterval struct {
	Index int   `json:"index"`
	Point []int `json:"point"`
}

// Detail 监控项粒度配置
type Detail struct {
	Logname      string          `json:"logname"`
	Ignore       []int           `json:"ignore"`
	TimeInterval []*TimeInterval `json:"time_interval"`
	AttaID       string          `json:"atta_id"`
	AttaToken    string          `json:"atta_token"`
}

// HawkLogConfig 中控获取的日志字段信息
type HawkLogConfig struct {
	Version       int                       `json:"version"`
	LogName       string                    `json:"name"`
	DimensionsNum int                       `json:"dimensionsNum"`
	SamplingRate  int                       `json:"samplingRate"`
	Ignore        []int                     `json:"ignore"`
	AttaID        string                    `json:"attaId"`
	AttaToken     string                    `json:"attaToken"`
	Subs          []*LogLevelAttaInfo       `json:"subs"`
	SubMaps       map[int]*LogLevelAttaInfo `json:"sub_maps"`
}

// LogLevelAttaInfo 需要单独上报的日志级别 对应的atta信息
type LogLevelAttaInfo struct {
	Level     int    `json:"logLevel"`
	AttaID    string `json:"attaId"`
	AttaToken string `json:"attaToken"`
}

// HawkLogDetail 日志详细
type HawkLogDetail struct {
	Detail []*HawkLogConfig `json:"detail"`
}

// RspInfo 007中控接口返回
type RspInfo struct {
	Code           int            `json:"code"`          // 0表示data中的字段有更新，1表示data中的字段无更新
	ConfigVersion  int            `json:"configVersion"` // 配置版本号。如果配置版本号无变动说明配置参数无更新，后台返回code为1
	Message        string         `json:"msg"`           // success表示成功拉取
	Data           *Data          `json:"data"`          // 整体服务粒度的数据
	Detail         []*Detail      `json:"detail"`        // 监控项粒度的数据
	HawkLogConfigs *HawkLogDetail `json:"log"`           // 日志相关信息
}

// HawkLogBody 日志请求体
type HawkLogBody struct {
	Params []*HawkLogSdk `json:"params"`
}

// ReqBody 007中控请求体
type ReqBody struct {
	ConfigInfoLogBody *HawkLogBody `json:"configInfoLogBody"` // 日志请求体
	// 如需其他请求体在后面添加 即可
}

// HawkSdk 日志请求体具体需要传的字段
type HawkLogSdk struct {
	Version       int    `json:"version"`
	Name          string `json:"name"`
	DimensionsNum int    `json:"dimensionsNum"`
}

const (
	defaultTimeInternal = "0"
)

// DefaultData 配置默认值
func DefaultData() *Data {
	return &Data{
		Prefix:            "pp_tr",
		AttaID:            "04d00000105",
		AttaToken:         "9151711544",
		TmMasterName:      "tm_api_master",
		TmReportName:      "tm_api_report",
		TmAttaName:        "tm_api_atta",
		TmAttaID:          "0c200006178",
		TmAttaToken:       "1926741726",
		SendNum:           1000,
		SendInterval:      10 * 1000,
		SendPerSize:       100,
		GetConfigInterval: 10 * 60 * 1000,
		TmInterval:        10 * 60 * 1000,
		LogAttaID:         "0a300015838",
		LogAttaToken:      "3449568377",
		LogOpen:           true,
		BizAttaID:         "0c500015042",
		BizAttaToken:      "1398647159",
		SamplingRate:      100,
	}
}

// GetTimeInternal 服务耗时转化时间区间
func GetTimeInternal(logName string, costMs float64, details map[string]*Detail) string {
	point, err := getPoint(logName, 1, details) // 模调index固定是1
	if err != nil {
		return defaultTimeInternal
	}

	if len(point) == 0 || costMs >= float64(point[len(point)-1]) {
		return fmt.Sprintf("%d", len(point))
	}

	result := 0
	for i, time := range point {
		if costMs < float64(time) {
			result = i
			break
		}
	}

	return fmt.Sprintf("%d", result)
}

// getPoint 获取监控项对应的时间区间点
func getPoint(logName string, index int, details map[string]*Detail) ([]int, error) {
	detail, ok := details[logName]
	if !ok || len(detail.TimeInterval) == 0 {
		return nil, fmt.Errorf("logName:%s no detail conf", logName)
	}

	var point []int
	for _, timeInternal := range detail.TimeInterval {
		if timeInternal.Index == index {
			point = timeInternal.Point
			break
		}
	}

	if len(point) == 0 {
		return nil, fmt.Errorf("logName:%s index：%d， no timeInternal", logName, index)
	}

	return point, nil
}
