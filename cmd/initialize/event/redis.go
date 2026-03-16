package event

import (
	"github.com/sbigtree/go-package-service/cmd/global"
	"strconv"

	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

//初始化redis连接

func InitRedis() {
	//配置Redis连接信息
	global.RedisDB = redis.NewClient(&redis.Options{
		Addr:     global.AppConfigMaster.REDISHOST + ":" + strconv.Itoa(global.AppConfigMaster.REDISPORT),
		Password: global.AppConfigMaster.REDISPASSWORD, // no password set
		DB:       global.AppConfigMaster.REDISDB,       // use default DB
	})
	//连接Redis数据库
	_, err := global.RedisDB.Ping().Result()
	if err != nil {
		zap.S().Error("Redis数据库连接异常失败" + err.Error())
		panic("Redis数据库连接失败" + err.Error())
	}
	zap.S().Info("✅ Redis数据库连接成功")
}
