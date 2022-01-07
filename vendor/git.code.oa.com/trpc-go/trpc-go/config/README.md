# tRPC-Go 配置库

trpc-go/config是一个组件化的配置库，提供了一种简单方式读取多种内容源、多种文件类型的配置。

## 功能

- 插件化：根据配置需要可从多个内容源（本地文件、TConf等）加载配置。

- 热加载：变更时自动载入新配置

- 默认设置：配置由于某些原因丢失时，可以回退使用默认值。

## 相关结构

- Config： 配置的通用接口，提供了相关的配置读取操作。

- ConfigLoader： 配置加载器，通过实现ConfigLoader相关接口以支持加载策略。

- Codec： 配置编解码接口，通过实现Codec相关接口以支持多种类型配置。

- DataProvider: 内容源接口，通过实现DataProvider相关接口以支持多种内容源。目前支持`file`，`tconf`，`rainbow`等配置内容源。

## 如何使用

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/config"
    _ "git.code.oa.com/trpc-go/trpc-config-tconf" // 使用TConf配置时引入注册
    _ "git.code.oa.com/trpc-go/trpc-config-rainbow" // 使用rainbow配置时引入注册
)


// 加载本地配置文件: config.WithProvider("file")
config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"), config.WithProvider("file"))

// 默认的DataProvider是使用本地文件
config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))

// 加载TConf配置文件: config.WithProvider("tconf")
c, _ := config.Load("test.yaml", config.WithCodec("yaml"), config.WithProvider("tconf"))

// 读取bool类型配置
c.GetBool("server.debug", false)

// 读取String类型配置
c.GetString("server.app", "default")

```

### 并发安全的监听远程配置变化

```go
import (
	"sync/atomic"
    ...
)

type yamlFile struct {
    Server struct {
        App string
    }
}

// 参考：https://golang.org/pkg/sync/atomic/#Value
var cfg atomic.Value // 并发安全的Value

// 使用trpc-go/config中Watch接口监听tconf远程配置变化
c, _ := config.Get("tconf").Watch(context.TODO(), "test.yaml")

go func() {
    for r := range c {
        yf := &yamlFile{}
        fmt.Printf("event: %d, value: %s", r.Event(), r.Value())

        if err := yaml.Unmarshal([]byte(r.Value()), yf); err == nil {
            cfg.Store(yf)
        }

    }
}()

// 当配置初始化完成后，可以通过 atomic.Value 的 Load 方法获得最新的配置对象
cfg.Load().(*yamlFile)

```

### 如何mock Watch
业务代码在单元测试时，需要对相关函数进行打桩

其他的方法需要mock，也是使用相同的方法，先打桩替换你需要的实现
```go

import (
	"context"
	"testing"

	"git.code.oa.com/trpc-go/trpc-go/config"
	mock "git.code.oa.com/trpc-go/trpc-go/config/mockconfig"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {
	ctrl := gomock.NewController(t)

    // 模拟七彩石及对应的Key
	mockKey := "test.json"
	mockProvider := "rainbow"

    // Watch 得到的所有数据都是通过写入mockChan去模拟
	mockChan := make(chan config.Response, 1)
	mockResp := &mockResponse{val: "mock-value"}
	mockChan <- mockResp

    // 模拟Watch的流程
	kv := mock.NewMockKVConfig(ctrl)
	kv.EXPECT().Name().Return(mockProvider).AnyTimes()
	m := kv.EXPECT().Watch(gomock.Any(), mockKey, gomock.Any()).AnyTimes()
	m.DoAndReturn(func(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) {
		return mockChan, nil
	})
    // 注册KVConfig
	config.Register(kv)

	// action
	got, err := config.Get(mockProvider).Watch(context.TODO(), mockKey)
	assert.Nil(t, err)
	assert.NotNil(t, got)

	resp := <-got
	assert.Equal(t, mockResp, resp)
}

type mockResponse struct {
    val string
}

func (r *mockResponse) Value() string {
    return r.val
}

func (r *mockResponse) MetaData() map[string]string {
    return nil
}

func (r *mockResponse) Event() config.EventType {
    return config.EventTypeNull
}


```

