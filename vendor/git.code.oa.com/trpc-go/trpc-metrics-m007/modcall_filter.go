package m007

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	pcgmonitor "git.code.oa.com/pcgmonitor/trpc_report_api_go"
	trpc "git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
)

// ActiveModuleCallClientFilter  主调模调上报拦截器:自身调用下游，下游回包时上报
func ActiveModuleCallClientFilter(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) error {

	begin := time.Now()

	err := handler(ctx, req, rsp)

	msg := trpc.Message(ctx)

	if !isCalleeComplete(msg) { // 下游字段不全，则不上报007
		return err
	}

	activeMsg := new(pcgmonitor.ActiveMsg)
	// 自身服务
	activeMsg.AService = msg.CallerService()  // 主调Service
	activeMsg.AInterface = msg.CallerMethod() // 主调Interface
	if msg.CallerMethod() == "" {
		activeMsg.AInterface = msg.ServerRPCName() // 主调Interface
	}

	// 下游服务
	activeMsg.PApp = msg.CalleeApp()
	activeMsg.PServer = msg.CalleeServer()
	activeMsg.PService = msg.CalleeService()  // 被调Service
	activeMsg.PInterface = msg.CalleeMethod() // 被调Interface
	if msg.RemoteAddr() != nil {
		activeMsg.PIp = msg.RemoteAddr().String()
	}
	activeMsg.PContainer = msg.CalleeContainerName() // 被调容器名
	activeMsg.PConSetId = msg.CalleeSetName()

	activeMsg.Status, activeMsg.RetCode = covert(DefaultGetStatusAndRetCodeFunc(ctx, req, rsp, err))

	// 耗时ms
	activeMsg.Time = float64(time.Since(begin)) / float64(time.Millisecond)

	cfg := client.Config(msg.CalleeServiceName())
	if cfg != nil {
		activeMsg.PTarget = cfg.Target
	}
	// 增加自定义维度上报,业务可通过此维度上报框架定义之外的数据
	activeMsg.Etx = getExtensionDimension(msg)

	_ = pcgmonitor.ReportActive(activeMsg)

	return err
}

// getServerAddress 通过服务名找到其address
func getServerAddress(serviceName string) string {
	globalConfig := trpc.GlobalConfig()
	if globalConfig == nil {
		return ""
	}

	serviceConfigArray := globalConfig.Server.Service
	if serviceConfigArray == nil || len(serviceConfigArray) == 0 {
		return ""
	}

	for _, serviceConfig := range serviceConfigArray {
		if serviceConfig != nil && serviceConfig.Name == serviceName {
			return serviceConfig.Address
		}
	}

	return ""
}

// isCalleeComplete
func isCalleeComplete(msg codec.Msg) bool {
	return msg.CalleeApp() != "" && msg.CalleeServer() != "" && msg.CalleeMethod() != ""
}

// PassiveModuleCallServerFilter  被调模调上报拦截器:上游调用自身, 处理函数结束时上报
func PassiveModuleCallServerFilter(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) error {
	begin := time.Now()

	err := handler(ctx, req, rsp)

	msg := trpc.Message(ctx)

	passiveMsg := new(pcgmonitor.PassiveMsg)
	// 自身服务
	passiveMsg.PService = msg.CalleeService()
	passiveMsg.PInterface = msg.CalleeMethod()
	if msg.CalleeMethod() == "" {
		passiveMsg.PInterface = msg.ServerRPCName()
	}
	if msg.LocalAddr() != nil {
		passiveMsg.PIp = msg.LocalAddr().String()
	}

	// 上游服务
	passiveMsg.AApp = msg.CallerApp()
	passiveMsg.AServer = msg.CallerServer()
	if msg.CallerServer() == "" {
		passiveMsg.AServer = msg.CallerServiceName()
	}
	passiveMsg.AService = msg.CallerService()  // 主调Service
	passiveMsg.AInterface = msg.CallerMethod() // 主调Interface 目前的协议无法知道上游的接口名，只能知道上游的服务名
	if msg.CallerMethod() == "" {
		passiveMsg.AInterface = "unknown"
	}
	addr := getAddr(msg)
	passiveMsg.AIp = addr // 主调IP

	reportErr := fixServerTimeoutErr(ctx, err)
	passiveMsg.Status, passiveMsg.RetCode = covert(DefaultGetStatusAndRetCodeFunc(ctx, req, rsp, reportErr))

	// 耗时ms
	passiveMsg.Time = float64(time.Since(begin)) / float64(time.Millisecond)

	passiveMsg.AAddress = getServerAddress(msg.CalleeServiceName())

	// 增加自定义维度上报,业务可通过此维度上报框架定义之外的数据
	passiveMsg.Etx = getExtensionDimension(msg)

	_ = pcgmonitor.ReportPassive(passiveMsg)

	return err
}

func fixServerTimeoutErr(ctx context.Context, err error) error {
	if err == nil {
		err = ctx.Err()
	}
	switch err {
	case context.Canceled:
		return errs.NewFrameError(errs.RetClientCanceled, "client cancel")
	case context.DeadlineExceeded:
		return errs.ErrServerTimeout
	default:
		return err
	}
}

// getAddr 获取IP，不包括端口
func getAddr(msg codec.Msg) string {
	var addr string
	if msg.RemoteAddr() != nil {
		addr = msg.RemoteAddr().String()
		s := strings.Split(addr, ":")
		if len(s) > 0 {
			addr = s[0]
		}
	}
	return addr
}

// Status 指007系统使用的Status
type Status int

const (
	// StatusSuccess 成功
	StatusSuccess Status = 0
	// StatusException 异常
	StatusException Status = 1
	// StatusTimeout 超时
	StatusTimeout Status = 2
)

// covert covert Status to int64
func covert(status Status, retCode string) (int64, string) {
	return int64(status), retCode
}

// GetStatusAndRetCodeFunc 根据ctx req rsp得到新的用于007监控使用的status和code
type GetStatusAndRetCodeFunc func(ctx context.Context, req interface{}, rsp interface{}, err error) (Status, string)

/*
DefaultGetStatusAndRetCodeFunc 可自定义007监控使用的status和code的计算方式, 适合业务handler没有返回error但需要使用007主调被调监控的场景
示例:
func defaultCodeStatus(code int32) (m007.Status, string) {
	if code == 0 {
		return m007.StatusSuccess, strconv.Itoa(int(code))
	}
	return m007.StatusException, strconv.Itoa(int(code))
}

func init() {
	m007.DefaultGetStatusAndRetCodeFunc = func(ctx context.Context,
		req interface{}, rsp interface{}, err error) (m007.Status, string) {

		if err != nil {
			return m007.GetStatusAndRetCodeFromError(err)
		}
		switch v := rsp.(type) {
		case interface {
			GetRetcode() int32
		}:
			return defaultCodeStatus(v.GetRetcode())
		case interface {
			GetRetCode() int32
		}:
			return defaultCodeStatus(v.GetRetCode())
		case interface {
			GetCode() int32
		}:
			return defaultCodeStatus(v.GetCode())
		default:
			return defaultCodeStatus(0)
		}
	}
}
*/
var DefaultGetStatusAndRetCodeFunc GetStatusAndRetCodeFunc = func(ctx context.Context,
	req interface{}, rsp interface{}, err error) (Status, string) {
	return GetStatusAndRetCodeFromError(err)
}

// GetStatusAndRetCodeFromError 默认从err中获取错误, 自定义的DefaultGetStatusAndRetCodeFunc也可以调用此函数
func GetStatusAndRetCodeFromError(err error) (Status, string) {
	retCode := "0"
	status := StatusSuccess

	if err != nil {
		e, ok := err.(*errs.Error)
		if ok && e != nil {
			status, retCode = getStatusAndRetFromTrpcError(e)
		} else {
			// 兼容 业务没有使用框架error,上报固定错误值
			status = StatusException // 异常
			retCode = "007_999"
		}
	}
	return status, retCode
}

func getStatusAndRetFromTrpcError(e *errs.Error) (Status, string) {
	status := StatusException
	if e.Type == errs.ErrorTypeFramework && (e.Code == errs.RetClientTimeout || e.Code == errs.RetServerTimeout) {
		status = StatusTimeout // 超时
	}
	if e.Code == errs.RetOK {
		status = StatusSuccess
	}

	var retCode string
	if e.Desc != "" {
		retCode = fmt.Sprintf("%s_%d", e.Desc, e.Code)
	} else {
		retCode = strconv.Itoa(int(e.Code))
	}

	return status, retCode
}
