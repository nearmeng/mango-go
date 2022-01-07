## 007监控上报SDK

### 介绍

Go版本，详细文档请参考[cpp版本](https://git.code.oa.com/pcgmonitor/pcgmonitor_report_cpp), Go版本功能、接口、实现基本一致

### 使用

简要说明如下

#### 框架上报

- trpc-go 监控上报请直接参考[trpc-metrics-m007](https://git.code.oa.com/trpc-go/trpc-metrics-m007) 文档

#### 非框架上报

- 请参考example/comm

初始化逻辑
```go
	_ = pcgmonitor.CommSetUp(&pcgmonitor.CommSvrSetupInfo{
		CommSvrInfo: pcgmonitor.CommSvrInfo{
			CommName:  "junkCommTest",
			IP:        "1.2.3.4",
			Container: "container",
		},
	})
```
CommSvrInfo数据与CPP版本一致，PolarisInfo是Go版本需要，SDK内部拉配置依赖名字服务北极星的相关配置，优先使用域名方式。解析有问题可替换注释掉IP配置(IP方式不好维护)

#### 注意
- 非框架上报使用之前需要到 007官网-配置管理-接入申请 ，新增接入处填写监控项名、负责人、关联业务
- 非框架上报CommName字段值不要带`comm_`前缀
- 数据上报依赖atta，使用前请确保atta agent已安装,进程有启动，详见[如何安装AttaAgent](http://km.oa.com/articles/show/447456?kmref=search&from_page=1&no=3)
- 框架自定义上报和非框架上报 dimensions和values数量自定义数量，但是每次上报的自定义数量需要保证相同（如果维度个数不同会导致错位，而指标个数不同会丢掉所有上报数据）
- 数据验证，首页--->监控搜索(关键字:监控项名字)---->获得监控项链接
- 不断迭代中，有问题请先更新到最新版本
    ``` shell script
    go get git.code.oa.com/pcgmonitor/trpc_report_api_go
    ```

#### 示例
* 请看examples目录，异步上报，要保证进程常驻，否则可能丢数据