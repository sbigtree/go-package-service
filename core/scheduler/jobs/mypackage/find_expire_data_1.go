package mypackage

import (
	"context"
	mongo2 "github.com/sbigtree/go-db-model/v2/mongo/models"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/scheduler/util/upackage"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/sbigtree/go-package-service/cmd/global"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FindExpireData1 每次生产任务只读取一页数据（1000条）
func FindExpireData1() {
	ctx := context.Background()
	collection := global.MongoDB.Collection(global.InventoryPackTable)

	pageSize := int64(1000) // 每次查询1000条
	now := time.Now().Unix()
	zap.S().Infof("now=%v", now)
	ids := []string{
		//"69bd37ab43f4e3d6928d3db4",
		//"69bd37a04dce64630aaf6c60",
		//"69bd3793a06c2e92d09a4640",
		//"69bd357390ea3d32ebef8c12",
		//"69bd359ee1bb5b117376c3ad",
		//"69bd35b360d8427a32327942",
		"69bd4ce8b6c8aab3a1c21cb5",
		"69bd4cf925c61ccbbf7cd230",
		"69bd4ae64e6c5d43715af8c0",
		"69bd4af74f547501ee22a90b",
	}

	objectIDs := make([]primitive.ObjectID, 0, len(ids))

	for _, id := range ids {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			zap.S().Warnf("非法ObjectID: %s err=%v", id, err)
			continue
		}
		objectIDs = append(objectIDs, oid)
	}
	filter := bson.M{
		//"expire_time": bson.M{"$lt": now},
		"_id": bson.M{
			"$in": objectIDs,
		},
		"is_del":      0, // 0 表示正常状态
		"take_usable": 1, //可提取状态
		//"Tags": bson.M{
		//	"$elemMatch": bson.M{
		//		"type":  "tradable_time",
		//		"value": bson.M{"$lt": fmt.Sprintf("%d", now)}, // tradable_time 小于当前时间
		//	},
		//},
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
				steamAIdInt, err := strconv.Atoi(origin.Value)
				if err != nil {
					zap.S().Warnf("steam_aid 转换失败%v %v %T", origin.Value, err, origin.Value)
				}
				steamAID = int32(steamAIdInt)
				break
			}
		}

		if steamAID == 0 {
			zap.S().Warnf("未找到steam_aid id=%s %v", result.ID.Hex(), result.Origins)
			continue
		}

		grouped[steamAID] = append(grouped[steamAID], result.ID)
		count++
	}

	zap.S().Infof(" current grouped count=%d", len(grouped))
	for steamAID, ids := range grouped {
		if len(ids) == 0 {
			continue
		}
		param := upackage.DealExpireDataParam{
			Ids: ids,
		}
		zap.S().Infof(" current grouped ids_count=%d", len(param.Ids))
		msg := event.NewEventMsg(map[string]interface{}{
			"ids":       param.Ids,
			"steam_aid": steamAID,
		})
		zap.S().Infof("write expire data channel %v", map[string]interface{}{
			"ids":       param.Ids,
			"steam_aid": steamAID,
		})
		global.ExpireDataChannel1 <- msg
	}

	zap.S().Infof("生产任务完成，本次共读取 %d 条数据", count)
}
