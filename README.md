# GOBuild
There are some projects developed with golang

下面是各个目录内的项目的介绍：

## HTTP Stress
使用go写的一个H5游戏项目（node）的并发测试用例，模拟了一个用户完整的游戏过程： 
- 去微信服务认证登陆信息，并获取code和uid
- 根据获取的code和uid去H5游戏服务认证，获取tocken
- 根据tocken和uid选择对应的游戏项目id开始游戏
- 获取游戏服务返回的中奖标识bingo并发送给游戏服务结束游戏
整个用例发送4次http请求。并且使用go来进行压力测试在并发上变得非常容易，需要注意的是在并发中进行recover异常捕获
