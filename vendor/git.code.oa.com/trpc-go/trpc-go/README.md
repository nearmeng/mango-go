# tRPC-Go framework

[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-20167ab337e04866b254949853c75b60/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://api.devops.oa.com/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-20167ab337e04866b254949853c75b60/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject) [![Coverage](https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-90b7156b71494b2dada67caf57290480)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-90b7156b71494b2dada67caf57290480) [![Benchmark](https://qualitydata.woa.com/GetEabData?frameName=tRPC-Go&pressTestInstance=echo_tag&frameVersion=v0.8.0)](http://show.wsd.com/show3.htm?viewId=t_md_platform_server_eab_pressuretest_data) [![Go Reference](https://pkg.woa.com/badge/git.code.oa.com/trpc-go/trpc-go.svg)](https://pkg.woa.com/git.code.oa.com/trpc-go/trpc-go) [![iwiki](https://img.shields.io/badge/Wiki-iwiki-green)](https://iwiki.oa.tencent.com/display/tRPC/tRPC-Go)


tRPC-Go框架是公司统一微服务框架的golang版本，主要是以高性能，可插拔，易测试为出发点而设计的rpc框架。

# 文档地址：[iwiki](https://iwiki.oa.tencent.com/display/tRPC/tRPC-Go)
# 需求管理：[tapd](http://tapd.oa.com/trpc_go/prong/stories/stories_list)


## TRY IT !!!

# 整体架构
![架构图](https://git.code.oa.com/trpc-go/trpc-go/uploads/76DF446E40304476B8E12903E78B5EC4/2FE60489777F72A5901D36F114CFF331.png)

- 一个server进程内支持启动多个service服务，监听多个地址。
- 所有部件全都可插拔，内置transport等基本功能默认实现，可替换，其他组件需由第三方业务自己实现并注册到框架中。
- 所有接口全都可mock，使用gomock&mockgen生成mock代码，方便测试。
- 支持任意的第三方业务协议，只需实现业务协议打解包接口即可。默认支持trpc和http协议，随时切换，无差别开发cgi与后台server。
- 提供生成代码模板的trpc命令行工具。

# 插件管理
- 框架插件化管理设计只提供标准接口及接口注册能力。
- 外部组件由第三方业务作为桥梁把系统组件按框架接口包装起来，并注册到框架中。
- 业务使用时，只需要import包装桥梁路径。
- 具体插件原理可参考[plugin](blob/master/plugin) 。

# 生成工具
- 安装

```
# 初次安装，请确保环境变量PATH已配置$GOBIN或者$GOPATH/bin
go get -u git.code.oa.com/trpc-go/trpc-go-cmdline/trpc

# 配置依赖工具，如protoc、protoc-gen-go、mockgen等等
trpc setup

# 后续更新、回退版本
trpc version                            # 检查版本
trpc upgrade -l                         # 检查版本更新
trpc upgrade [--version <version>]      # 更新到指定版本
```

- 使用
```bash
trpc help create
```
```bash
指定pb文件快速创建工程或rpcstub，

'trpc create' 有两种模式:
- 生成一个完整的服务工程
- 生成被调服务的rpcstub，需指定'-rpconly'选项.

Usage:
  trpc create [flags]

Flags:
      --alias                  enable alias mode of rpc name
      --assetdir string        path of project template
  -f, --force                  enable overwritten existed code forcibly
  -h, --help                   help for create
      --lang string            programming language, including go, java, python (default "go")
  -m, --mod string             go module, default: ${pb.package}
  -o, --output string          output directory
      --protocol string        protocol to use, trpc, http, etc (default "trpc")
      --protodir stringArray   include path of the target protofile (default [.])
  -p, --protofile string       protofile used as IDL of target service
      --rpconly                generate rpc stub only
      --swagger                enable swagger to gen swagger api document.
  -v, --verbose                show verbose logging info
```

# 服务协议
- trpc框架支持任意的第三方协议，同时默认支持了trpc和http协议
- 只需在配置文件里面指定protocol字段等于http即可启动一个cgi服务
- 使用同样的服务描述协议，完全一模一样的代码，可以随时切换trpc和http，达到真正意义上无差别开发cgi和后台服务的效果
- 请求数据使用http post方法携带，并解析到method里面的request结构体，通过http header content-type(application/json or application/pb)指定使用pb还是json 
- 第三方自定义业务协议可以参考[codec](blob/master/codec)
 
# 相关文档
- [框架设计文档](https://iwiki.oa.tencent.com/display/tRPC/tRPC-Go)
- [trpc工具详细说明](https://git.code.oa.com/trpc-go/trpc-go-cmdline)
- [helloworld开发指南](https://git.code.oa.com/trpc-go/trpc-go/tree/master/examples/helloworld)
- [第三方插件cl5实现demo](https://git.code.oa.com/trpc-go/trpc-selector-cl5)
- [第三方协议实现demo](https://git.code.oa.com/trpc-go/trpc-codec)

# 如何贡献
tRPC-Go项目组有专门的[tapd需求管理](http://tapd.oa.com/trpc_go/prong/stories/stories_list)，里面包括了各个具体功能点以及负责人和排期时间，
有兴趣的同学可以先看一下[贡献规范文档](https://iwiki.woa.com/pages/viewpage.action?pageId=655869831)，再看看tapd里面 <font color=#DC143C>需求状态为规划中</font> 的功能，自己认领任务，一起为tRPC-Go做贡献。
认领时将状态流转为： 需求已确认
开始投入将状态流转为： 开发中
开发完成将状态流转为： 已发布
开发中 和 已发布 之间时间不要超过两周。需求比较大的单可以拆分成多个子需求。
