package rainbow

import (
	"fmt"

	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/confapi"
)

var (
	globalSDKBuilder SDKBuilder
)

func init() {
	globalSDKBuilder = NewSDKBuilder()
}

// RegisterSDKBuilder 设置全局的sdkbuilder
func RegisterSDKBuilder(builder SDKBuilder) {
	globalSDKBuilder = builder
}

// SDKBuilder 构造sdk
type SDKBuilder interface {
	BuildSDK(opts []types.AssignInitOption) (SDK, error)
}

type sdkBuilder struct {
}

// NewSDKBuilder 初始化SDKBuilder
func NewSDKBuilder() SDKBuilder {
	return &sdkBuilder{}
}

func (b *sdkBuilder) BuildSDK(opts []types.AssignInitOption) (SDK, error) {
	rainbow, err := confapi.NewAgain(opts...)
	if err != nil {
		return nil, fmt.Errorf("trpc-config-rainbow: confapi init failed %s", err.Error())
	}
	return rainbow, nil
}
