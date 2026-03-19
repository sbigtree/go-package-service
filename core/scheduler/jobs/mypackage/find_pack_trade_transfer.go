package mypackage

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

func FindPackTradeTransfer() {
	var (
		batchSize      = 500
		lastID    uint = 64
	)

	for {
		var items []models.InventoryPackTradeTransferLog
		query := global.DB.Model(&models.InventoryPackTradeTransferLog{}).Where("offer_status=?", 2).Order("id ASC").Limit(batchSize)

		if lastID > 0 {
			query = query.Where("id > ?", lastID)
		}

		if err := query.Find(&items).Error; err != nil {
			zap.S().Errorf("查询失败%v", err)
			return
		}

		if len(items) == 0 {
			break
		}

		uniqueOffers := make(map[string]int32)
		var results []OfferGroup
		for _, item := range items {
			if item.OfferID != "" {
				if _, exists := uniqueOffers[item.OfferID]; !exists {
					uniqueOffers[item.OfferID] = item.SteamAID
					results = append(results, OfferGroup{
						OfferID:          item.OfferID,
						ReceivedSteamAid: item.ReceivedSteamAID,
						PackID:           item.PackID,
						CategoryName:     item.CategoryName,
						AssetID:          item.AssetID,
					})
				}
			}
		}

		for _, result := range results {
			zap.S().Infof("处理交易:OfferID=%s，接收者Aid=%d", result.OfferID, result.ReceivedSteamAid)
			checkResult := CheckOfferTradeStatus(result)
			if checkResult {
				//修改表中数据
				updateResult := UpdateTradeTransferLogs(result.OfferID)
				if !updateResult {
					updateResult = UpdateTradeTransferLogs(result.OfferID)
					if !updateResult {
						continue
					}
				}
				//修改mongo表中数据
				UpdateMongoLineDataById(result.PackID)
				//修改表中数据
				switch result.CategoryName {
				case "cs2_case_key":
					//自开箱yym_un_case_record
					UpdateCs2CaseKey(result.PackID, result.AssetID)
				case "cs2_xp_shop_account":
					//武库yym_xp_shop_exchange_record
					UpdateCs2XpShopAccount(result.PackID, result.AssetID)
				case "cs2_tradeup":
					//汰换yym_cs2_tradeup_un_record
					UpdateCs2Tradeup(result.PackID, result.AssetID)
				}
			}
		}

		if len(items) < batchSize {
			break
		}
		lastID = items[len(items)-1].ID
	}

}

// 自开箱yym_un_case_record
func UpdateCs2CaseKey(packId, assetId string) bool {
	if err := global.DB.Model(&models.YYMUnCaseRecord{}).
		Where("pack_id = ? and asset_id = ?", packId, assetId).
		Update("item_status=?", 3).Error; err != nil {
		zap.S().Errorf("update table yym_un_case_record error pack_id=%v asset_id=%v", packId, assetId)
		return false
	}
	return true
}

// 武库yym_xp_shop_exchange_record
func UpdateCs2XpShopAccount(packId, assetId string) bool {
	if err := global.DB.Model(&models.YYMXpShopExchangeRecord{}).
		Where("pack_id = ? and asset_id = ?", packId, assetId).
		Update("item_status=?", 3).Error; err != nil {
		zap.S().Errorf("update table yym_xp_shop_exchange_record error pack_id=%v asset_id=%v", packId, assetId)
		return false
	}
	return true
}

// 汰换yym_cs2_tradeup_un_record
func UpdateCs2Tradeup(packId, assetId string) bool {
	if err := global.DB.Model(&models.YYMCs2TradeupUnRecord{}).
		Where("pack_id = ? and asset_id = ?", packId, assetId).
		Update("item_status=?", 3).Error; err != nil {
		zap.S().Errorf("update table yym_cs2_tradeup_un_record error pack_id=%v asset_id=%v", packId, assetId)
		return false
	}
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
	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"is_del":      1,
			"update_time": time.Now().UnixMilli(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		zap.S().Errorf("更新失败: %v %v", err, id)
		return false
	}
	zap.S().Infof("成功修改记录，匹配数量: %d, 修改数量: %d", result.MatchedCount, result.ModifiedCount)
	return true
}

func CheckOfferTradeStatus(result OfferGroup) bool {
	tempReceivedSteamAid := result.ReceivedSteamAid
	tempTradeofferid := result.OfferID
	tempAppId := int32(730)
	getOfferDetailResponse, err := global.SteamTools.OfferServerClient.GetOfferDetail(context.Background(), &steam_tools_grpc.OfferGetOfferDetailRequest{
		SteamAid:     &result.ReceivedSteamAid,
		Tradeofferid: &result.OfferID,
		Appid:        &tempAppId,
	})

	if err != nil || getOfferDetailResponse == nil {
		zap.S().Errorf("调用steam_tools服务失败 response=%v steamAid=%v tempTradeofferid=%v err=%v", getOfferDetailResponse, tempReceivedSteamAid, tempTradeofferid, err)
		return false
	}

	if getOfferDetailResponse.Success != nil && *getOfferDetailResponse.Success == false {
		zap.S().Errorf("调用steam_tools服务失败 success value,  response=%v steamAid=%v tempTradeofferid=%v", *getOfferDetailResponse.Success, tempReceivedSteamAid, tempTradeofferid)
		return false
	}

	if getOfferDetailResponse.Data == nil || getOfferDetailResponse.Data.TradeOfferState == nil {
		zap.S().Errorf("调用steam_tools服务失败 data value,  data=%v steamAid=%v tempTradeofferid=%v tradeOfferState=%v",
			getOfferDetailResponse.Data, tempReceivedSteamAid, tempTradeofferid, getOfferDetailResponse.Data.TradeOfferState)
		return false
	}

	if *getOfferDetailResponse.Data.TradeOfferState != 3 {
		zap.S().Infof("调用steam_tools服务成功 未交易成功 TradeOfferState value,  steamAid=%v tempTradeofferid=%v tradeOfferState=%v",
			tempReceivedSteamAid, tempTradeofferid, *getOfferDetailResponse.Data.TradeOfferState)
		return false
	}
	zap.S().Infof("调用steam_tools服务成功 交易成功 TradeOfferState value,  steamAid=%v tempTradeofferid=%v tradeOfferState=%v",
		tempReceivedSteamAid, tempTradeofferid, *getOfferDetailResponse.Data.TradeOfferState)
	return true
}

func UpdateTradeTransferLogs(offerId string) bool {
	if err := global.DB.Model(&models.InventoryPackTradeTransferLog{}).
		Where("offer_id = ? and offer_status = 2", offerId).
		Update("offer_status=?", 3).Error; err != nil {
		zap.S().Errorf("update table inventory_pack_trade_transfer_logs error offerId=%v", offerId)
		return false
	}
	return true
}
