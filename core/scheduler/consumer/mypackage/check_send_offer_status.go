package mypackage

import (
	"encoding/json"
	"github.com/sbigtree/go-db-model/v2/models"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	jobs "github.com/sbigtree/go-package-service/core/scheduler/jobs/mypackage"
	"go.uber.org/zap"
)

func CheckSendOfferStatus(_event event.EventMsg) error {
	var params DealExpireDataParam
	err := json.Unmarshal(_event.Params, &params)
	if err != nil {
		zap.S().Error("事件 FinalDispose 解析消息参数失败", zap.Error(err))
		return err
	}

	if len(params.Ids) == 0 {
		return nil
	}

	packIds := make([]string, 0, len(params.Ids))
	for _, id := range params.Ids {
		packId := id.Hex()
		packIds = append(packIds, packId)
	}

	if len(packIds) == 0 {
		return nil
	}

	items, err := FindTradeTransferLogsByOfferIDs(packIds)
	if err != nil {
		return err
	}

	uniqueOffers := make(map[string]int32)
	var results []jobs.OfferGroup
	for _, item := range items {
		if item.OfferID != "" {
			if _, exists := uniqueOffers[item.OfferID]; !exists {
				uniqueOffers[item.OfferID] = item.SteamAID
				results = append(results, jobs.OfferGroup{
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
		checkResult := jobs.CheckOfferTradeStatus(result)
		if checkResult {
			//修改表中数据
			updateResult := jobs.UpdateTradeTransferLogs(result.OfferID)
			if !updateResult {
				updateResult = jobs.UpdateTradeTransferLogs(result.OfferID)
				if !updateResult {
					continue
				}
			}
			//修改mongo表中数据
			jobs.UpdateMongoLineDataById(result.PackID)
			//修改表中数据
			switch result.CategoryName {
			case "cs2_case_key":
				//自开箱yym_un_case_record
				jobs.UpdateCs2CaseKey(result.PackID, result.AssetID)
			case "cs2_xp_shop_account":
				//武库yym_xp_shop_exchange_record
				jobs.UpdateCs2XpShopAccount(result.PackID, result.AssetID)
			case "cs2_tradeup":
				//汰换yym_cs2_tradeup_un_record
				jobs.UpdateCs2Tradeup(result.PackID, result.AssetID)
			}
		}
	}

	return nil
}

func FindTradeTransferLogsByOfferIDs(offerIDs []string) ([]models.InventoryPackTradeTransferLog, error) {
	var list []models.InventoryPackTradeTransferLog
	if len(offerIDs) == 0 {
		return list, nil
	}
	err := global.DB.Model(&models.InventoryPackTradeTransferLog{}).
		Where("offer_id IN ?", offerIDs).
		Where("offer_status = ?", 2).
		Find(&list).Error
	return list, err
}
