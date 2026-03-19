package mypackage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sbigtree/go-db-model/v2/models"
	mongo2 "github.com/sbigtree/go-db-model/v2/mongo/models"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/scheduler/util/upackage"
	"github.com/sbigtree/go-package-service/core/tools"
	steam_tools "github.com/sbigtree/go-package-service/core/tools/steam"
	steam_tools_grpc "github.com/sbigtree/steam-tools-grpc/go/generator"
	steam_tools_tools "github.com/sbigtree/steam-tools-grpc/go/tools"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type TradeTask struct {
	SteamAid       int32
	TradeUrl       string
	UserID         int64
	AssetID        string
	MarketName     string
	MarketHashName string
	Img            string
	AppID          string
	UintAppID      uint32
	Amount         string
	PackID         string
	CategoryName   string
}

type PartSteamAccount struct {
	Steamid  string
	TradeURL string
}

func checkGameSellerStatus(logId string, tempSteamAid int) bool {
	// 查询 SteamAccount
	var steam models.SteamAccount
	err := global.DB.Where("id = ?", tempSteamAid).Take(&steam).Error
	if err != nil {
		zap.S().Warnf("%s 查询 SteamAccount 失败 steamAid=%v err=%v", logId, tempSteamAid, err)
		return false
	}

	// 检查状态
	if (steam.CommunityBanned != nil && *steam.CommunityBanned == 1) ||
		(steam.VacBanned != nil && *steam.VacBanned == 1) ||
		(steam.LoginEresult != nil && *steam.LoginEresult == 5) ||
		steam.Guard == "" {
		zap.S().Warnf("%s SteamAccount 状态不可用 steamAid=%v", logId, tempSteamAid)
		return false
	}

	// 解密 Guard
	jsonData, err := tools.DecryptAES(steam.Guard, []byte(global.AppConfigMaster.AESCRYPTKEY), "hex")
	if err != nil {
		zap.S().Warnf("%s 解密 Guard 失败 steamAid=%v", logId, tempSteamAid)
		return false
	}
	var steamAuthData steam_tools.SteamAuthInfo
	if err := json.Unmarshal([]byte(jsonData), &steamAuthData); err != nil || steamAuthData.IdentitySecret == "" {
		zap.S().Warnf("%s SteamAuthInfo 解析失败 steamAid=%v", logId, tempSteamAid)
		return false
	}
	return true
}

func prepareTradeTasksBatch(
	ids []primitive.ObjectID,
	realTimeAssetIDMap map[string]bool,
	logId string,
) ([]*TradeTask, []primitive.ObjectID, error) {

	start := time.Now()
	collection := global.MongoDB.Collection("my_inventory_pack")

	// 1️⃣ 批量查询 MongoDB
	filter := bson.M{"_id": bson.M{"$in": ids}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		zap.S().Errorf("%s Mongo 批量查询失败 err=%v", logId, err)
		return nil, nil, err
	}

	var results []mongo2.MyInventoryPack
	if err := cursor.All(ctx, &results); err != nil {
		zap.S().Errorf("%s Mongo 批量查询解析失败 err=%v", logId, err)
		return nil, nil, err
	}
	zap.S().Infof("%s Mongo 批量查询完成, 查询到 %d 条数据, 耗时=%v", logId, len(results), time.Since(start))

	// 2️⃣ 建立结果 map
	packMap := make(map[primitive.ObjectID]mongo2.MyInventoryPack, len(results))
	for _, p := range results {
		packMap[p.ID] = p
	}

	// 3️⃣ 遍历 ids 构建 TradeTask
	tradeTaskMap := make([]*TradeTask, 0, len(ids))
	objectIds := make([]primitive.ObjectID, 0, len(ids))
	now := time.Now().Unix()
	for _, id := range ids {
		result, ok := packMap[id]
		if !ok {
			zap.S().Warnf("%s Mongo 中不存在该数据，跳过 id=%v", logId, id.Hex())
			continue
		}

		var assetId, appId, marketHashName, categoryName string
		for _, o := range result.Origins {
			switch o.Type {
			//case "steam_aid":
			//	steamAid = fmt.Sprintf("%v", o.Value)
			case "asset_id":
				assetId = o.Value
			case "appid":
				appId = o.Value
			case "market_hash_name":
				marketHashName = o.Value
			case "from_type":
				categoryName = o.Value
			}
		}

		if _, ok := realTimeAssetIDMap[assetId]; !ok {
			zap.S().Warnf("%s 物品:%s不在steam实时库存中，跳过", logId, result.Name)
			continue
		}

		appIdInt, err := strconv.Atoi(appId)
		if err != nil {
			zap.S().Warnf("%s appId 转换失败 err %v", logId, err)
			continue
		}

		// 检查 tradable_time 标签
		skip := false
		for _, tag := range result.Tags {
			if tag.Type == "tradable_time" {
				tradeableTimeInt, err := strconv.ParseInt(tag.Value, 10, 64)
				if err != nil || now < tradeableTimeInt {
					skip = true
					break
				}
			}
		}
		if skip {
			continue
		}

		task := &TradeTask{
			UserID:         result.UserId,
			AssetID:        assetId,
			MarketName:     result.Name,
			MarketHashName: marketHashName,
			Img:            result.ImageUrl,
			AppID:          appId,
			UintAppID:      uint32(appIdInt),
			Amount:         "1",
			PackID:         id.Hex(),
			CategoryName:   categoryName,
		}
		objectIds = append(objectIds, id)
		tradeTaskMap = append(tradeTaskMap, task)
		zap.S().Infof("%s prepareTradeTask 构建完成 id=%v", logId, id.Hex())
	}

	return tradeTaskMap, objectIds, nil
}
func getRealTimeInventory(tempSteamAid int32, logId string) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resp, err := global.SteamTools.InventoryCsgoServerClient.GetInventoryCsgo(ctx, &steam_tools_grpc.InventoryCsgoRequest{
		SteamAid: tempSteamAid,
	})
	if err != nil {
		zap.S().Errorf("%s %d 调用 SteamTools 获取库存失败  %v", logId, tempSteamAid, err)
		return nil, err
	}
	if !resp.Success {
		zap.S().Warnf("%s %d 调用 SteamTools 获取库存失败2  %v", logId, tempSteamAid, resp.Message)
		return nil, errors.New("调用 SteamTools 获取库存失败2")
	}
	// 解析实时库存
	realTimeInventory := steam_tools_tools.ParseCsgoInventory(resp.Data)
	zap.S().Infof("%s %d 获取到实时库存 %d 件", logId, tempSteamAid, len(realTimeInventory))

	//拿到实时库存之后进行对比
	realTimeAssetIDMap := map[string]bool{}
	for _, item := range realTimeInventory {
		realTimeAssetIDMap[*item.Assets.Assetid] = true
	}
	return realTimeAssetIDMap, nil
}

func reGetTradeUrl(tradeUrl string, steamaid int32) (string, error) {
	zap.S().Warnf("该steam账号未设置交易链接 %v", tradeUrl)
	//调用steam_tools服务获取交易链接
	// 同步账号信息 先同步账号信息再查询数据库
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	SteamProfileInfo, err := global.SteamTools.AccountServerClient.GetProfile(ctx, &steam_tools_grpc.AccountGetProfileRequest{
		SteamAid: steamaid,
	})
	if err != nil {
		zap.S().Errorf("发送报价调用steam_tools服务获取交易链接失败 %v", err)
		return "", errors.New("发送报价调用steam_tools服务获取交易链接失败")
	}
	if !SteamProfileInfo.Success {
		zap.S().Warnf("%d 发送报价检查账号信息失败", steamaid)
		return "", errors.New("get profile failed")
	}
	if SteamProfileInfo.Data.TradeUrl != nil && *SteamProfileInfo.Data.TradeUrl != "" {
		tradeUrl = *SteamProfileInfo.Data.TradeUrl
		zap.S().Warnf("new该steam账号未设置交易链接 %s", tradeUrl)
	} else {
		// 如果还是没有 则提示用户去设置交易链接
		zap.S().Warnf("%d 该steam账号未设置交易链接", steamaid)
		return "", errors.New("trade url empty")
	}
	return tradeUrl, nil
}

func getTradeCount(tempSteamAid int32, logId string) bool {
	//查看用户当下是否可以发起交易  一个用户最多发起5次交易
	offerListReq := &steam_tools_grpc.OfferGetOfferListRequest{
		SteamAid:   tempSteamAid,
		ActiveOnly: int32(1),
	}

	// 注意：请确认你的 global.SteamTools 中是否有 OfferServerClient 实例
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	offerResp, err := global.SteamTools.OfferServerClient.GetOfferList(ctx, offerListReq)
	if err != nil {
		zap.S().Warnf("%s %d 调用Node gRPC服务[GetOfferList]失败! err %+v ", logId, tempSteamAid, err)
		return false
	}

	// 3. 校验 Node 返回的业务逻辑状态
	if offerResp == nil || offerResp.Data == nil || !offerResp.GetSuccess() {
		zap.S().Warnf("%s %d 调用Node gRPC服务[GetOfferList]失败! get nil val err %+v ", logId, tempSteamAid, err)
		return false
	}

	if len(offerResp.Data.TradeOffersSent) >= 5 {
		zap.S().Infof("%s %d beyond SendOffer limit %d", logId, tempSteamAid, tempSteamAid)
		return false
	}

	return true
}

func DealExpireData(_event event.EventMsg) error {
	var params upackage.DealExpireDataParam
	err := json.Unmarshal(_event.Params, &params)
	if err != nil {
		zap.S().Error("事件 FinalDispose 解析消息参数失败", err)
		return err
	}

	logId := fmt.Sprintf(" DealExpireData SteamAID=%v ", params.SteamAID)

	//检查卖家用户状态
	sellerStatus := checkGameSellerStatus(logId, int(params.SteamAID))
	if !sellerStatus {
		zap.S().Errorf(" checkGameSellerStatus the  SteamAID forbid %+v", params.SteamAID)
		return nil
	}

	zap.S().Infof("%s", logId)
	if !getTradeCount(params.SteamAID, logId) {
		zap.S().Errorf(" getTradeCount data err %+v", params.SteamAID)
		return nil
	}

	//前置查询
	realTimeAssetIDMap, err := getRealTimeInventory(params.SteamAID, logId)
	if err != nil {
		zap.S().Errorf("%+v getRealTimeInventory data err %+v", params.SteamAID, err)
		return err
	}
	zap.S().Infof("logID = %s will execute logid count=%v", logId, len(params.Ids))
	tradeTaskMap, objectIds, err := prepareTradeTasksBatch(params.Ids, realTimeAssetIDMap, logId)
	if err != nil {
		zap.S().Errorf("%s prepareTradeTasksBatch error %+v", logId, err)
		return err
	}
	// 转成切片和 objectIds
	tasks := make([]TradeTask, 0, len(tradeTaskMap))

	for _, task := range tradeTaskMap {
		if task != nil {
			tasks = append(tasks, *task)
		}
	}

	zap.S().Infof("%s current task num %d", logId, len(tasks))
	if len(tasks) == 0 {
		zap.S().Warnf("%s 没有可执行的任务 steamAid=%d", logId, params.SteamAID)
		return nil
	}
	zap.S().Infof("%s current task num %d", logId, len(tasks))

	myItem := make([]*steam_tools_grpc.OfferSendOfferRequest_TradeItem, 0, len(tasks))

	for _, task := range tasks {
		itemData := steam_tools_grpc.OfferSendOfferRequest_TradeItem{
			Amount:  task.Amount,
			Assetid: task.AssetID,
			Game: &steam_tools_grpc.OfferSendOfferRequest_Game{
				Appid:     task.UintAppID,
				ContextId: 2,
			},
		}
		myItem = append(myItem, &itemData)
	}

	zap.S().Infof("%s 当前可执行的任务信息 steamAid=%d tasks=%d", logId, params.SteamAID, len(tasks))

	//查表user_groups中数据
	var userGroups []models.UserGroup
	if err := global.DB.Model(&models.UserGroup{}).Find(&userGroups).Error; err != nil {
		zap.S().Errorf("%s select user_groups table error %v", logId, err)
		return nil
	}
	zap.S().Infof("%s 当前usergroup info steamAid=%d userGroups=%d", logId, params.SteamAID, len(userGroups))

	groupIds := make([]int32, 0, len(userGroups))
	for _, g := range userGroups {
		groupIds = append(groupIds, g.GroupID)
	}

	zap.S().Infof("%s 当前groupIds info steamAid=%d groupIds=%d", logId, params.SteamAID, len(groupIds))
	if len(groupIds) == 0 {
		zap.S().Errorf("%s get empty group id", logId)
		return nil
	}

	var tempSteamAccounts []models.SteamAccount
	if err := global.DB.Model(&models.SteamAccount{}).Where("group_id in ?", groupIds).Find(&tempSteamAccounts).Error; err != nil {
		zap.S().Errorf("%s check user_groups table error %v %v", logId, err, groupIds)
		return nil
	}
	zap.S().Infof("%s tempSteamAccounts info steamAid=%d tempSteamAccounts=%d", logId, params.SteamAID, len(tempSteamAccounts))
	if len(tempSteamAccounts) == 0 {
		zap.S().Errorf("%s get empty transfer result groupIds=%v", logId, groupIds)
		return nil
	}

	ttSteamAccount := make(map[string]struct{}, len(tempSteamAccounts))
	executeSteamAccount := make([]PartSteamAccount, 0, len(tempSteamAccounts))
	for _, tempSteamAccount := range tempSteamAccounts {
		if tempSteamAccount.Steamid == "" {
			continue
		}
		if _, ok := ttSteamAccount[tempSteamAccount.Steamid]; !ok {
			executeSteamAccount = append(executeSteamAccount, PartSteamAccount{
				Steamid:  tempSteamAccount.Steamid,
				TradeURL: tempSteamAccount.TradeURL,
			})
			ttSteamAccount[tempSteamAccount.Steamid] = struct{}{}
		}
	}

	if len(executeSteamAccount) == 0 {
		zap.S().Errorf("%s executeSteamAccount empty", logId)
		return nil
	}

	transferSteamId, err := strconv.Atoi(executeSteamAccount[0].Steamid)
	if err != nil {
		zap.S().Infof("%s executeSteamAccount TradeURL string to int error Steamid=%v err=%v", logId, executeSteamAccount[0].Steamid, err)
		return err
	}

	if executeSteamAccount[0].TradeURL == "" {
		executeSteamAccount[0].TradeURL, err = reGetTradeUrl(executeSteamAccount[0].TradeURL, int32(transferSteamId))
		if err != nil {
			zap.S().Infof("SteamAid %s TradeUrl %s empty val", executeSteamAccount[0].Steamid, executeSteamAccount[0].TradeURL)
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	response, err := global.SteamTools.OfferServerClient.SendOffer(ctx, &steam_tools_grpc.OfferSendOfferRequest{
		SteamAid:    int32(transferSteamId),
		TradeUrl:    executeSteamAccount[0].TradeURL,
		MyItems:     myItem,
		ThemItems:   []*steam_tools_grpc.OfferSendOfferRequest_TradeItem{},
		AutoConfirm: 1,
	})

	if err != nil {
		zap.S().Errorf("%s 调用steam_tools服务失败 response=%+v steamAid=%d err=%v",
			logId,
			response,
			transferSteamId,
			err,
		)
		return err
	}
	if response == nil || response.Data == nil || response.Data.Tradeofferid == nil || *response.Data.Tradeofferid == "" {
		zap.S().Errorf("%s send offer response invalid steamAid=%d", logId, transferSteamId)
		return errors.New("send offer response invalid")
	}
	var logs []models.InventoryPackTradeTransferLog
	for _, task := range tasks {
		logs = append(logs, models.InventoryPackTradeTransferLog{
			UserID:           task.UserID,
			OfferID:          *response.Data.Tradeofferid,
			OfferStatus:      2,
			AssetID:          task.AssetID,
			MarketName:       task.MarketName,
			MarketHashName:   task.MarketHashName,
			Img:              task.Img,
			AppID:            task.AppID,
			SteamAID:         params.SteamAID,
			ReceivedSteamAID: int32(transferSteamId),
			ReceivedTradeURL: executeSteamAccount[0].TradeURL,
			PackID:           task.PackID,
			CategoryName:     task.CategoryName,
		})
	}

	//数据落库
	if len(logs) > 0 {
		if err := global.DB.Create(&logs).Error; err != nil {
			zap.S().Errorf("%s 批量插入 inventory_pack_trade_transfer_log 失败 %v", logId, err)
		} else {
			zap.S().Infof("%s 批量插入成功 %d 条", logId, len(logs))
		}
	}

	if len(objectIds) > 0 {
		//写通道数据
		msg := event.NewEventMsg(map[string]interface{}{
			"ids":       objectIds,
			"steam_aid": int32(transferSteamId),
			"offer_id":  *response.Data.Tradeofferid, //交易id
		})
		global.CheckSendOfferChannel <- msg
	}

	return nil
}
