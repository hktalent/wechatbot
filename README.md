# wechatbot
> 最近chatGPT异常火爆，本项目可以将个人微信化身GPT机器人，
> 项目基于[openwechat](https://github.com/eatmoreapple/openwechat) 开发。

```azure 

EMAIL_HOST_USER = "briefly.ai.tech@gmail.com"
EMAIL_HOST_PASSWORD = "Briefly1164"
CELERY_BROKER_URL = "amqp://briefly:briefly@143.244.190.3:5672/"
openai_api = "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"

SECRET_KEY = 'd!^in%v44tknzw=8b5^*=i#=_3sc=3nqt6#=(okywu2-p+^gly'

        
sk-LS5Pgc9DaNlbholGwJu6N3BlbkFJD3hbVFYOgK9mxuNU3rOS

```


### 目前实现了以下功能
 * 提问增加上下文，更接近官网效果
 * 机器人群聊@回复
 * 机器人私聊回复
 * 好友添加自动通过

# 使用前提
> * ~~目前只支持在windows上运行因为需要弹窗扫码登录微信，后续会支持linux~~   已支持
> * 有openai账号，并且创建好api_key，注册事项可以参考[此文章](https://juejin.cn/post/7173447848292253704) 。
> * 微信必须实名认证。

# 注意事项
> * 项目仅供娱乐，滥用可能有微信封禁的风险，请勿用于商业用途。
> * 请注意收发敏感信息，本项目不做信息过滤。

# 使用docker运行

你可以使用docker快速运行本项目。

`第一种：基于环境变量运行`

```sh
# 运行项目
$ docker run -itd --name wechatbot -e ApiKey=xxxx -e AutoPass=false -e SessionTimeout=60 docker.mirrors.sjtug.sjtu.edu.cn/qingshui869413421/wechatbot:latest

# 查看二维码
$ docker logs -f wechatbot
```

运行命令中映射的配置文件参考下边的配置文件说明。

`第二种：基于配置文件挂载运行`

```sh
# 复制配置文件，根据自己实际情况，调整配置里的内容
cp config.dev.json config.json  # 其中 config.dev.json 从项目的根目录获取

# 运行项目
docker run -itd --name wechatbot -v ./config.json:/app/config.json docker.mirrors.sjtug.sjtu.edu.cn/qingshui869413421/wechatbot:latest

# 查看二维码
$ docker logs -f wechatbot
```

其中配置文件参考下边的配置文件说明。

# 快速开始
> 非技术人员请直接下载release中的[压缩包](https://github.com/hktalent/wechatbot/releases/tag/v1.1.1) ，解压运行。
````
# 获取项目
git clone https://github.com/hktalent/wechatbot.git

# 进入项目目录
cd wechatbot

# 复制配置文件
copy config.dev.json config.json

# 启动项目
go run main.go

# linux编译，守护进程运行（可选）
# 编译
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o wechatbot  ./main.go
# 守护进程运行
nohup ./wechatbot > run.log &
````

# 配置文件说明
````
{
"api_key": "your api key",
"auto_pass": true,
"session_timeout": 60
}

api_key：openai api_key
auto_pass:是否自动通过好友添加
session_timeout：会话超时时间，默认60秒，单位秒，在会话时间内所有发送给机器人的信息会作为上下文。
````

