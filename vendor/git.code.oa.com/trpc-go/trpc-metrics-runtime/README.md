# tRPC-Go runtime监控
每分钟定时上报runtime关键监控信息


## 如何使用
业务服务import即可：

```golang
import _ "git.code.oa.com/trpc-go/trpc-metrics-runtime"
```

# 说明
go的runtime可认为是go运行所需要的基础设施, 主要有内存分配/垃圾回收, 协程调度, 操作系统及CPU差异性封装, pprof支持, 内置类型和反射的实现等. 

go runtime package提供了一些函数来inspect runtime本身的状况(runtime.ReadMemStats, runtime.GOMAXPROCS). 这个库会把这些数据做定时上报.

这里简单介绍这些监控的含义.
详细了解golang runtime, 可以参考这个篇文章 http://km.oa.com/group/19253/articles/show/403713

注意这些并不能完全解决runtime相关的问题. 还可以使用gctrace或者pprof来排查.
## 指标说明

|         指标          |                                  含义                                   |                                                        异常判断                                                         |
| --------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| GoroutineNum          | 协程数                                                                  | 一般和请求量还有服务耗时有关, 几千qps的服务有1000,2000的协程很正常. 协程数增加后不会再减少. 只要不是特别多(比如>1万), 就不用太关注      |
| ThreadNum             | go runtime中的m数量(大致可认为是线程数, 不包括c代码里启动的线程)               | 一般和机器核数有一些关系, 正常来说10-50都比较正常.如果使用了阻塞的cgo, 可能会比较多. 结合负载来看, 不超过100                         |
| GOMAXPROCSNum         | go这一层代码的最大并行度(非并发度)                                          | 一般等于go识别到的机器核数, 在容器中需要使用uber的automaxprocs库来根据配额正确设置, 否则会出现很多异常                              |
| CPUCoreNum            | go认为的机器核数(其实是给进程设置的cpu亲和性,sched_getaffinity)              | 无                                                                                                                     |
| PauseNsLt100usTimes   | 近256次gc中停顿<100us的次数                                              | 这个并不能判断gc是否频繁. 只是近256次gc的数据. 一般<500us是最多的. 偶尔有超过1ms也正常. 如果出现比较多的>10ms, 就要关注了             |
| PauseNs100_500usTimes | 近256次gc中停顿100-500us的次数                                           | 同上                                                                                                                    |
| PauseNs500us_1msTimes | 近256次gc中停顿500us-1ms的次数                                           | 同上                                                                                                                    |
| PauseNs1_10msTimes    | 近256次gc中停顿1ms-10ms的次数                                            | 同上                                                                                                                    |
| PauseNs10_50msTimes   | 近256次gc中停顿10ms-50ms的次数                                           | 同上                                                                                                                    |
| PauseNs50_100msTimes  | 近256次gc中停顿50ms-100ms的次数                                          | 同上                                                                                                                    |
| PauseNs100_500msTimes | 近256次gc中停顿100ms-500ms的次数                                         | 同上                                                                                                                    |
| PauseNs500ms_1sTimes  | 近256次gc中停顿500ms-1s的次数                                            | 同上                                                                                                                    |
| PauseNsBt1sTimes      | 近256次gc中停顿>1s的次数                                                 | 同上                                                                                                                    |
| AllocMem_MB           | 当前gc周期到现在分配的gc heap中的对象的字节数                                | 看分配的进度,一般意义不大                                                                                                  |
| SysMem_MB             | go运行时认为go从系统中申请的内存, 包含go的heap, 栈,维护一些运行时结构等的内存    | 一般可认为go进程占用的虚拟内存量                                                                                            |
| NextGCMem_MB          | go本次gc的目标heap值.                                                    | 会在AllocMem_MB大致小于NextGCMem_MB时, 开始本次gc. 大致在等于NextGCMem_MB结束gc. 一般可认为是gc时占的内存                        |
| PauseNs_us            | 历史gc停顿总时间                                                         |                                                                                                                         |
| GCCPUFraction_ppb     | gc总消耗占从进程启动到现在所有cpu时间的比例                                  | 没有太大意义. 可能值很小, 因为被平均了. 正常服务都是0%,1%, 如果有5%这样子, 那就要考虑是否有问题                                      |
| MaxFdNum              | 给进程设置的允许最大fd数量                                                 |                                                                                                                        |
| CurrentFdNum          | 当前进程打开的fd数量                                                      |                                                                                                                        |
| PidNum                | 当前机器/容器中的进程数                                                    |                                                                                                                        |
| TcpNum                | 机器已使用(已分配+待关闭等)的所有协议套接字描述符总量                           | 一般在容器中会比较大(比如几万). 对于大多数业务来说意义不大. 在需要巨大量(比如几十万?)连接的服务上可供参考                              |