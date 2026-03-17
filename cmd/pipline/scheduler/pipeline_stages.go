package scheduler

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"log"
	"time"
)

type RetryItem struct {
	Data  interface{} `json:"data"`
	Retry int         `json:"retry"`
}

// 模拟 Mongo 流式读取（生产替换 cursor）
func fetchMongoStream(ctx context.Context, handler func(item interface{})) {
	data := []interface{}{"a", "b", "c"}
	for _, item := range data {
		select {
		case <-ctx.Done():
			return
		default:
			handler(item)
		}
	}
}

// 生产阶段
func produceFromMongoAsync(ctx context.Context, out chan<- interface{}, pool *GoroutinePool, db *mongo.Database) {
	collection := db.Collection("my_inventory_pack")
	pageSize := int64(1000) // 每次批量读取1000条，可根据实际内存调节
	var lastExpireTime int64 = 0
	var lastID primitive.ObjectID = primitive.NilObjectID // 用于分页的 _id 记录

	for {
		now := time.Now().Unix()
		select {
		case <-ctx.Done():
			zap.S().Info("生产阶段 ctx 已取消，停止读取 MongoDB")
			return
		default:

		}
		filter := bson.M{
			"expire_time": bson.M{"$lt": now}, // 筛选 expire_time < 当前时间
			"$or": []bson.M{
				{
					"expire_time": bson.M{
						"$gt": lastExpireTime,
					},
				},
				{
					"expire_time": lastExpireTime,
					"_id": bson.M{
						"$gt": lastID,
					},
				},
			},
		}
		findOpts := options.Find()
		findOpts.SetSort(bson.D{{Key: "expire_time", Value: 1}, {Key: "_id", Value: 1}})
		findOpts.SetLimit(pageSize)
		cursor, err := collection.Find(ctx, filter, findOpts)
		if err != nil {
			zap.S().Error("MongoDB 查询失败:", err)
			return
		}

		count := 0
		for cursor.Next(ctx) {
			select {
			case <-ctx.Done():
				cursor.Close(ctx)
				zap.S().Info("生产阶段 ctx 已取消，停止读取 MongoDB")
				return
			default:
			}

			var item map[string]interface{}
			if err := cursor.Decode(&item); err != nil {
				zap.S().Error("MongoDB 解码失败:", err)
				continue
			}
			//tempjsonBytes, err := bson.MarshalExtJSON(item, true, true)
			//if err != nil {
			//	zap.S().Error("序列化 MongoDB 数据为 JSON 失败:", err)
			//} else {
			//	zap.S().Infof("读取 MongoDB 数据 JSON: %s", string(tempjsonBytes))
			//}
			//zap.S().Infof("读取MongoDB数据:%+v", item)
			expireTime, ok := item["expire_time"].(int64)
			if !ok {
				switch v := item["expire_time"].(type) {
				case int32:
					expireTime = int64(v)
				case float64:
					expireTime = int64(v)
				default:
					zap.S().Warn("expire_time 类型异常:", item["expire_time"])
					continue
				}
			}
			id, ok := item["_id"].(primitive.ObjectID)
			if !ok {
				zap.S().Warn("ObjectID解析失败")
				continue
			}
			idCopy := id
			zap.S().Infof("读取MongoDB _id数据:%+v", idCopy)
			pool.Go(func() {
				select {
				case out <- idCopy:
				case <-ctx.Done():
				}
			})
			lastExpireTime = expireTime
			lastID = id
			count++
		}

		cursor.Close(ctx)
		if count == 0 {
			zap.S().Info("没有更多符合条件的数据，结束分页")
			break
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func consumerStageAsync(ctx context.Context, in <-chan interface{}, out chan<- interface{}, pool *GoroutinePool, cfg TaskPipelineConfig) {
	collection := cfg.MongoDB.Collection("my_inventory_pack")
	for item := range in {
		it := item
		zap.S().Infof("[Consumer]收到数据:type=%T value=%+v", it, it)
		pool.Go(func() {
			start := time.Now()

			id, ok := it.(primitive.ObjectID)
			if !ok {
				zap.S().Errorf("[Consumer]类型断言失败 item=%+v", it)
				return
			}
			filter := bson.M{"_id": id}
			var result bson.M
			//查mongo表中行记录是否存在
			err := collection.FindOne(ctx, filter).Decode(&result)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					zap.S().Warnf("[Consumer] Mongo 中不存在该数据，跳过 id=%v", id.Hex())
				} else {
					zap.S().Warnf("[Consumer] 查询Mongo失败 id=%v err=%v", id.Hex(), err)
				}
				return
			}

			zap.S().Infof("[Consumer] Mongo校验通过 id=%v 数据=%+v", id.Hex(), result)

			zap.S().Infof("[Consumer]开始处理 item=%+v", it)
			log.Printf("start= %v\n", start)
			success := retry(ctx, cfg.MaxRetry, func() error {
				err := writeTrace(it)
				if err != nil {
					zap.S().Warnf("[Consumer] writeTrace 失败 item=%+v err=%v", it, err)
				} else {
					zap.S().Warnf("[Consumer] writeTrace 成功 item=%+v", it)
				}
				return err
			})
			if !success {
				zap.S().Errorf("[Consumer]最终失败，写入Redis重试队列 item=%+v", it)
				pushToRedisQueue(ctx, cfg.Redis, cfg.RedisQueueKey, RetryItem{Data: it, Retry: 1})
			}
			select {
			case out <- it:
				zap.S().Infof("[Consumer]已写入下游通道 item=%+v", it)
			case <-ctx.Done():
				zap.S().Warn("[Consumer] ctx取消 未写入下游通道")
			}
			zap.S().Infof("[Consumer]完成 item=%+v success=%v item耗时:%v", it, success, time.Since(start))
		})
	}
}

func modifyStageAsync(ctx context.Context, in <-chan interface{}, out chan<- interface{}, pool *GoroutinePool, cfg TaskPipelineConfig) {
	for item := range in {
		it := item
		pool.Go(func() {
			start := time.Now()
			success := retry(ctx, cfg.MaxRetry, func() error {
				return modifyDB(it)
			})
			if !success {
				pushToRedisQueue(ctx, cfg.Redis, cfg.RedisQueueKey, RetryItem{
					Data:  it,
					Retry: 1,
				})
			}
			select {
			case out <- it:
			case <-ctx.Done():
			}
			zap.S().Infof("修改阶段处理 item耗时:%v", time.Since(start))
		})
	}
}

func fetchMongoData(ctx context.Context) []interface{} {
	return []interface{}{"a", "b", "c"}
}

func writeTrace(item interface{}) error {
	zap.S().Info("写留痕:", item)
	return nil
}

func modifyDB(item interface{}) error {
	zap.S().Info("修改数据库:", item)
	return nil
}

func retry(ctx context.Context, maxRetry int, fn func() error) bool {
	for i := 0; i < maxRetry; i++ {
		select {
		case <-ctx.Done():
			zap.S().Warn("retry 被中断")
			return false
		default:
		}

		if err := fn(); err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return true
	}
	return false
}

func pushToRedisQueue(ctx context.Context, rdb *redis.Client, key string, item RetryItem) bool {
	bytes, err := json.Marshal(item)
	if err != nil {
		return false
	}
	rdb.RPush(key, bytes)
	return true
}
