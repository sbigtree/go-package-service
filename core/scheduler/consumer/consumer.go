package consumer

import (
	"context"
	"github.com/google/uuid"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/scheduler/consumer/mypackage"
	consumer_test "github.com/sbigtree/go-package-service/core/scheduler/consumer/test"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// 定义每个任务的最大并发数 测试阶段设为1000
var (
	TestSem                 = NewConcurrencySem(50, "TestSem")
	CalculatePackageSem     = NewConcurrencySem(1, "CalculatePackageSem")
	CheckSendOfferStatusSem = NewConcurrencySem(1, "CheckSendOfferStatusSem")
)
var activeCounts = sync.Map{}

// 启动内部消费者
// 修改 StartInternalConsumer，移除 wg 参数
func StartInternalConsumer(ctx context.Context) {
	zap.S().Info("🎯 Internal Inventory Consumer 启动")
	consume[event.EventMsg](ctx, global.TestChannel, "TestSem", TestSem, func(msg event.EventMsg) {
		err := consumer_test.Test1(msg)
		if err != nil {
			zap.S().Errorf("TestChannel", err)

		}
	})
	consume[event.EventMsg](ctx, global.ExpireDataChannel, "CalculatePackageSem", CalculatePackageSem, func(msg event.EventMsg) {
		err := mypackage.DealExpireData(msg)
		if err != nil {
			zap.S().Errorf("ExpireDataChannel", err)
		}
	})
	consume[event.EventMsg](ctx, global.CheckSendOfferChannel, "CheckSendOfferStatusSem", CheckSendOfferStatusSem, func(msg event.EventMsg) {
		err := mypackage.DealExpireData(msg)
		if err != nil {
			zap.S().Errorf("CheckSendOfferChannel", err)
		}
	})
}

func consume[T any](
	ctx context.Context,
	ch <-chan T,
	tag string,
	sem *ConcurrencySem,
	handler func(T),
) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				zap.S().Infof("[%s] 监听停止", tag)
				return

			case req, ok := <-ch:
				if !ok {
					zap.S().Warnf("[%s] 通道已关闭", tag)
					return
				}

				// 获取并发令牌（这里阻塞是“可控的”）
				sem.Acquire()

				reqCopy := req
				go func() {
					defer func() {
						// 兜底释放令牌，避免泄漏
						sem.Release()
						//zap.S().Debugf("[%s] 释放并发令牌 当前并发 %d/%d 队列剩余 %d ", tag, sem.Current(), sem.Max(), len(ch))
					}()

					val, _ := activeCounts.LoadOrStore(tag, new(int64))
					counter := val.(*int64)
					atomic.AddInt64(counter, 1)
					defer atomic.AddInt64(counter, -1)

					SafeRun(tag, func() {
						taskId := uuid.NewString()
						zap.S().Debugf("[%s] 开始执行 uid_%s 当前并发 %d/%d 队列剩余 %d", tag, taskId, sem.Current(), sem.Max(), len(ch))
						handler(reqCopy)
						zap.S().Debugf("[%s] 执行完成 uid_%s  当前并发 %d/%d 队列剩余 %d", tag, taskId, sem.Current(), sem.Max(), len(ch))

					})
				}()
			}
		}
	}()
}

// 可选：定义默认获取信号量超时时间，避免永久阻塞
//const defaultSemAcquireTimeout = 10 * time.Second

// 带有并发限制的 SafeGo
func SafeGoWithLimit(tag string, sem *semaphore.Weighted, fn func()) {
	if err := sem.Acquire(context.Background(), 1); err != nil {
		zap.S().Warnf("[%s] goroutine 获取信号量失败: %v", tag, err)
		return
	}
	zap.S().Infof("[%s] goroutine 获取信号量成功: %v", tag, sem)
	go func() {
		defer sem.Release(1)
		// 调用你的 SafeGo
		SafeRun(tag, fn)
	}()
}

func SubscribeTopic(topic string, worker *WorkerPool, handler func(msg *primitive.MessageExt) error) error {
	return global.RocketMQConsumer.Subscribe(
		topic,
		consumer.MessageSelector{
			Type:       consumer.TAG,
			Expression: global.MQGroup.TopicTag,
		},
		func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {

			for _, msg := range msgs {
				if len(msg.Body) == 0 {
					continue
				}

				worker.Go(func() {
					err := handler(msg)
					if err != nil {
						zap.S().Error("handler failed", zap.Error(err), zap.ByteString("body", msg.Body))
					}
				}, msg)
			}

			return consumer.ConsumeSuccess, nil
		},
	)
}

// WorkerPool 控制并发
type WorkerPool struct {
	sem chan struct{}
}

func NewWorkerPool(size int) *WorkerPool {
	return &WorkerPool{sem: make(chan struct{}, size)}
}

func (w *WorkerPool) Go(fn func(), msg *primitive.MessageExt) {
	w.sem <- struct{}{}
	go func() {
		defer func() {
			<-w.sem
			if r := recover(); r != nil {
				zap.S().Error("panic recovered",
					zap.Any("error", r),
					zap.ByteString("body", msg.Body),
					zap.String("topic", msg.Topic),
				)
			}
		}()
		fn()
	}()
}

// 安全执行 goroutine，防止 panic 崩溃
func SafeGo(name string, fn func()) {
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

func SafeRun(name string, fn func()) {
	// 这里没有 go func() 了，直接在当前协程执行
	defer func() {
		if r := recover(); r != nil {
			zap.S().Errorf("[%s] panic recovered: %v\n堆栈信息:\n%s", name, r, debug.Stack())
		}
	}()
	fn()
}
