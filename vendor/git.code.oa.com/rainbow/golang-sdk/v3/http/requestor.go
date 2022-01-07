package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"git.code.oa.com/going/l5"
	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/version"
	v3 "git.code.oa.com/rainbow/proto/api/configv3"
	"github.com/golang/protobuf/jsonpb"
)

// Requestor 请求
type Requestor struct {
	types.AddressBase
	sdkopts      types.InitOptions // 本地sdk参数
	proxyPull    *http.Client
	proxyPolling *http.Client
	sgnp         *types.SignatureParams
	consumer     api.ConsumerAPI
	flowID       uint64
}

// Init 初始化
func (r *Requestor) Init(o types.InitOptions) (err error) {
	err = r.ParseAddress(o.ConnectStr)
	if err != nil {
		return err
	}
	if r.Type == "polaris" {
		configuration := api.NewConfiguration()
		configuration.GetGlobal().GetServerConnector().SetConnectTimeout(o.TimeoutConnNaming)
		configuration.GetGlobal().GetServerConnector().SetMessageTimeout(o.TimeoutNaming)
		r.consumer, err = api.NewConsumerAPIByConfig(configuration)
		if err != nil {
			return err
		}
	}
	r.sdkopts = o
	r.proxyPull = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   o.TimeoutCS,
				KeepAlive: o.TimeoutCS + 3*time.Second,
			}).Dial,
			TLSHandshakeTimeout: o.TimeoutCS,
			IdleConnTimeout:     3 * time.Second,
		},
		Timeout: o.TimeoutCS,
	}
	r.proxyPolling = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   o.TimeoutPolling,
				KeepAlive: o.TimeoutPolling + 3*time.Second,
			}).Dial,
			TLSHandshakeTimeout: o.TimeoutCS,
			IdleConnTimeout:     3 * time.Second,
		},
		Timeout: o.TimeoutPolling + 1*time.Second,
	}
	r.sgnp = types.DefaultNew(r.sdkopts.AppID, r.sdkopts.UserID, r.sdkopts.HmacWay)
	err = r.sgnp.SignedString([]byte(r.sdkopts.UserKey), nil)
	if err != nil {
		return fmt.Errorf("SignedString Error=%s", err.Error())
	}
	return nil
}

// Getdatas  拉取配置
func (r *Requestor) Getdatas(ctx context.Context, req *v3.ReqGetDatas,
	getOpts *types.GetOptions) (rsp *v3.RspGetDatas, err error) {
	urlSuffix := "/rainbowapi.configs/getdatas"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body, err := r.httpRequest(r.proxyPull, urlSuffix, reqBody, getOpts)
	if err != nil {
		return nil, err
	}
	rsp = &v3.RspGetDatas{}
	var Unmarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
	err = Unmarshaler.Unmarshal(strings.NewReader(string(body)), rsp)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

// Poll 长轮询
func (r *Requestor) Poll(ctx context.Context, req *v3.ReqPoll,
	getOpts *types.GetOptions) (rsp *v3.RspPoll, err error) {
	urlSuffix := "/rainbowapi.configs/poll"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body, err := r.httpRequest(r.proxyPolling, urlSuffix, reqBody, getOpts)
	if err != nil {
		return nil, err
	}
	rsp = &v3.RspPoll{}
	var Unmarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
	err = Unmarshaler.Unmarshal(strings.NewReader(string(body)), rsp)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

// Heartbeat 心跳
/*func (r *Requestor) Heartbeat(ctx context.Context, req *config.ReqHeartbeat,
	getOpts *types.GetOptions) (rsp *config.RspHeartbeat, err error) {
	urlSuffix := "/config.v2.ConfigService/Heartbeat"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body, err := r.httpRequest(r.proxyPolling, urlSuffix, reqBody, getOpts)
	if err != nil {
		return nil, err
	}
	rsp = &config.RspHeartbeat{}
	var Unmarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
	err = Unmarshaler.Unmarshal(strings.NewReader(string(body)), rsp)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}*/

func (r *Requestor) httpRequest(proxy *http.Client, urlSuffix string, reqBody []byte,
	getOpts *types.GetOptions) ([]byte, error) {
	userKey := r.sdkopts.UserKey
	url, l5Node := r.selectAddress()
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}
	url += urlSuffix
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(version.Header, version.Version)
	if r.sdkopts.OpenSign {
		var sgnp *types.SignatureParams
		// 本次请求带有签名
		if getOpts.UserID != "" && getOpts.UserKey != "" {
			sgnp = types.DefaultNew(getOpts.AppID, getOpts.UserID, getOpts.HmacWay)
			userKey = getOpts.UserKey
		} else { // 没带用全局的
			sgnp = types.DefaultNew(r.sdkopts.AppID, r.sdkopts.UserID, r.sdkopts.HmacWay)
		}
		err = sgnp.SignedString([]byte(userKey), nil)
		if err != nil {
			return nil, fmt.Errorf("SignedString Error=%s", err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("rainbow_sgn_type", sgnp.SgnType)
		req.Header.Set("rainbow_version", sgnp.Version)
		req.Header.Set("rainbow_app_id", sgnp.AppID)
		req.Header.Set("rainbow_user_id", sgnp.UserID)
		req.Header.Set("rainbow_timestamp", sgnp.Timestamp)
		req.Header.Set("rainbow_nonce", sgnp.Nonce)
		req.Header.Set("rainbow_sgn_method", sgnp.SgnMethod)
		req.Header.Set("rainbow_sgn_body", sgnp.SgnBody)
		req.Header.Set("rainbow_signature", sgnp.Signature)
	}
	timing := time.Now()
	var (
		rsp    *http.Response
		netErr error
	)
	defer func() {
		cost := time.Now().Sub(timing)
		r.report(l5Node, cost, netErr)
	}()
	rsp, netErr = proxy.Do(req)
	if netErr != nil {
		return nil, netErr
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("code=%d, %s", rsp.StatusCode, string(body))
	}
	return body, nil
}

// selectAddress 获取地址
func (r *Requestor) selectAddress() (string, interface{}) {
	var err error
	var node *l5.Server
	if r.Type == "http" {
		return r.Addr, nil
	}
	if r.Type == "ip" {
		return string("http://") + r.IPList[rand.Int()%len(r.IPList)], nil
	}
	if r.Type == "cl5" {
		for i := 0; i < 3; i++ {
			node, err = l5.ApiGetRoute(int32(r.ModID), int32(r.CmdID))
			if err != nil {
				// that's good
				continue
			}
			prefix := string("http://") + node.Ip() + ":"
			prefix += strconv.FormatInt(int64(node.Port()), 10)
			return prefix, node
		}
	}
	if r.Type == "polaris" {
		getInstancesReq := &api.GetOneInstanceRequest{}
		getInstancesReq.FlowID = atomic.AddUint64(&r.flowID, 1)
		getInstancesReq.Namespace = "Production"
		getInstancesReq.Service = r.Addr
		getInstResp, err := r.consumer.GetOneInstance(getInstancesReq)
		if err != nil {
			log.Errorf("%v", err)
			return "", nil
		}
		targetInstance := getInstResp.Instances[0]
		prefix := fmt.Sprintf("http://%s:%d", targetInstance.GetHost(), targetInstance.GetPort())
		return prefix, targetInstance
	}
	log.Errorf("selectAddress failed addr=%s:%v", r.Addr, err)
	return "", nil
}

// report cl5
func (r *Requestor) report(n interface{}, cost time.Duration, success error) error {
	var result int32
	if n == nil {
		return nil
	}
	switch r.Type {
	case "cl5":
		node := n.(*l5.Server)
		if success != nil {
			result = -1
		}
		err := l5.ApiRouteResultUpdate(node, result, uint64(cost.Nanoseconds()/1e6))
		if err != nil {
			return fmt.Errorf("cl5 ApiRouteResultUpdate %v", err)
		}
	case "polaris":
		svcCallResult := &api.ServiceCallResult{}
		svcCallResult.SetCalledInstance(n.(model.Instance))
		svcCallResult.SetRetStatus(api.RetSuccess)
		if success != nil {
			svcCallResult.SetRetStatus(api.RetFail)
		}
		svcCallResult.SetRetCode(0)
		svcCallResult.SetDelay(cost)
		err := r.consumer.UpdateServiceCallResult(svcCallResult)
		if err != nil {
			return fmt.Errorf("fail to UpdateServiceCallResult, err is %v", err)
		}
	}
	return nil
}
