package mq

import (
	"context"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/store/mq"
	"go.uber.org/zap"
	"runtime/debug"
)

// startRocketMQConsumer 启动 RocketMQ 消费者
func StartRocketMQConsumer() error {
	//订阅 mq.InventoryRisk 主题
	err := global.RocketMQConsumer.Subscribe(mq.Test, consumer.MessageSelector{
		Type: consumer.TAG,
		//Expression: global.MQGroup.TopicTag, //
	}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {

		for _, msg := range msgs {
			zap.S().Info("订阅 InventoryRisk 收到消息", msg.GetTags())
			if len(msg.Body) == 0 {
				continue
			}
			// 构造 EventMsg
			eventMsg := event.EventMsg{
				Params: msg.Body,
			}
			select {
			case global.TestChannel <- eventMsg:
				// ok
			default:
				zap.S().Error("RiskInventoryChannel full, retry later")
				return consumer.ConsumeRetryLater, nil
			}
		}
		return consumer.ConsumeSuccess, nil
	})
	if err != nil {
		zap.S().Error("订阅 InventoryRisk 主题失败", zap.Error(err))
		return err
	}

	err = global.RocketMQConsumer.Start()
	if err != nil {
		zap.S().Error("RocketMQ 消费者启动失败", zap.Error(err))
		//panic(err)
		return err
	}
	zap.S().Info("✅ RocketMQ 消费者已启动，等待消息...")
	return nil
}

// 优雅关闭 RocketMQ 消费者
func CloseRocketMQConsumer(ctx context.Context) {
	// 这里假设你有一个全局的消费者实例
	// 如果你有多个消费者实例，可以根据需要传递具体的实例
	global.RocketMQConsumer.Shutdown()
	select {
	case <-ctx.Done():
		zap.S().Info("RocketMQ 消费者已优雅关闭")
	}
}

// 安全执行 goroutine，防止 panic 崩溃
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				zap.S().Errorf("goroutine panic: %v\n堆栈信息:\n%s", r, debug.Stack())
				// 可选报警扩展
			}
		}()
		fn()
	}()
}
