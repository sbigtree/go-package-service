package mypackage

import (
	"encoding/json"
	"fmt"
	mongo2 "github.com/sbigtree/go-db-model/v2/mongo/models"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/scheduler/util/upackage"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func DealExpireData3(_event event.EventMsg) error {
	var params upackage.DealExpireDataParam
	err := json.Unmarshal(_event.Params, &params)
	if err != nil {
		zap.S().Errorf("事件 DealExpireData3 解析消息参数失败 %v", err)
	}

	if len(params.Ids) == 0 || params.SteamAID == 0 || params.OfferID == "" {
		zap.S().Warnf(" DealExpireData3 Ids=%v SteamAID=%v OfferID=%v", len(params.Ids), params.SteamAID, params.OfferID)
		return nil
	}

	logId := fmt.Sprintf(" DealExpireData3 SteamAID=%v offerId=%v ", params.SteamAID, params.OfferID)
	zap.S().Infof("%s", logId)

	packIds := make([]string, 0, len(params.Ids))
	for _, id := range params.Ids {
		packId := id.Hex()
		packIds = append(packIds, packId)
	}

	if len(packIds) == 0 {
		zap.S().Warnf("%s empty packIds", logId)
		return nil
	}

	items, err := upackage.FindTradeTransferLogsByPackIDs(packIds)
	if err != nil {
		zap.S().Warnf("%s FindTradeTransferLogsByPackIDs error", logId)
		return err
	}

	successPackIds := make([]string, 0, len(items))
	executeObjectIds := make([]primitive.ObjectID, 0, len(items))
	for _, item := range items {
		if item.OfferID != params.OfferID {
			zap.S().Warnf("%s item.OfferID=%v not equal params.OfferID=%v ", logId, item.OfferID, params.OfferID)
			continue
		}
		objID, err := primitive.ObjectIDFromHex(item.PackID)
		if err != nil {
			zap.S().Warnf("%s transfer packId to objID err packId=%v", logId, item.PackID)
			continue
		}
		offerTradeParam := upackage.OfferGroup{
			OfferID:          item.OfferID,
			ReceivedSteamAid: item.ReceivedSteamAID,
			PackID:           item.PackID,
			CategoryName:     item.CategoryName,
			AssetID:          item.AssetID,
		}
		checkResult := upackage.CheckOfferTradeStatus(offerTradeParam)
		if !checkResult {
			//主动做一次交易查询，核查是否交易成功
			zap.S().Warnf("%s CheckOfferTradeStatus fail offerTradeParam=%v", logId, offerTradeParam)
			continue
		}

		switch item.CategoryName {
		case "cs2_case_key":
			//自开箱yym_un_case_record
			ret := upackage.UpdateCs2CaseKey(item.PackID, item.AssetID)
			if !ret {
				continue
			}
		case "cs2_xp_shop_account":
			//武库yym_xp_shop_exchange_record
			ret := upackage.UpdateCs2XpShopAccount(item.PackID, item.AssetID)
			if !ret {
				continue
			}
		case "cs2_tradeup":
			//汰换yym_cs2_tradeup_un_record
			ret := upackage.UpdateCs2Tradeup(item.PackID, item.AssetID)
			if !ret {
				continue
			}
		}
		successPackIds = append(successPackIds, item.PackID)
		executeObjectIds = append(executeObjectIds, objID)
	}

	if len(successPackIds) == 0 {
		//表示上面的修改都失败了 打印一条日志
		zap.S().Warnf("%s empty executeSuccessIds", logId)
		return nil
	}

	//3表示确认报价成功
	//批量修改表inventory_pack_trade_transfer_logs数据
	batchLogets := upackage.UpdateTradeTransferLogsBatch(successPackIds, params.OfferID, 3)
	if !batchLogets {
		//不成功再来一次兜底修改
		zap.S().Warnf("%s UpdateTradeTransferLogsBatch error", logId)
		upackage.UpdateTradeTransferLogsBatch(successPackIds, params.OfferID, 3)
	}

	//修改mongo表中数据
	abnormalReason := mongo2.MyInventoryPack_AbnormalReason.Stash
	//加修改mongo表数据的逻辑
	upRet := upackage.BatchUpdateMongoLineDataByIds(executeObjectIds, 0, int(mongo2.MyInventoryPack_TakeUsable_Status.Extract), &abnormalReason)
	if !upRet {
		//不成功再来一次兜底修改
		zap.S().Warnf("%s BatchUpdateMongoLineDataByIds error", logId)
		upackage.BatchUpdateMongoLineDataByIds(executeObjectIds, 0, int(mongo2.MyInventoryPack_TakeUsable_Status.Extract), &abnormalReason)
	}

	//给my_inventory_pack_records插入数据 暂时先不插入数据@南哥要求的

	return nil
}
