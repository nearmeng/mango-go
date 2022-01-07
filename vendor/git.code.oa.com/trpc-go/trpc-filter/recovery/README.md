[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-77a39cfc200a42f1ae86dd822e83700b/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-77a39cfc200a42f1ae86dd822e83700b/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject)

# recovery 
请求入口panic捕获插件, 一般放在filter第一个

## 使用说明:

 - 增加import

````
import (
   _ "git.code.oa.com/trpc-go/trpc-filter/recovery"
)
````

 - TRPC框架配置文件,server开启recovery拦截器,自动recover捕获panic

````
server:
 ...
 filter:
  - recovery 
  ...
````

