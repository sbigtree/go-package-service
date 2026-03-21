package mypackage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/scheduler/util/upackage"
	steam_tools "github.com/sbigtree/steam-tools-grpc/go/generator"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"strconv"
)

func DealExpireData2(_event event.EventMsg) error {
	var params upackage.DealExpireDataParam
	err := json.Unmarshal(_event.Params, &params)
	if err != nil {
		zap.S().Error("事件 DealExpireData2 解析消息参数失败", zap.Error(err))
		return err
	}

	if len(params.Ids) == 0 || params.SteamAID == 0 || params.OfferID == "" {
		zap.S().Warnf(" DealExpireData2 Ids=%v SteamAID=%v OfferID=%v", len(params.Ids), params.SteamAID, params.OfferID)
		return nil
	}

	logId := fmt.Sprintf(" DealExpireData2 SteamAID=%v offerId=%v ", params.SteamAID, params.OfferID)
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
	objectIds := make([]primitive.ObjectID, 0, len(items))
	for _, item := range items {
		//背包id转成mongoid
		objID, err := primitive.ObjectIDFromHex(item.PackID)
		if err != nil {
			zap.S().Warnf("%s transfer packId to objID err packId=%v", logId, item.PackID)
			continue
		}
		if item.OfferID != params.OfferID {
			zap.S().Infof("%s  item.OfferID=%s not equal params.OfferID=%s ", logId, item.OfferID, params.OfferID)
			continue
		}
		req := &steam_tools.OfferAcceptOfferRequest{
			SteamAid:     item.ReceivedSteamAID,
			Tradeofferid: item.OfferID,
			SteamidOther: strconv.Itoa(int(item.SteamAID)),
		}
		res, err := global.SteamTools.OfferServerClient.OfferAcceptOffer(context.Background(), req)
		if err != nil || res == nil || !res.GetSuccess() || res.Data == nil {
			zap.S().Warnf("%s OfferAcceptOffer 操作失败: %s ", logId, res.GetMessage())
			continue
		}

		if res.Data.GetNeedsMobileConfirmation() == "" {
			//打一条日志当交易id是空的时候记一下
			zap.S().Warnf("%s get empty  GetNeedsMobileConfirmation val SteamidOther=%v", logId, item.SteamAID)
			continue
		}
		objectIds = append(objectIds, objID)
	}

	if len(objectIds) > 0 {
		//给通道写发送报价成功的数据
		msg := event.NewEventMsg(map[string]interface{}{
			"ids":       objectIds,
			"steam_aid": params.SteamAID,
			"offer_id":  params.OfferID, //交易id
		})
		global.ExpireDataChannel3 <- msg
	}

	return nil
}
