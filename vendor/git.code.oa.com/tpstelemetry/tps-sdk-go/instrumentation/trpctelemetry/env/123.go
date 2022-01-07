//

// Copyright 2020 The TpsTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Copyright 2020 The TpsTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package env 提供应用运行环境信息
package env

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hanjm/etcd/clientv3"
	"google.golang.org/grpc"
)

var (
	// defaultDialTimeout default dail timeout
	defaultDialTimeout = time.Second * 5
	// etcdEndPoint 保存appcode和appsecret的etcd endpoints
	etcdEndPoint = []string{"registry.tpstelemetry.woa.com:2379"}
)

const (
	// appSecretKey 保存123平台appcode信息的key
	appSecretKey = "/tpstelemetry/123appsecret"
)

// is123 判断机器是否为123平台
func is123() bool {
	return strings.HasPrefix(os.Getenv("SUMERU_CLUSTER_NAME"), "cls-")
}

func get123Info() (*ServerInfo, error) {
	appCode, appSecret, err := getAppSercet()
	if err != nil {
		log.Printf("get app code err:%v", err)
		return nil, err
	}

	app := os.Getenv("SUMERU_APP")
	server := os.Getenv("SUMERU_SERVER")
	if app == "" || server == "" {
		return nil, errors.New("invalid app or server")
	}
	moduleInfo, err := getModule(&queryModuleReq{
		AppCode:   appCode,
		AppSecret: appSecret,
		Req: struct {
			App    string `json:"app"`
			Server string `json:"server"`
			Env    string `json:"env"`
		}{
			App:    app,
			Server: server,
		},
	})
	if err != nil {
		log.Printf("get 123 module info err:%v", err)
		return nil, err
	}
	info := &ServerInfo{}
	info.CmdbID = moduleInfo.Data.VModuleInfoList[0].CmdbID
	owner, err := getOwner(moduleInfo)
	if err != nil {
		log.Printf("get owner info err:%v", err)
		return nil, err
	}
	info.Owner = owner
	return info, nil
}

// getAppSercet 获取用于从123平台拉服务信息的app code和app sercet
func getAppSercet() (string, string, error) {
	cli, err := newEtcdCli()
	if err != nil {
		return "", "", err
	}
	defer func() {
		_ = cli.Close()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rsp, err := cli.Get(ctx, appSecretKey)
	if err != nil {
		return "", "", err
	}
	if rsp.Count != 1 || len(rsp.Kvs) != 1 {
		return "", "", errors.New("appcode count invalid")
	}
	type appInfo struct {
		AppCode   string `json:"app_code"`
		AppSecret string `json:"app_secret"`
	}
	kv := rsp.Kvs[0]
	appCode := &appInfo{}
	err = json.Unmarshal(kv.Value, appCode)
	if err != nil {
		return "", "", err
	}
	return appCode.AppCode, appCode.AppSecret, nil
}

// newEtcdCli new etcd cli
func newEtcdCli() (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndPoint,
		DialTimeout: defaultDialTimeout,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	return client, err
}

// queryModuleReq 请求123平台接口/prod/admin/queryModule的请求体
type queryModuleReq struct {
	AppCode   string `json:"app_code"`
	AppSecret string `json:"app_secret"`
	Req       struct {
		App    string `json:"app"`
		Server string `json:"server"`
		Env    string `json:"env"`
	} `json:"req"`
}

// queryModuleRsp 请求123平台接口/prod/admin/queryModule的回包
type queryModuleRsp struct {
	Ret  int `json:"ret"`
	Data struct {
		VModuleInfoList []modelInfo `json:"vModuleBaseInfo"`
		Total           int         `json:"total"`
	} `json:"data"`
}

type modelInfo struct {
	App        string `json:"app"`
	Server     string `json:"server"`
	Developer  string `json:"developer"`
	Maintainer string `json:"maintainer"`
	CreateUser string `json:"createUser"`
	UpdateUser string `json:"updateUser"`
	CmdbID     string `json:"cmdbId"`
}

// getModule 123平台接口可参考http://open.oa.com/esb/docs/api_gateway/paasadminserver/queryEnvModule/
func getModule(req *queryModuleReq) (*queryModuleRsp, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	queryURL := "http://paasadminserver.apigw.o.oa.com/prod/admin/queryModule"
	reqHTTP, err := http.NewRequest(http.MethodPost, queryURL, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}
	rspHTTP, err := client.Do(reqHTTP)
	if rspHTTP != nil {
		defer rspHTTP.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if rspHTTP.StatusCode >= 400 {
		return nil, fmt.Errorf("httpStatusCode: %d, httpStatus: %s", rspHTTP.StatusCode, rspHTTP.Status)
	}
	rspData, err := ioutil.ReadAll(rspHTTP.Body)
	if err != nil {
		return nil, err
	}
	if len(rspData) == 0 {
		return nil, errors.New("missing return body")
	}
	queryModuleRsp := &queryModuleRsp{}
	err = json.Unmarshal(rspData, queryModuleRsp)
	if err != nil {
		return nil, err
	}
	if queryModuleRsp.Ret != 0 {
		return nil, fmt.Errorf("ret: %d, msg", queryModuleRsp.Ret)
	}
	if len(queryModuleRsp.Data.VModuleInfoList) == 0 {
		return nil, errors.New("nil rsp vModuleBaseInfo")
	}
	return queryModuleRsp, nil
}

// getOwner 获取owner的业务逻辑
func getOwner(queryModuleRsp *queryModuleRsp) (string, error) {
	if len(queryModuleRsp.Data.VModuleInfoList) == 0 {
		return "", errors.New("nil rsp vModuleBaseInfo")
	}
	var allOwners []string
	developers := strings.Split(queryModuleRsp.Data.VModuleInfoList[0].Developer, ";")
	allOwners = append(allOwners, developers...)
	maintainers := strings.Split(queryModuleRsp.Data.VModuleInfoList[0].Maintainer, ";")
	allOwners = append(allOwners, maintainers...)
	allOwners = removeDuplicateAndInvalidStrings(allOwners)
	ownerResult := strings.Join(allOwners, `;`)
	return ownerResult, nil
}

// removeDuplicateAndInvalidStrings 去除string slice中重复元素以及NONE
// 背景：如果某类型用户为空，123平台接口会返回NONE。这种非法的用户名应该将其去除
func removeDuplicateAndInvalidStrings(owners []string) []string {
	result := make([]string, 0, len(owners))
	temp := map[string]struct{}{}
	for _, owner := range owners {
		// 不要加入非法的用户名
		if !isOwnerNameValid(owner) {
			continue
		}
		if _, ok := temp[owner]; !ok {
			temp[owner] = struct{}{}
			result = append(result, owner)
		}
	}
	return result
}

// 合法的owner英文名
var ownerReg = regexp.MustCompile("(^[a-z]+$)|(^([a-z]_)?[a-z]+$)")

func isOwnerNameValid(owner string) bool {
	if owner == "" {
		return false
	}
	if !ownerReg.MatchString(owner) {
		return false
	}
	return true
}
