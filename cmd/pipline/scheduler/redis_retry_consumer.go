package scheduler

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
	"time"
)

func StartRedisRetryConsumer(ctx context.Context, rdb *redis.Client) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				zap.S().Info("Redis补偿消费者退出")
				return
			default:
				consumerFailedQueue(ctx, rdb)
				time.Sleep(10 * time.Second)
			}
		}
	}()
}

func consumerFailedQueue(ctx context.Context, rdb *redis.Client) {
	key := "pipeline_failed_queue"

	for {
		select {
		case <-ctx.Done():
			zap.S().Info("停止消费失败队列")
			return
		default:
		}
		data, err := rdb.LPop(key).Result()
		if err == redis.Nil {
			return
		}
		if err != nil {
			zap.S().Error("redis读取失败:", err)
			return
		}
		var item RetryItem
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			zap.S().Error("反序列化失败:", err)
			continue
		}
		processFailedItem(ctx, rdb, key, item)
	}
}
func processFailedItem(ctx context.Context, rdb *redis.Client, key string, item RetryItem) {
	if item.Retry > 5 {
		zap.S().Error("超过最大重试次数丢弃:", item)
		return
	}
	start := time.Now()
	success := retry(ctx, 3, func() error {
		return modifyDB(item)
	})
	if !success {
		select {
		case <-ctx.Done():
			zap.S().Warn("任务取消，不再入队:", item)
			return
		default:

		}
		item.Retry++
		bytes, _ := json.Marshal(item)
		rdb.RPush(key, bytes)
		zap.S().Warn("补偿失败，重新入队:", item)
	} else {
		zap.S().Infof("补偿成功，耗时:%v, item:%v", time.Since(start), item)
	}
}
