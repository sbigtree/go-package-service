package initialize

import (
	"github.com/sbigtree/go-package-service/cmd/initialize/event"
)

func init() {
	//初始化AppConfig配置信息
	event.InitAppConfig()
	//初始化zap日志
	event.InitZapLogger()
	//初始化mysql连接
	event.InitMysql()
	//初始化redis连接
	event.InitRedis()
	//初始化mongodb连接
	event.InitMongoDB()
	//初始化elasticsearch连接
	event.InitElasticsearchClient()
	//初始化rocketmq
	//event.InitRocketmq()
	event.InitSnowflake(1)
	//初始化总路由
	event.InitRouter()
	//初始化定时任务
	event.InitScheduler()
	//初始化steam_tools服务连接
	event.InitSteamToolsClient()
}
