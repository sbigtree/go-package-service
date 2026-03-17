package scheduler

import (
	"context"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"time"
)

type TaskPipelineConfig struct {
	ProducerPoolSize int
	ConsumerPoolSize int
	ModifyPoolSize   int
	LockKey          string
	LockTTL          time.Duration
	MaxRetry         int
	RedisQueueKey    string
	Redis            *redis.Client
	MongoDB          *mongo.Database
}

func RunTaskPipeline(ctx context.Context, cfg TaskPipelineConfig) {
	startTime := time.Now()
	zap.S().Info("流水线任务开始")

	ok, err := cfg.Redis.SetNX(cfg.LockKey, 1, cfg.LockTTL).Result()

	if err != nil {
		zap.S().Error("获取分部署锁失败:", err)
		return
	}

	if !ok {
		zap.S().Info("任务正在执行，跳过本次调度")
		return
	}

	defer func() {
		cfg.Redis.Del(cfg.LockKey)
		zap.S().Info("分布式锁释放")
	}()
	rawChan := make(chan interface{}, 1000)
	stageChan := make(chan interface{}, 1000)
	modifyChan := make(chan interface{}, 1000)
	producePool := NewGoroutinePool(cfg.ProducerPoolSize)
	consumerPool := NewGoroutinePool(cfg.ConsumerPoolSize)
	modifyPool := NewGoroutinePool(cfg.ModifyPoolSize)
	//异步生产
	go func() {
		produceFromMongoAsync(ctx, rawChan, producePool, cfg.MongoDB)
		producePool.Wait()
		close(rawChan)
		zap.S().Info("生产阶段完成，rawChan关闭")
	}()
	//异步消费 + 写留痕 + Redis队列缓存
	go func() {
		consumerStageAsync(ctx, rawChan, stageChan, consumerPool, cfg)
		consumerPool.Wait()
		close(stageChan)
		zap.S().Info("消费阶段完成，stageChan关闭")
	}()

	//异步修改
	go func() {
		modifyStageAsync(ctx, stageChan, modifyChan, modifyPool, cfg)
		modifyPool.Wait()
		close(modifyChan)
		zap.S().Info("修改阶段完成，modifyChan关闭")
	}()

	go func() {
		for item := range modifyChan {
			zap.S().Info("最终处理item:", item)
		}
	}()
	// 等所有执行完
	producePool.Wait()
	consumerPool.Wait()
	modifyPool.Wait()

	zap.S().Infof("流水线任务结束，总耗时：%v", time.Since(startTime))
}
