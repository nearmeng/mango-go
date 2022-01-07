# 熔断器插件

```yaml
selector:                                          #针对trpc框架服务发现的配置
  polaris:                                         #北极星服务发现的配置
    circuitbreaker:
      checkPeriod: 30s                             #实例定时熔断检测周期, 默认值:30s
      requestCountAfterHalfOpen: 10                #熔断器半开后最大允许的请求数, 默认值:10
      sleepWindow: 30s                             #熔断器打开后，多久后转换为半开状态，默认值:30s
      successCountAfterHalfOpen: 8                 #熔断器半开到关闭所必须的最少成功请求数，默认值:8
      chain:                                       #熔断策略，默认值：[errorCount, errorRate]
        - errorCount                               #基于周期错误率的熔断
        - errorRate                                #基于周期连续错误数熔断
      errorCount:
        continuousErrorThreshold: 10               #触发连续错误熔断的阈值，默认值:10
        metricNumBuckets: 10                       #连续错误数的最小统计单元数量，默认值:10
        metricStatTimeWindow: 1m0s                 #连续失败的统计周期，默认值:1m
      errorRate:
        errorRateThreshold: 0.5                    #触发错误率熔断的阈值，默认值:0.5
        metricNumBuckets: 5                        #错误率熔断的最小统计单元数量，默认值:5
        metricStatTimeWindow: 1m0s                 #错误率熔断的统计周期，默认值:1m
        requestVolumeThreshold: 10                 #触发错误率熔断的最低请求阈值，默认值:10
```
