// Package fc 提供007流控相关接口
package fc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/route"
)

var (
	ErrorFCOverflow = errors.New("fc over flow")
)

// Unit 流控基础数据
type Unit struct {
	Timestamp int64 `json:"time"`  // 时间戳
	Quote     int64 `json:"quota"` // 额度
}

// ReqInfo 流控接口请求参数
type ReqInfo struct {
	LogName   string
	Status    int
	Quote     int64
	TimeStamp int64
	NodeID    string
}

// RespInfo 流控接口返回值
type RespInfo struct {
	Code int     `json:"code"`
	Data []*Unit `json:"data"`
}

// GetFlowControl 拉取流控接口数据
func GetFlowControl(r route.Router, req *ReqInfo) (*RespInfo, error) {
	addr, err := r.GetAddr("Development", "trpc.m007.flowcontrol.getFlowInfo")
	if err != nil {
		return nil, err
	}
	URL, err := url.Parse(fmt.Sprintf("http://%s/getFlowInfo", addr))
	if err != nil {
		return nil, err
	}
	params := url.Values{
		"logname": []string{req.LogName},
		"status":  []string{fmt.Sprintf("%d", req.Status)},
		"lquota":  []string{fmt.Sprintf("%d", req.Quote)},
		"time":    []string{fmt.Sprintf("%d", req.TimeStamp)},
		"uuid":    []string{req.NodeID},
	}
	URL.RawQuery = params.Encode()
	client := http.Client{Timeout: 1 * time.Second}
	hrsp, err := client.Get(URL.String())
	if err != nil {
		return nil, err
	}
	defer hrsp.Body.Close()

	switch hrsp.StatusCode {
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("getFlowInfo fail statusCode:%d", hrsp.StatusCode)
	}

	body, err := ioutil.ReadAll(hrsp.Body)
	if err != nil {
		return nil, err
	}
	var result RespInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	switch result.Code {
	case 0:
	case 1:
		return nil, ErrorFCOverflow
	default:
		return nil, fmt.Errorf("getFlowInfo fail, code:%d", result.Code)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("getFlowInfo resp invalid")
	}

	return &result, nil
}
