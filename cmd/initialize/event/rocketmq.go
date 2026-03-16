package event

import (
	"github.com/apache/rocketmq-client-go/v2/rlog"
	"github.com/sbigtree/go-package-service/cmd/global"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"
)

func InitRocketmq() {
	//设置日志级别
	InitSetRocketmqLogLevel()

	// 初始化rocketmq生产者
	InitRocketmqProducer()

	// 初始化rocketmq消费者
	err := InitRocketmqConsumer()
	if err != nil {
		panic(err)
		//syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}
}

// InitSetRocketmqLogLevel 初始化RocketMQ日志级别和输出路径
func InitSetRocketmqLogLevel() {
	// 1. 获取今天的日期作为目录名
	today := time.Now().Format("2006-01-02") // 格式化为 YYYY-MM-DD

	// 2. 构建日志目录路径
	logDir := filepath.Join(".", "logs", today)
	logFile := filepath.Join(logDir, "rocketmq.log")

	// 3. 创建目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		zap.S().Errorf("创建RocketMQ日志目录失败: %v", err)
		return
	}

	// 4. 创建日志文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		zap.S().Errorf("创建RocketMQ日志文件失败: %v", err)
		return
	}
	file.Close()

	// 5. 设置日志等级和输出路径
	rlog.SetLogLevel("error") // 或 "info"/"debug" 测试使用
	if err := rlog.SetOutputPath(logFile); err != nil {
		zap.S().Errorf("设置RocketMQ日志输出路径失败: %v", err)
		return
	}

	zap.S().Info("✅ RocketMQ日志配置成功", zap.String("path", logFile))
}

// 初始化rocketmq生产者
func InitRocketmqProducer() {
	var err error

	//拼接RocketMQ的NameServer地址
	nameServe := global.AppConfigMaster.ROCKETMQ_HOST + ":" + strconv.Itoa(global.AppConfigMaster.ROCKETMQ_PORT)

	global.RocketMQProducer, err = rocketmq.NewProducer(producer.WithNameServer([]string{nameServe}),
		producer.WithGroupName(global.MQGroup.GroupName))
	if err != nil {
		zap.S().Error("初始化rocketmq生产者失败" + err.Error())
		return
	}
	zap.S().Info("✅ 初始化rocketmq生产者成功")
	err = global.RocketMQProducer.Start()
	if err != nil {
		zap.S().Error("生产者启动失败", err)
		return
	}
	// 发送空消息注册topic  样例
	//_, err = global.RocketMQProducer.SendSync(context.Background(), &primitive.Message{
	//	Topic: "outer_to_master_order_channel",
	//	Body:  []byte(""),
	//})
	//if err != nil {
	//	zap.S().Error("outer_to_master_order_channel发送消息失败", err)
	//	return
	//}
	zap.S().Info("✅ rocketmq生产者启动成功")
}

// 初始化 RocketMQ 消费者
func InitRocketmqConsumer() error {
	var err error

	//拼接RocketMQ的NameServer地址
	nameServe := global.AppConfigMaster.ROCKETMQ_HOST + ":" + strconv.Itoa(global.AppConfigMaster.ROCKETMQ_PORT)

	global.RocketMQConsumer, err = rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{nameServe}),
		consumer.WithGroupName(global.MQGroup.GroupName),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromFirstOffset),
		//consumer.WithConsumeMessageBatchMaxSize(1),
	)
	if err != nil {
		zap.S().Error("初始化rocketmq消费者失败" + err.Error())
		return err
	}
	zap.S().Infof("✅ 初始化rocketmq消费者成功 GroupName %s  MQ_TOPIC_TAG=[%s]", global.MQGroup.GroupName, global.MQGroup.TopicTag)
	return nil
}
