package mypackage

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FindExpireData 每次生产任务只读取一页数据（1000条）
func FindExpireData() {
	ctx := context.Background()
	collection := global.MongoDB.Collection(global.InventoryPackTable)

	pageSize := int64(1000) // 每次查询1000条
	now := time.Now().Unix()

	filter := bson.M{
		"expire_time": bson.M{"$lt": now},
		"is_del":      0, // 0 表示正常状态
	}

	findOpts := options.Find()
	findOpts.SetSort(bson.D{{Key: "expire_time", Value: 1}, {Key: "_id", Value: 1}})
	findOpts.SetLimit(pageSize)

	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		zap.S().Error("MongoDB 查询失败:", err)
		return
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		var item map[string]interface{}
		if err := cursor.Decode(&item); err != nil {
			zap.S().Error("MongoDB 解码失败:", err)
			continue
		}

		id, ok := item["_id"].(primitive.ObjectID)
		if !ok {
			zap.S().Warn("ObjectID解析失败")
			continue
		}

		// 写入通道，如果通道满会阻塞
		global.ExpireDataChannel <- event.NewEventMsg(map[string]interface{}{
			"id": id,
		})
		count++
	}

	zap.S().Infof("生产任务完成，本次共读取 %d 条数据", count)
}
