# goauto
一个基于命令行调用git的测试环境自动化实现

## 安装方法

```bash
git clone https://github.com/lroyia/goauto #clone code
cd goauto #cd dir
go build # build
```

## 使用方法

> 事前准备：
> 1、安装git
> 2、执行`git config --global credential.helper store`使得git不用重复输入密码
> 3、使用git将所有需要配置自动化的仓库拉取下来
> 4、对每个仓库都进行一次`git pull`保证以后不需要输入帐号密码
> 5、安装好所有构建所需要的应用，并配置好相应的环境变量

1、在构建出来的可执行程序存放的同一目录下下创建conf.ini配置文件，并进行配置

2、运行可执行程序

## 配置说明

示例：
```ini
[conf]
# 轮询进行构建的休眠时间
sleep=10

[dir]
# key为项目自定义标识，用于分辨下面对应的各个配置，value为项目的管理目录
boot=D:/test/springboot-example

[branch]
# 这里配置的是项目需要监听并构建的是哪一个分支
boot=develop

[script]
# 构建脚本，多个用“|”分开，并默认以最后一条为执行命令
# 在下面示例中标识的是用maven打包项目并跳过测试，完成后直接使用java执行jar包
boot=mvn package -Dmaven.test.skip=true | java -jar target/springboot-example-0.0.1-SNAPSHOT.jar
```