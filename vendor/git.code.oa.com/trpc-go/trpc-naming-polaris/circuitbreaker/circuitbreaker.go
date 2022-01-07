package circuitbreaker

import (
	"errors"
	"fmt"
	"time"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/naming/circuitbreaker"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
)

// Config 熔断器配置
type Config struct {
	// ReportTimeout 如果设置了，则下游超时，并且少于设置的值，则忽略错误不上报
	ReportTimeout *time.Duration
}

const (
	errRetCode   = 10000
	DeltaTimeout = time.Millisecond
)

// Setup 注册
func Setup(sdkCtx api.SDKContext, cfg *Config, setDefault bool) error {
	cb := &CircuitBreaker{
		consumer: api.NewConsumerAPIByContext(sdkCtx),
		cfg:      cfg,
	}
	circuitbreaker.Register("polaris", cb)
	if setDefault {
		circuitbreaker.SetDefaultCircuitBreaker(cb)
	}
	return nil
}

// CircuitBreaker 熔断器结构
type CircuitBreaker struct {
	consumer api.ConsumerAPI
	cfg      *Config
}

// Available 判断节点是否可用
func (cb *CircuitBreaker) Available(node *registry.Node) bool {
	inst, ok := node.Metadata["instance"].(model.Instance)
	if !ok {
		return false
	}
	if inst.GetCircuitBreakerStatus() == nil {
		return true
	}
	return inst.GetCircuitBreakerStatus().IsAvailable()
}

// Report 上报请求状态
func (cb *CircuitBreaker) Report(node *registry.Node, cost time.Duration, err error) error {
	return Report(cb.consumer, node, cb.cfg.ReportTimeout, cost, err)
}

// Report 上报请求状态
func Report(
	consumer api.ConsumerAPI,
	node *registry.Node,
	reportTimeout *time.Duration,
	cost time.Duration,
	err error,
) error {
	delta := DeltaTimeout
	if reportTimeout != nil {
		delta = *reportTimeout
	}
	retStatus := model.RetSuccess
	var retCode int32
	if !canIgnoreError(err, delta, cost) {
		retStatus = model.RetFail
		retCode = errRetCode
	}
	inst, ok := node.Metadata["instance"].(model.Instance)
	if !ok {
		return errors.New("report err: invalid instance")
	}
	if err := consumer.UpdateServiceCallResult(&api.ServiceCallResult{
		ServiceCallResult: model.ServiceCallResult{
			CalledInstance: inst,
			RetStatus:      retStatus,
			Delay:          &cost,
			RetCode:        &retCode,
		},
	}); err != nil {
		return fmt.Errorf("report err: %v", err)
	}
	return nil
}

func canIgnoreError(e error, reportTimeout, cost time.Duration) bool {
	if e == nil {
		return true
	}
	// 如果errorCode==101 && cost < reportTimeout && errorType==framework
	// 这种情况，熔断器应该认为是正常。
	err, ok := e.(*errs.Error)
	if ok &&
		err.Code == errs.RetClientTimeout &&
		err.Type == errs.ErrorTypeFramework &&
		cost < reportTimeout {
		return true
	}
	return false
}
