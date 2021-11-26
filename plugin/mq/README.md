# 消息队列框架
> 指在将不同的消息队列接入到统一的消息队列框架中，使用统一API
## 设计文档
* [iwiki](https://iwiki.woa.com/pages/viewpage.action?pageId=718228539)

## example
* 主要提供两套API，具体见examples文件夹下的例子

## TODO
[✓] Reactor模式自定义context，支持ack  
[✓] Writer,Reader端添加interceptor（前置preWrite后置afterWrite处理接口）支持  
[✓] Reader添加读指定分区的接口
[✓] 消息结构进一步抽象
[✓] 切换Kafka实现
[✗] namespace多租户支持  
[✗] 过滤消费
## 优化
[✓] 拦截器优化设计  
[x] 配置收拢 50%