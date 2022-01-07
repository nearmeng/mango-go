# 参与天机阁贡献

## 开发

你可以通过克隆如下仓库查看或者开发天机阁源码：

```bash
git clone git@git.code.oa.com:tpstelemetry/tps-sdk-go.git
```

你可以执行`make precommit`对源码进行编译，代码规范，单元测试。**注：这是提交代码前的必要操作!**

## Merge Requests

### 如何提交 MR

欢迎任何人通过工蜂MergeRequest（MR）贡献代码到天机阁.

为了创建新的MR，请在工蜂系统Fork此项目，并添加你Fork的项目为作为本地Git仓库的一个新远端仓库。

```bash
git remote add <你的工蜂用户名> git@git.code.oa.com:<你的工蜂用户名>/tps-sdk-go
```

检出一个新的分支，进行代码编写贡献，执行`make precommit`进行代码提交前的本地代码检查，通过后Push
你的分支到你Fork的远程仓库。

```sh
$ git checkout -b <分支名称>
# edit files
# update changelog
$ make precommit
$ git add -p
$ git commit
$ git push <你的工蜂用户名> <分支名称>
```

在天机阁工蜂主页提交Merge Requests.
