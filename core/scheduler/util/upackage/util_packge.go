package upackage

import (
	"context"
	"github.com/sbigtree/go-db-model/v2/models"
	"github.com/sbigtree/go-package-service/cmd/global"
	steam_tools_grpc "github.com/sbigtree/steam-tools-grpc/go/generator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"time"
)

type OfferGroup struct {
	OfferID          string
	ReceivedSteamAid int32
	PackID           string
	CategoryName     string
	AssetID          string
}
type DealExpireDataParam struct {
	Ids      []primitive.ObjectID `json:"ids"` // mongodb表主键id
	SteamAID int32                `json:"steam_aid"`
	OfferID  string               `json:"offer_id"`
}

// 资产数量
const TotalAssetsNum = 1000

func UpdateTradeTransferLogs(offerId, assetId string) bool {
	if err := global.DB.Model(&models.InventoryPackTradeTransferLog{}).
		Where("offer_id = ? and offer_status = 2 and asset_id= ? ", offerId, assetId).
		Update("offer_status", 3).Error; err != nil {
		zap.S().Errorf("update table inventory_pack_trade_transfer_logs error offerId=%v", offerId)
		return false
	}
	return true
}

func UpdateTradeTransferLogsBatch(packIds []string, offerId string, newStatus int) bool {
	if len(packIds) == 0 || offerId == "" {
		zap.S().Warnf("UpdateTradeTransferLogsBatch: empty packIds or offerId")
		return false
	}
	if err := global.DB.Model(&models.InventoryPackTradeTransferLog{}).
		Where("offer_id = ? AND pack_id IN ? AND offer_status = 2", offerId, packIds).
		Update("offer_status", newStatus).Error; err != nil {
		zap.S().Errorf("update table inventory_pack_trade_transfer_logs error offerId=%v, packIds=%v, err=%v", offerId, len(packIds), err)
		return false
	}
	return true
}

func CheckOfferTradeStatus(result OfferGroup) bool {
	if global.SteamTools == nil || global.SteamTools.OfferServerClient == nil {
		zap.S().Errorf("steamTools client is nil")
		return false
	}
	steamAid := result.ReceivedSteamAid
	offerId := result.OfferID
	tempAppId := int32(730)
	resp, err := global.SteamTools.OfferServerClient.GetOfferDetail(context.Background(), &steam_tools_grpc.OfferGetOfferDetailRequest{
		SteamAid:     &steamAid,
		Tradeofferid: &offerId,
		Appid:        &tempAppId,
	})

	if err != nil || resp == nil {
		zap.S().Errorf("调用steam_tools服务失败 response=%v steamAid=%v tempTradeofferid=%v err=%v", resp, steamAid, offerId, err)
		return false
	}

	if resp.Success == nil || !*resp.Success {
		zap.S().Errorf("调用steam_tools服务失败 success value,  response=%v steamAid=%v tempTradeofferid=%v", resp, steamAid, offerId)
		return false
	}

	if resp.Data == nil || resp.Data.TradeOfferState == nil {
		zap.S().Errorf("调用steam_tools服务失败 data value,  data=%v steamAid=%v tempTradeofferid=%v",
			resp, steamAid, offerId)
		return false
	}

	if *resp.Data.TradeOfferState != 3 {
		zap.S().Infof("调用steam_tools服务成功 未交易成功 TradeOfferState value,  steamAid=%v tempTradeofferid=%v tradeOfferState=%v",
			steamAid, offerId, *resp.Data.TradeOfferState)
		return false
	}
	zap.S().Infof("调用steam_tools服务成功 交易成功 TradeOfferState value,  steamAid=%v tempTradeofferid=%v resp=%v",
		steamAid, offerId, resp)
	return true
}
func UpdateMongoLineDataById(id string) bool {
	ctx := context.Background()
	collection := global.MongoDB.Collection("my_inventory_pack")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		zap.S().Errorf("无效的ID格式:%v %v", err, id)
		return false
	}
	filter := bson.M{
		"_id":    objID,
		"is_del": bson.M{"$ne": 1},
	}
	update := bson.M{
		"$set": bson.M{
			"is_del":      1,
			"take_usable": 0, //已提取
			"update_time": time.Now().UnixMilli(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		zap.S().Errorf("更新失败: %v %v", err, id)
		return false
	}
	zap.S().Infof("UpdateMongoLineDataById success id=%v matched=%d modified=%d",
		id, result.MatchedCount, result.ModifiedCount)
	return true
}

func BatchUpdateMongoLineDataByIds(ids []primitive.ObjectID, isDel int, takeUsable int, abnormalReason *string) bool {
	if len(ids) == 0 {
		zap.S().Warn("BatchUpdateMongoLineDataByIds ids is empty")
		return false
	}
	ctx := context.Background()
	collection := global.MongoDB.Collection("my_inventory_pack")
	filter := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
		"is_del": bson.M{"$ne": 1},
	}
	setData := bson.M{
		"is_del":      isDel,
		"take_usable": takeUsable,
		"update_time": time.Now().UnixMilli(),
	}
	if abnormalReason != nil {
		setData["abnormal_reason"] = *abnormalReason
	}
	update := bson.M{
		"$set": setData,
	}
	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		zap.S().Errorf("BatchUpdateMongoLineDataByIds 更新失败 err=%v ids=%v", err, ids)
		return false
	}
	zap.S().Infof(
		"BatchUpdateMongoLineDataByIds success matched=%d modified=%d ids=%d",
		result.MatchedCount,
		result.ModifiedCount,
		len(ids),
	)
	return true
}

// 自开箱yym_un_case_record
func UpdateCs2CaseKey(packId, assetId string) bool {
	if err := global.DB.Model(&models.YYMUnCaseRecord{}).
		Where("pack_id = ? and asset_id = ?", packId, assetId).
		Update("item_status", "pack_transfer").Error; err != nil {
		zap.S().Errorf("❌ UpdateCs2CaseKey fail pack_id=%v asset_id=%v err=%v", packId, assetId, err)
		return false
	}
	zap.S().Infof("✅ UpdateCs2CaseKey success pack_id=%v asset_id=%v", packId, assetId)
	return true
}

// 武库yym_xp_shop_exchange_record
func UpdateCs2XpShopAccount(packId, assetId string) bool {
	if err := global.DB.Model(&models.YYMXpShopExchangeRecord{}).
		Where("pack_id = ? and asset_id = ?", packId, assetId).
		Update("item_status", "pack_transfer").Error; err != nil {
		zap.S().Errorf("❌ UpdateCs2XpShopAccount fail pack_id=%v asset_id=%v err=%v", packId, assetId, err)
		return false
	}
	zap.S().Infof("✅ UpdateCs2XpShopAccount success pack_id=%v asset_id=%v", packId, assetId)
	return true
}

// 汰换yym_cs2_tradeup_un_record
func UpdateCs2Tradeup(packId, assetId string) bool {
	if err := global.DB.Model(&models.YYMCs2TradeupUnRecord{}).
		Where("pack_id = ? and asset_id = ?", packId, assetId).
		Update("item_status", "pack_transfer").Error; err != nil {
		zap.S().Errorf("❌ UpdateCs2Tradeup fail pack_id=%v asset_id=%v err=%v", packId, assetId, err)
		return false
	}
	zap.S().Infof("✅ UpdateCs2Tradeup success pack_id=%v asset_id=%v", packId, assetId)
	return true
}

// 通过背包id查表中数据
func FindTradeTransferLogsByPackIDs(packIDs []string) ([]models.InventoryPackTradeTransferLog, error) {
	var list []models.InventoryPackTradeTransferLog
	if len(packIDs) == 0 {
		return list, nil
	}
	err := global.DB.Model(&models.InventoryPackTradeTransferLog{}).
		Where("pack_id IN ?", packIDs).
		Where("offer_status = ?", 2).
		Find(&list).Error
	return list, err
}
