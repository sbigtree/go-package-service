package mypackage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"time"
	//"github.com/sourcegraph/conc/pool"
	//"go.mongodb.org/mongo-driver/bson"
	//"go.mongodb.org/mongo-driver/bson/primitive"
	//"go.mongodb.org/mongo-driver/mongo"
	//"go.uber.org/zap"
	//"log"
	//"time"
)

func DealExpireData(req event.EventMsg) error {
	start := time.Now()
	zap.S().Infof("req context %+v", req)
	zap.S().Infof("req111111 context %+v", req)
	paramsStr := string(req.Params)
	zap.S().Infof("req.Params: %+v", req.Params)
	zap.S().Infof("req params string: %s", paramsStr)

	var paramMap map[string]string
	if err := json.Unmarshal([]byte(paramsStr), &paramMap); err != nil {
		zap.S().Errorf("[Consumer] JSON 解析失败 params=%s err=%v", paramsStr, err)
		return err
	}

	idHex, ok := paramMap["id"]
	if !ok {
		zap.S().Errorf("[Consumer] id 字段不存在 params=%s", paramsStr)
		return fmt.Errorf("id 字段不存在")
	}

	id, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		zap.S().Errorf("[Consumer] ObjectID 解析失败 id=%s err=%v", idHex, err)
		return err
	}

	ctx := context.Background()
	collection := global.MongoDB.Collection("my_inventory_pack")
	filter := bson.M{"_id": id}

	var result bson.M
	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			zap.S().Warnf("[Consumer] Mongo 中不存在该数据，跳过 id=%v", id.Hex())
			return nil
		} else {
			zap.S().Warnf("[Consumer] 查询Mongo失败 id=%v err=%v", id.Hex(), err)
			return err
		}
	}

	// 成功拿到数据，做处理...
	zap.S().Infof("[Consumer] 查询到数据 id=%v result=%+v, 耗时=%v", id.Hex(), result, time.Since(start))
	return nil
}
