# AttaAPI-GO

[阿塔系统](http://atta.pcg.com) GO版 SDK，用于上报标准AttaId。**`不能上报BossId、DCId、ASF及他们对应的AttaId。`**

流程图：

```
+----------------+             +----------------+             +----------------+
|                |             |                |             |                |
|      SDK       +------------>+   Atta Agent   +------------>+   Atta Server  |
|                |             |                |             |                |
+----------------+             +----------------+             +----------------+
```

## 老兵排雷
- AttaAPI，只用于上报标准AttaId。**`不能上报BossId、DCID、ASF及他们对应的AttaId`**，会导致入库数据异常，不能正常使用。
- BossId、DCID、ASF需要使用原来的BossAPI、DCAPI、LibFZLog来上报数据。
- AttaAPI在调用发送数据接口前，必须 **`先调用初始化接口，并判断初始化返回值，发现初始化异常后还继续发送，会导致数据丢失`**，这时请发送MM告警或其他告警，联系业务运维安装AttaAgent或重启AttaAgent。
- AttaAPI依赖本机安装并启动了AttaAgent，否则上报都会失败。
 - 物理机，要求本机安装并启动AttaAgent；
 - 独立IP容器，如stke环境，要求本容器安装并启动AttaAgent；
 - 共享IP容器，如sumeru环境，要求宿主机安装并启动AttaAgent，并且上报方式只能是UDP和TCP方式；
 - **`建议使用 udp 上报，已经满足绝大多数场景了；如果对数据质量要求比较苛刻，可以使用 tcp，但要清楚极端情况下每次调用会卡住 10ms `**。
- **`一个进程建议只用一个atta_api对象上报数据，只需要初始化一次！只需要初始化一次！只需要初始化一次！`**，初始化接口会和AttaAgent建立网络连接，频繁调用会引发性能问题。
- **`需要对发送数据接口的返回值进行判断`**，当发生AttaAgent处理不过来、发送数据超长或不符合要求等异常情况时，发送数据接口会返回异常错误码，业务代码需要根据错误码进行异常处理，否则会导致数据丢失。
- 如果发送数据的字段中有回车符、换行符、竖线分隔符、反斜杠(转义符)，需要先进行转义后再拼接(AttaAPI有转义接口)；
 - **`如果不转义，那么入库后数据的字段顺序会发生错乱，换行符会导致一条数据变两条，竖线分隔符会导致字段错位；`**
 - **`转义操作需要消耗系统资源，降低发送性能，请只对有需要的字段转义，减少性能消耗。`**
- 共享IP容器是共用宿主机的AttaAgent，AttaAgent是有[性能指标](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=126767452)的，指标是基于宿主机上所有共享IP容器的总量，不是单个共享IP容器的数据量，总量超过性能指标的安全阈值，会导致数据丢失。
- 发送接口有三个：发送binary、发送string、发送字段列表，请根据业务特性选择最合适的接口。

## 快速开始
## 引入依赖

- mod方式
import (
	"git.code.oa.com/atta/attaapi-go"
)
- 文件方式
将git库里的[attaapi-go包](https://git.code.oa.com/atta/attaapi-go/blob/master/attaapi_go.go)放入$GOPATH可访问的目录中；

## 实例代码

[更多实例](https://git.code.oa.com/atta/attaapi-go/blob/master/demo/toatta_go.go)

```go
import (
	"git.code.oa.com/atta/attaapi-go"
)

var apiObj attaapi.AttaApi
...
func init(){
	//初始化很消耗资源，切忌频繁调用！建议单进程复用一个 atta api 对象
	initResult := apiObj.InitUDP()
	if initResult != attaapi.AttaReportCodeSuccess {
		//TODO 发送告警，或上报监控，或重启初始化
	}	
	...
}

func work(){
	var testAttaId = "x8b00097101"
	var testToken = "6735996849"
	fmt.Println("begin to send log,sAttaid:%s", testAttaId)
	//字段数组上报例子，字段顺序需要和http://atta.pcg.com中配置的AttaId字段顺序一致
	var fieldValues = make([]string, 4)
	fieldValues[0] = "fieldValue1"
	fieldValues[1] = "fieldValue2"
	fieldValues[2] = apiObj.EscapeString("specialFieldValue3 :\t\n\r\\|")//只对含有特殊字符的字段做转义
	fieldValues[3] = "fieldValue4"
	// 不自动转义
	var auto_escape bool = false
	result := apiObj.SendFields(testAttaId, testToken, fieldValues, auto_escape)
	if result != attaapi.AttaReportCodeSuccess {
	 	//TODO 重新发送，或容灾落地，或发送告警
	}	
	...
}

func stop(){
	apiObj.Release()
	...
}
```


## 上报验证
登录[Atta系统](http://atta.pcg.com)查看上报AttaID的实时数据，验证数据是否正确上报。

## 错误码
```go
const AttaReportCodeSuccess int = 0          // 成功
const AttaReportCodeInvalidMsg int = -1     //消息无效，可能原因有消息空指针、消息长度为0、消息字段列表为空、批量发送消息列表为空
const AttaReportCodeOverLimit int = -2      // 消息超长
const AttaReportCodeNetFailed int = -3        //发送失败，可能原因有UDP系统缓存满了、发送过程中socket连接被中断、UDP发送消息长度+Atta消息头长度超过UDP协议约束64K-20
const AttaReportCodeNotInit = -4            //未初始化，没有调用init初始化函数
const AttaReportCodeNoUsablePort = -5 //无可用端口
const AttaReportCodeCreateSocketFailed = -6  //创建Socket失败
const AttaReportCodeUnknownSocketType = -7 //连接Socket失败
const AttaReportCodeUnknownSocketType = -8  //socket类型未知
```



## API

### func (p *AttaApi) InitUDP() int
UDP初始化接口，会和AttaAgent建立网络连接，频繁调用会引发性能问题。

### func (p *AttaApi) InitTCP() int
TCP初始化接口，会和AttaAgent建立网络连接，频繁调用会引发性能问题。

### func (p *AttaApi) InitUnix() int
Unix套接字初始化接口，会和AttaAgent建立网络连接，频繁调用会引发性能问题。

### func (p *AttaApi) release() 
释放连接接口，必须和初始化接口成对出现；如果没有调用释放连接接口，会导致机器上出现大量TIME_WAIT连接，严重时会影响所有依赖网络通讯的业务。

### func (p *AttaApi) SendFields(sAttaid string, sToken string, _fields []string, autoEscape bool) int
将多个字段转义后的值，用竖线|拼接后发送AttaAgent
1. sAttaId，在[阿塔系统](http://atta.pcg.com)中申请；
2. sToken，和attaId配套，在阿塔系统的AttaId查看页面的上报指引中获取；
3. _fields，字段值列表
4. autoEscape，是否转义；选择自动转义时会有一定的性能消耗。
```bash
转义规则如下：
0x00-->\0（反斜杠+字符0）
回车符0x0D-->\r（反斜杠+字符r）
换行符0x0A-->\n（反斜杠+字符n）
反斜杠\-->\\（两个反斜杠）
分隔符竖线|-->\|（反斜杠+字符|）
```

### func (p *AttaApi) SendBinary(sAttaid string, sToken string, bData []byte) int
发送二进制数据到AttaAgent

### func (p *AttaApi) SendString(sAttaid string, sToken string, sData string) int
发送字符串数据到AttaAgent，字段用竖线|分隔

### func (p *AttaApi) BatchSendFields(sAttaid string, sToken string, _fields [][]string, autoEscape bool) int 
```bash
批量发送string slice
参数:sAttaid Attaid, sToken Token，详见attaid 信息；autoEscape（true 自动转义，false 不自动转义）。
返回:0 成功, 其它 失败
注意:选择自动转义时会有一定的性能消耗。
```

### func (p *AttaApi) BatchSendString(sAttaid string, sToken string, sData []string) int 
```bash
批量发送string
参数:sAttaid Attaid, sToken Token，详见attaid 信息
返回:0 成功, 其它 失败
```

### func (p *AttaApi) BatchSendBinary(sAttaid string, sToken string, bData [][]byte) int
```bash
批量发送byte slice
参数:sAttaid Attaid, sToken Token，详见attaid 信息
返回:0 成功, 其它 失败
```

### func (p *AttaApi) EscapeString(fieldValue string) string
字段值转义

### func (p *AttaApi) UnescapeString(fieldValue string) string
字段值反转义

### func (p *AttaApi) EscapeFields(fields []string) string
字段数组转义

### func (p *AttaApi) UnescapeFields(fieldValues string) []string
字段数组反转义

## 关于Atta
### Atta管理平台
http://atta.pcg.com

### AttaAPI统一入口
https://git.code.oa.com/atta/atta_api

### 协议文档
如果AttaAPI没有您需要的语言，可以考虑协同共建，协议文档如下：
http://km.oa.com/group/2306/articles/show/389025?kmref=author_post

### AttaAgent安装
PCG机器，AttaAgent通常会预装在每台机器；
如果没有安装或者不确定，可以参考文档[AttaAgent安装](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=126767410)，保障AttaAgent安装和启动。

### AttaAgent性能约束
参见文档[AttaAgent性能](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=126767452)。

### AttaAgent使用指南
参见文档[AttaAgent使用指南](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=118670748)。

### Atta系统流量切换方案
https://docs.qq.com/slide/DTFpsdU5MbGhZd0NQ

## FAQ
### AttaAgent端口是哪个？
```bash
AttaAgent端口为了支持独立IP容器中可以分别启动多个AttaAgent，不是固定一个端口，而是一个端口列表：
6588,16588,26588,36588,46588,56588,9112,19112,29112,39112,49112,59112,10015,20015,30015,40015,50015,60015
AttaAgent启动时会依次尝试监听，如果该端口没被占用，并经过签名接口验证后就绑定监听，并且不再往下查找。
如果端口全部被用满了，AttaAgent就启动不了了。
```

### AttaAPI是否线程安全？
AttaAPI是线程安全的，可以封装成单例统一调用，但一个AttaAPI只建立一个与AttaAgent之间的网络连接，高上报量时有性能风险，会导致数据丢失。

### AttaAgent Unix套接字在哪？
Unix套接字文件在/data/pcgatta/agent/atta_agent.unix，但不建议使用，在容器环境支撑困难。

### Atta上报约束是多少？
- UDP方式上报的大小需要小于63KB。
- v1.0.7及以下版本的api，TCP方式上报的大小需要小于127KB。
- v1.0.8及以上版本的api，TCP方式上报的大小可以支持到500KB，此外，atta_agent需要使用1.9.231及以上的版本。
- 单次批量上报的数据条数不可超过100条。

### 蓝盾等开发网的机器为什么上传Atta数据失败？
AttaServer目前都部署在内网的TAF或者TRPC环境，和DEVNET开发网环境网络不通，虽然可以配置代理，TAF或者TRPC环境的容器IP会因扩缩容和迁移而经常变动。
暂时不支持开发网里上报数据，如需上报请参考前台上报的HTTP外网上报方式。

### 在Atta管理平台的实时预览上查询不到刚上报的数据
- 首先用[上报测试程序](https://git.code.oa.com/atta/attaapi_go/blob/master/demo/toatta_go.go)验证一下；
- 如果测试工具上报成功，那么需要检查调用AttaApi的代码有什么问题；
- 如果测试工具也上报不成功，那么需要企业微信联系Venus_Helper确定上报机器是否已经安装AttaAgent，联系业务运维确定上报机器和MIG_Sumeru之间网络是否通畅；
- 如果是一个大数据量AttaId，实时预览不能保证刚好搜索到您的数据，因为实时预览的数据是抽样的，主要用于新AttaId的开发验证。

