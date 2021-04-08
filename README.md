# 镜像迁移工具：image-transfer

`image-transfer` 是一个docker镜像迁移工具，用于对不同镜像仓库的镜像进行批量迁移。
## 特性
- 支持多对多镜像仓库迁移
- 支持腾讯云CCR一键全量迁移模式：腾讯云TCR个人版(CCR) 迁移至 TCR企业版
- 支持基于Docker Registry V2搭建的docker镜像仓库服务 (如 腾讯云TCR个人版(CCR)/TCR企业版、Docker Hub、 Quay、 阿里云镜像服务ACR、 Harbor等)
- 支持自定义qps限速，避免迁移时对仓库造成过大压力
- 同步不落盘，提升同步速度
- 利用pipeline模型，提高任务执行效率
- 增量同步, 通过对同步过的镜像blob信息落盘，不重复同步已同步的镜像
- 并发同步，可以通过配置文件调整并发数
- 自动重试失败的同步任务，可以解决大部分镜像同步中的网络抖动问题
- 不依赖docker以及其他程序

## 模式
tcr镜像迁移工具具有两种运行模式：
- 通用模式：支持多种镜像仓库迁移
- 腾讯云CCR一键全量迁移模式：腾讯云TCR个人版(CCR) -> TCR企业版

## 架构
![image](https://github.com/tkestack/image-transfer/blob/main/docs/arch.png)

## 使用

### 下载和安装

在[releases](https://github.com/tkestack/image-transfer/releases)页面可下载源码以及二进制文件

### 手动编译
```
git clone https://github.com/tkestack/image-transfer.git
cd ./image-transfer

# 编译
make
```

## 使用方法

使用帮助：
```shell
./image-transfer -h
```


使用示例：通用模式
```
# 设置镜像鉴权配置文件为security.yaml，镜像迁移仓库规则文件为rule.yaml，默认registry为ccr.ccs.tencentyun.com
# 默认namespace为default，并发数为5，qps设置为100
./image-transfer --routines=5 --securityFile=./security.yaml --ruleFile=./rule.yaml --ns=default \
--registry=ccr.ccs.tencentyun.com --retry=3 --qps=100
```


使用示例：腾讯云CCR一键全量迁移模式：腾讯云TCR个人版(CCR) -> TCR企业版
```
# 打开ccr迁移模式ccrToTcr=true, 设置镜像鉴权配置文件为security.yaml，腾讯云secretId&配secretKey配置文件为secret.yaml，ccr和tcr所在地域均为ap-guangzhou（广州）
# tcr名称为tcr-test, 并发数为3（默认为5），失败重试次数为3（默认为2）
./image-transfer --ccrToTcr=true --routines=5 --securityFile=./security.yaml --secretFile=./secret.yaml --tcrName=tcr-test \
 --retry=3 --tcrRegion=ap-guangzhou --ccrRegion=ap-guangzhou
```

#### 腾讯云secret配置文件
```
ccr:
    secretId: xxx
    secretKey: xxx
tcr:
    secretId: xxx
    secretKey: xxx
```

#### 镜像鉴权配置文件
```
grant-test.tencentcloudcr.com:
  username: xxx
  password: xxx
grant-test2.tencentcloudcr.com:
  username: xxx
  password: xxx
registry.cn-hangzhou.aliyuncs.com:
  username: xxx
  password: xxx
ccr.ccs.tencentyun.com:
  username: xxx
  password: xxx
registry.hub.docker.com:
  username: xxx
  password: xxx
test-jingwei.tencentcloudcr.com:
  username: xxx
  password: xxx
```


#### 镜像迁移仓库配置文件
```
sichenzhao/private-test:xx: grant-test2.tencentcloudcr.com/xxx/xxx
```
