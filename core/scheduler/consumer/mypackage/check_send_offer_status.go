package mypackage

import (
	"encoding/json"
	"github.com/sbigtree/go-package-service/core/event"
	"go.uber.org/zap"
)

func CheckSendOfferStatus(_event event.EventMsg) error {
	var params DealExpireDataParam
	err := json.Unmarshal(_event.Params, &params)
	if err != nil {
		zap.S().Error("事件 FinalDispose 解析消息参数失败", zap.Error(err))
		return err
	}

	return nil
}
