package global

import (
	"github.com/sbigtree/go-package-service/cmd/appconf"
	"os"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/bwmarrin/snowflake"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-redis/redis"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DEV              string // 是否为开发环境
	DebugMode        = os.Getenv("DEBUG_MODE") == "true"
	CMDConfig        *appconf.CMDConfig
	DB               *gorm.DB
	RedisDB          *redis.Client
	AppConfigMaster  *appconf.AppConfigMaster
	AESIv            = []byte("0000000000000000") // 16 字节 IV
	SnowflakeNode    *snowflake.Node
	SnowflakeOnce    sync.Once
	ESClient         *elasticsearch.Client
	Once             sync.Once
	RocketMQProducer rocketmq.Producer
	RocketMQConsumer rocketmq.PushConsumer
	ZapLog           *zap.Logger
	MongoDB          *mongo.Database
	Cron             *cron.Cron
)

// 默认是 mq-goods-server  在 docker中配置环境变量  开发环境通过环境变量配置自定义自己的组，以免和测试环境抢资源
var _MQ_GROUP_NAME = os.Getenv("MQ_GROUP_NAME")
var _MQ_TOPIC_TAG = os.Getenv("MQ_TOPIC_TAG")

var MQGroup = struct {
	GroupName string
	TopicTag  string
}{
	GroupName: _MQ_GROUP_NAME,
	TopicTag:  _MQ_TOPIC_TAG,
}
