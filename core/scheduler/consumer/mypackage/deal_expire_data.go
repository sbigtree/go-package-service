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
	"github.com/sbigtree/go-package-service/core/tools"
	steam_tools "github.com/sbigtree/go-package-service/core/tools/steam"
	steam_tools_grpc "github.com/sbigtree/steam-tools-grpc/go/generator"
	steam_tools_tools "github.com/sbigtree/steam-tools-grpc/go/tools"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type DealExpireDataParam struct {
	Ids      []primitive.ObjectID `json:"ids"` // mongodb表主键id
	SteamAID int32                `json:"steam_aid"`
	OfferID  string               `json:"offer_id"`
}

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

func prepareTradeTask(ctx context.Context, id primitive.ObjectID, realTimeAssetIDMap map[string]bool) (*TradeTask, string, error) {

	logID := fmt.Sprintf("DealExpireData id:%s", id.Hex())
	start := time.Now()
	collection := global.MongoDB.Collection("my_inventory_pack")
	filter := bson.M{"_id": id}
	var result mongo2.MyInventoryPack
	if err := collection.FindOne(ctx, filter).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			zap.S().Warnf("Mongo 中不存在该数据，跳过 id=%v", id.Hex())
			return nil, logID, err
		}
		return nil, logID, err
	}

	var steamAid, assetId, appId, marketHashName, categoryName string
	for _, o := range result.Origins {
		switch o.Type {
		case "steam_aid":
			steamAid = fmt.Sprintf("%v", o.Value)
		case "asset_id":
			assetId = fmt.Sprintf("%v", o.Value)
		case "appid":
			appId = fmt.Sprintf("%v", o.Value)
		case "market_hash_name":
			marketHashName = fmt.Sprintf("%v", o.Value)
		case "from_type":
			categoryName = fmt.Sprintf("%v", o.Value)
		}
	}

	if _, ok := realTimeAssetIDMap[assetId]; !ok {
		zap.S().Warnf("%s 物品:%s不在steam实时库存当中，请检查后再操作", logID, result.Name)
		return nil, logID, errors.New("assetId not exist")
	}

	appIdInt, err := strconv.Atoi(appId)
	if err != nil {
		zap.S().Warnf("%s appId 转换失败 err %v", logID, err)
		return nil, logID, err
	}

	tempSteamAid, err := strconv.Atoi(steamAid)
	if err != nil {
		zap.S().Warnf("%s steamId 转换失败 err %+v ", logID, steamAid)
		return nil, logID, err
	}

	//第4步查询表中数据 判断账号信息是否正常
	var steam models.SteamAccount
	err = global.DB.Where("id = ?", tempSteamAid).First(&steam).Error
	if err != nil {
		zap.S().Warnf("%s === 库存风控 %s 查询steam失败 ", logID, steamAid)
		return nil, logID, err
	}

	// 检查状态是否可用
	if *steam.CommunityBanned == 1 {
		zap.S().Errorf("%s 账号社区封禁，无法发起报价", logID)
		return nil, logID, err
	}

	if steam.LoginEresult != nil && *steam.LoginEresult == 5 {
		zap.S().Errorf("%s 卖家密码错误，添加用户黑名单失败", logID)
		return nil, logID, err
	}

	//如果 VAC 封禁了 直接修改可用状态为不可用
	if *steam.VacBanned == 1 {
		zap.S().Errorf("%s 账号VAC封禁，无法发起报价", logID)
		return nil, logID, err
	}

	if steam.Guard == "" {
		zap.S().Warnf("%s 卖家账号令牌不存在，无法发起报价", logID)
		return nil, logID, err
	}

	jsonData, err := tools.DecryptAES(steam.Guard, []byte(global.AppConfigMaster.AESCRYPTKEY), "hex")
	if err != nil {
		zap.S().Warnf("%s 解析guard得到一个json结构体失败", logID)
		return nil, logID, err
	}

	var steamAuthData steam_tools.SteamAuthInfo
	err = json.Unmarshal([]byte(jsonData), &steamAuthData)
	if err != nil {
		zap.S().Warnf("%s %+v 序列化失败的令牌内容", logID, jsonData)
		zap.S().Errorf("解析 SteamAuthInfo 失败 err %+v %s", err, logID)
		return nil, logID, err
	}

	if steamAuthData.IdentitySecret == "" {
		zap.S().Warnf("%s 卖家账号令牌内容不完整，无法发起报价", logID)
		return nil, logID, err
	}

	for _, tag := range result.Tags {
		if tag.Type == "tradable_time" {
			tradeableTimeInt, err := strconv.ParseInt(tag.Value, 10, 64)
			if err != nil {
				zap.S().Errorf("%s 解析可交易时间失败 %v", logID, zap.Error(err))
				return nil, logID, err
			}
			currentTime := time.Now().Unix()
			if currentTime < tradeableTimeInt {
				zap.S().Warnf("%s 物品: %s 存在冷却中的物品，请检查后再操作 %s", result.Name, logID, time.Unix(tradeableTimeInt, 0).Format("2006-01-02 15:04:05"))
				return nil, logID, err
			}
		}
	}

	tradeUrl := steam.TradeURL //可用账号的url接收方
	zap.S().Infof("tradeUrl %s", tradeUrl)

	task := &TradeTask{
		SteamAid:       int32(tempSteamAid),
		TradeUrl:       tradeUrl,
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
	zap.S().Infof("[prepareTradeTask] 查询到数据 id=%v  耗时=%v", id.Hex(), time.Since(start))
	return task, logID, nil
}

func getRealTimeInventory(tempSteamAid int32) (map[string]bool, error) {
	resp, err := global.SteamTools.InventoryCsgoServerClient.GetInventoryCsgo(context.Background(), &steam_tools_grpc.InventoryCsgoRequest{
		SteamAid: tempSteamAid,
	})
	if err != nil {
		zap.S().Errorf("%d 调用 SteamTools 获取库存失败  %v", tempSteamAid, zap.Error(err))
		return nil, err
	}
	if !resp.Success {
		zap.S().Warnf("%d 调用 SteamTools 获取库存失败2  %v", tempSteamAid, resp.Message)
		return nil, err
	}
	// 解析实时库存
	realTimeInventory := steam_tools_tools.ParseCsgoInventory(resp.Data)
	zap.S().Infof("%d 获取到实时库存 %d 件", tempSteamAid, len(realTimeInventory))
	realTimeInventoryMarshal, err := json.Marshal(realTimeInventory)
	if err != nil {
		zap.S().Errorf("%d %+v序列化实时库存失败", tempSteamAid, zap.Error(err))
		return nil, err
	}
	zap.S().Infof("%d %s 获取到实时库存", tempSteamAid, string(realTimeInventoryMarshal))

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
	SteamProfileInfo, err := global.SteamTools.AccountServerClient.GetProfile(context.Background(), &steam_tools_grpc.AccountGetProfileRequest{
		SteamAid: steamaid,
	})
	if err != nil {
		zap.S().Errorf("发送报价调用steam_tools服务获取交易链接失败 %v", zap.Error(err))
		return "", err
	}
	if !SteamProfileInfo.Success {
		zap.S().Warnf("%d 发送报价检查账号信息失败", steamaid)
		return "", err
	}
	if SteamProfileInfo.Data.TradeUrl != nil && *SteamProfileInfo.Data.TradeUrl != "" {
		tradeUrl = *SteamProfileInfo.Data.TradeUrl
		zap.S().Warnf("new该steam账号未设置交易链接 %s", tradeUrl)
	} else {
		// 如果还是没有 则提示用户去设置交易链接
		zap.S().Warnf("%d 该steam账号未设置交易链接", steamaid)
		return "", err
	}
	return tradeUrl, nil
}

func getTradeCount(ctx context.Context, tempSteamAid int32) bool {
	//查看用户当下是否可以发起交易  一个用户最多发起5次交易
	offerListReq := &steam_tools_grpc.OfferGetOfferListRequest{
		SteamAid:   tempSteamAid,
		ActiveOnly: int32(1),
	}

	// 注意：请确认你的 global.SteamTools 中是否有 OfferServerClient 实例
	offerResp, err := global.SteamTools.OfferServerClient.GetOfferList(ctx, offerListReq)
	if err != nil {
		zap.S().Warnf("%d 调用Node gRPC服务[GetOfferList]失败! err %+v ", tempSteamAid, err)
		return false
	}

	// 3. 校验 Node 返回的业务逻辑状态
	if offerResp == nil || offerResp.Data == nil || !offerResp.GetSuccess() {
		zap.S().Warnf("%d 调用Node gRPC服务[GetOfferList]失败! get nil val err %+v ", tempSteamAid, err)
		return false
	}

	if len(offerResp.Data.TradeOffersSent) >= 5 {
		zap.S().Infof("%d beyond SendOffer limit %d", tempSteamAid, tempSteamAid)
		return false
	}

	return true
}

func DealExpireData(_event event.EventMsg) error {
	var params DealExpireDataParam
	err := json.Unmarshal(_event.Params, &params)
	if err != nil {
		zap.S().Error("事件 FinalDispose 解析消息参数失败", zap.Error(err))
		return err
	}

	ctx := context.Background()

	if !getTradeCount(ctx, params.SteamAID) {
		zap.S().Errorf(" getTradeCount data err %+v", params.SteamAID)
		return nil
	}

	//前置查询
	realTimeAssetIDMap, err := getRealTimeInventory(params.SteamAID)
	if err != nil {
		zap.S().Errorf(" getRealTimeInventory data err %+v %+v", params.SteamAID, zap.Error(err))
		return err
	}

	var tasks []TradeTask
	//通道数据
	var objectIds []primitive.ObjectID
	for _, id := range params.Ids {
		t, logID, err := prepareTradeTask(ctx, id, realTimeAssetIDMap)
		zap.S().Infof("logID = %s mongoId = %s", logID, id)
		if err != nil {
			zap.S().Warnf("prepareTradeTask 失败 id=%s err=%v", id.Hex(), err)
			continue
		}
		if t != nil {
			objectIds = append(objectIds, id)
			tasks = append(tasks, *t)
		}
	}
	myItem := make([]*steam_tools_grpc.OfferSendOfferRequest_TradeItem, 0, len(tasks))
	themItemsReq := []*steam_tools_grpc.OfferSendOfferRequest_TradeItem{}

	var logs []models.InventoryPackTradeTransferLog
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

	if len(tasks) == 0 {
		zap.S().Warnf("没有可执行的任务 steamAid=%d", params.SteamAID)
		return nil
	}

	//当TradeUrl是空的时候，加兜底逻辑
	if tasks[0].TradeUrl == "" {
		tasks[0].TradeUrl, err = reGetTradeUrl(tasks[0].TradeUrl, tasks[0].SteamAid)
		if err != nil {
			zap.S().Infof("SteamAid %s TradeUrl %s empty val", tasks[0].SteamAid, tasks[0].TradeUrl)
			return nil
		}
	}

	response, err := global.SteamTools.OfferServerClient.SendOffer(context.Background(), &steam_tools_grpc.OfferSendOfferRequest{
		SteamAid:    tasks[0].SteamAid,
		TradeUrl:    tasks[0].TradeUrl,
		MyItems:     myItem,
		ThemItems:   themItemsReq,
		AutoConfirm: 1,
	})

	if err != nil {
		zap.S().Errorf("调用steam_tools服务失败 response=%+v steamAid=%d err=%v",
			response,
			tasks[0].SteamAid,
			err,
		)
		return nil
	}

	if !response.Success {
		zap.S().Errorf("调用steam_tools服务失败 err=%+v response=%+v steamAid=%d", err, response, tasks[0].SteamAid)
		if response.Data == nil {
			zap.S().Errorf("发送报价失败，请稍后再试 err=%+v response=%+v steamAid=%d data=%v", err, response, tasks[0].SteamAid, response.Data)
		}
		if response.Data.ErrorCode != nil && *response.Data.ErrorCode == 112 {
			zap.S().Errorf("发送报价失败，请稍后再试 err=%+v response=%+v steamAid=%d data=%v StrError=%v", err, response, tasks[0].SteamAid, response.Data, *response.Data.StrError)
		}
		return nil
	}

	if response.Data == nil || response.Data.Tradeofferid == nil || *response.Data.Tradeofferid == "" {
		zap.S().Errorf("调用steam_tools服务失败,返回数据异常 response=%s steamAid=%d ", response, tasks[0].SteamAid)
		return nil
	}
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
			ReceivedSteamAID: task.SteamAid,
			ReceivedTradeURL: task.TradeUrl,
			PackID:           task.PackID,
			CategoryName:     task.CategoryName,
		})
	}

	//数据落库
	if len(logs) > 0 {
		if err := global.DB.Create(&logs).Error; err != nil {
			zap.S().Error("批量插入 inventory_pack_trade_transfer_log 失败", zap.Error(err))
		} else {
			zap.S().Infof("批量插入成功 %d 条", len(logs))
		}
	}

	if len(objectIds) > 0 {
		//写通道数据
		msg := event.NewEventMsg(map[string]interface{}{
			"ids":       objectIds,
			"steam_aid": tasks[0].SteamAid,
			"offer_id":  *response.Data.Tradeofferid, //交易id
		})
		global.CheckSendOfferChannel <- msg
	}

	return nil
}
