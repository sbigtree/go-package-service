package mypackage

import (
	"github.com/sbigtree/go-db-model/v2/models"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

func FindExpireData3() {
	var (
		batchSize int  = 500
		lastID    uint = 0
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

		steamOfferDatas := make(map[string][]primitive.ObjectID, len(items))
		for _, item := range items {
			if item.OfferID != "" {
				zap.S().Warnf(" FindExpireData2 empty OfferID item.PackID=%v", item.PackID)
				continue
			}
			objID, err := primitive.ObjectIDFromHex(item.PackID)
			if err != nil {
				zap.S().Warnf(" FindExpireData2 transfer packId to objID err packId=%v", item.PackID)
				continue
			}
			SteamAIDStr := strconv.Itoa(int(item.ReceivedSteamAID))
			tempKey := SteamAIDStr + "-" + item.OfferID

			steamOfferDatas[tempKey] = append(steamOfferDatas[tempKey], objID)
		}

		for key, val := range steamOfferDatas {
			if len(val) == 0 {
				zap.S().Infof(" FindExpireData2 empty val %v", key)
				continue
			}
			parts := strings.Split(key, "-")
			if len(parts) != 2 {
				zap.S().Warnf(" FindExpireData2 invalid tempKey format: %s", key)
				continue
			}
			tempSteamAid, err := strconv.Atoi(parts[0])
			if err != nil {
				zap.S().Warnf(" FindExpireData2 parts[0] string to int error: %v", parts)
				continue
			}
			msg := event.NewEventMsg(map[string]interface{}{
				"ids":       val,
				"steam_aid": int32(tempSteamAid),
				"offer_id":  parts[1], //交易id
			})
			global.ExpireDataChannel3 <- msg
		}

		if len(items) < batchSize {
			break
		}
		lastID = items[len(items)-1].ID
	}
}
