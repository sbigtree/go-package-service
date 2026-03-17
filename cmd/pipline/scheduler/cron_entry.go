package scheduler

import (
	"context"
	"github.com/go-redis/redis"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"time"
)

func StartCronPipeline(ctx context.Context, client *redis.Client, mongoDB *mongo.Database) {
	c := cron.New(cron.WithSeconds())
	cfg := TaskPipelineConfig{
		ProducerPoolSize: 1,
		ConsumerPoolSize: 20,
		ModifyPoolSize:   20,
		LockKey:          "task_pipeline_lock",
		LockTTL:          10 * time.Minute,
		MaxRetry:         3,
		RedisQueueKey:    "pipeline_failed_queue",
		Redis:            client,
		MongoDB:          mongoDB,
	}
	c.AddFunc("0/10 * * * * *", func() {
		defer func() {
			if r := recover(); r != nil {
				zap.S().Errorf("定时任务 panic %v", r)
			}
		}()
		select {
		case <-ctx.Done():
			zap.S().Info("任务已取消，跳过执行")
			return
		default:

		}
		zap.S().Info("定时任务触发")
		RunTaskPipeline(ctx, cfg)
	})
	c.Start()
	go func() {
		<-ctx.Done()
		zap.S().Info("停止cron调度器")
		c.Stop()
	}()
	//启动失败补偿消费者
	StartRedisRetryConsumer(ctx, client)
}
