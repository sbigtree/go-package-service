package mypackage

import (
	"github.com/sbigtree/go-db-model/v2/models"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/scheduler/util/upackage"
	"go.uber.org/zap"
)

func FindPackTradeTransfer() {
	var (
		batchSize int  = 500
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
		var results []upackage.OfferGroup
		for _, item := range items {
			if item.OfferID != "" {
				if _, exists := uniqueOffers[item.OfferID]; !exists {
					uniqueOffers[item.OfferID] = item.SteamAID
					results = append(results, upackage.OfferGroup{
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
			zap.S().Infof("%s FindPackTradeTransfer 处理交易:OfferID=%s，接收者Aid=%d", result.ReceivedSteamAid, result.OfferID, result.ReceivedSteamAid)
			checkResult := upackage.CheckOfferTradeStatus(result)
			if checkResult {
				//修改表中数据
				updateResult := upackage.UpdateTradeTransferLogs(result.OfferID)
				if !updateResult {
					updateResult = upackage.UpdateTradeTransferLogs(result.OfferID)
					if !updateResult {
						continue
					}
				}
				//修改mongo表中数据
				upackage.UpdateMongoLineDataById(result.PackID)
				//修改表中数据
				switch result.CategoryName {
				case "cs2_case_key":
					//自开箱yym_un_case_record
					upackage.UpdateCs2CaseKey(result.PackID, result.AssetID)
				case "cs2_xp_shop_account":
					//武库yym_xp_shop_exchange_record
					upackage.UpdateCs2XpShopAccount(result.PackID, result.AssetID)
				case "cs2_tradeup":
					//汰换yym_cs2_tradeup_un_record
					upackage.UpdateCs2Tradeup(result.PackID, result.AssetID)
				}
			}
		}

		if len(items) < batchSize {
			break
		}
		lastID = items[len(items)-1].ID
	}

}
