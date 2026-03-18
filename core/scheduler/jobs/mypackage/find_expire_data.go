package mypackage

import (
	"context"
	"fmt"
	mongo2 "github.com/sbigtree/go-db-model/v2/mongo/models"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/scheduler/consumer/mypackage"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/sbigtree/go-package-service/cmd/global"
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
		"take_usable": 1, //可提取状态
		"Tags": bson.M{
			"$elemMatch": bson.M{
				"type":  "tradable_time",
				"value": bson.M{"$lt": fmt.Sprintf("%d", now)}, // tradable_time 小于当前时间
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
	defer cursor.Close(ctx)
	grouped := make(map[int32][]primitive.ObjectID)

	count := 0
	for cursor.Next(ctx) {
		var result mongo2.MyInventoryPack
		if err := cursor.Decode(&result); err != nil {
			zap.S().Error("MongoDB 解码失败:", err)
			continue
		}
		var steamAID int32
		for _, origin := range result.Origins {
			if origin.Type == "steam_aid" {
				var aid int
				_, err := fmt.Sscan(origin.Value, "%d", &aid)
				if err == nil {
					steamAID = int32(aid)
				} else {
					zap.S().Warn("steam_aid 转换失败", origin.Value)
				}
				break
			}
		}

		if steamAID == 0 {
			zap.S().Warn("未找到steam_aid", "id", result.ID.Hex())
			continue
		}

		grouped[steamAID] = append(grouped[steamAID], result.ID)
		count++
	}

	for steamAID, ids := range grouped {
		if len(ids) == 0 {
			continue
		}
		param := mypackage.DealExpireDataParam{
			Ids: ids,
		}
		msg := event.NewEventMsg(map[string]interface{}{
			"ids":       param.Ids,
			"steam_aid": steamAID,
		})
		global.ExpireDataChannel <- msg
	}

	zap.S().Infof("生产任务完成，本次共读取 %d 条数据", count)
}
