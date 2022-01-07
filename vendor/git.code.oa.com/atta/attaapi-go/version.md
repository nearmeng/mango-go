### CHANGELOG

#### [v1.0.0](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.0) - 2021-01-12
- 【修改】
    + 按照代码规范修改package的名字。

#### [v1.0.2](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.2) - 2021-01-12
- 【修改】
    + 修复token为空时，组包异常的问题。
    + TCP、Unix上报单条限制提升到510k。

#### [v1.0.3](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.3) - 2021-01-14
- 【修改】
  + 超长的判定修改为大于判定为超长，与其它语言保持一致。

#### [v1.0.4](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.4)
- 【修改】
  + 修复定时器未释放的bug。

#### [v1.0.5](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.5)
- 【修改】
  + 增加连接失败时重试，重试5次，每次间隔10ms。

#### [v1.0.6](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.6)
- 【修改】
  + tcp协议增加读写超时。

#### [v1.0.7](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.7)
- 【修改】
  + 细化错误处理, 增加自定义超时设置。

#### [v1.0.8](https://git.code.oa.com/atta/attaapi-go/-/tags/v1.0.8) - 2021-05-10
- 【修改】
  + TCP上报上限修改为500k。
  + 超长判定逻辑与attaapi_cplus保持一致。