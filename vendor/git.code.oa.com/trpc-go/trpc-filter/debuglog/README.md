# debuglog 
[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-9219a8e533ac4248bcf98670069ec250/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com:/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-9219a8e533ac4248bcf98670069ec250/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject)[![Coverage](https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-9219a8e533ac4248bcf98670069ec250)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-9219a8e533ac4248bcf98670069ec250)[![GoDoc](https://img.shields.io/badge/API%20Docs-GoDoc-green)](http://godoc.oa.com/git.code.oa.com/trpc-go/trpc-filter/debuglog)
所有接口请求自动打印debug日志

## 使用说明:

- 增加import
```go
import (
   _ "git.code.oa.com/trpc-go/trpc-filter/debuglog"
)
```
- TRPC框架配置文件。其中debuglog可以替换
  - debuglog：默认打印
  - simpledebuglog：不打印包体
  - pjsondebuglog：格式化json打印包体
  - jsondebuglog：压缩型json打印包体
```yaml
server:
 ...
 filter:
  ...
  - debuglog

client:
 ...
 filter:
  ...
  - debuglog
```


## 详细配置，可选

```yaml
plugins:
  tracing:
    debuglog:
      log_type: simple # 默认日志打印方式
      server_log_type: prettyjson # server日志打印方式，会覆盖log_type的设定
      client_log_type: json # client日志打印方式，会覆盖log_type的设定
      exclude: # 排除以下指定的接口
        - method: /trpc.app.server.service/method   # 按接口名来排除
        - retcode: 51 # 按错误码来排除
```
- 注意，使用插件配置时，filter必须为`debuglog`，否则插件的配置不会生效
- log_type/server_log_type/client_log_type的可填项为：
  - default：对应debuglog，默认打印
  - simple：对应simpledebuglog，不打印包体
  - prettyjson：对应pjsondebuglog，格式化json打印包体
  - json：对应jsondebuglog，压缩型json打印包体

## 自定义打印方法，可选

- 有时候业务需要自定义请求回包的打印方法，可以通过自己注册自定义的打印方法来实现
- 注意，使用自定义打印方式时，filter必须为`debuglog`，并会覆盖插件配置中的配置
```go
import (
	"context"
	"fmt"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-filter/debuglog"
)

func main() {
	// 自定义Server打印函数
	debugServerLogFunc := func(ctx context.Context, req, rsp interface{}) string {
		return fmt.Sprintf(", req:%+v, rsp:%+v, this is server log test", req, rsp)
	}
	// 自定义Client打印函数
	debugClientLogFunc := func(ctx context.Context, req, rsp interface{}) string {
		return fmt.Sprintf(", req:%+v, rsp:%+v, this is client log test", req, rsp)
	}
	// 注册filter
	filter.Register("debuglog",
		debuglog.ServerFilter(debuglog.WithLogFunc(debugServerLogFunc)),
		debuglog.ClientFilter(debuglog.WithLogFunc(debugClientLogFunc)))

	s := trpc.NewServer()

	pb.RegisterHttp_helloworldService(s, &Http_helloworldServerImpl{})
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}

}
```
