package test

import (
	"encoding/json"
	"fmt"
	"github.com/sbigtree/go-package-service/core/event"
	"github.com/sbigtree/go-package-service/core/throttle"
	"go.uber.org/zap"
)

type TestParam struct {
	OrderNo       string `json:"order_no"`        // 业务订单号
	MasterOrderNo string `json:"master_order_no"` // 主订单号
}

// 并发转同步限制器
var Throttle = throttle.NewLocalSyncThrottle[string]()

// 测试方法
func Test(_event event.EventMsg) error {
	var params TestParam
	err := json.Unmarshal(_event.Params, &params)
	if err != nil {
		zap.S().Error("测试消费者 解析消息参数失败", zap.Error(err))
		return nil
	}
	// 转同步
	release := Throttle.Allow(params.OrderNo)
	defer release()

	logId := fmt.Sprintf("测试消费者 %v", params.OrderNo)
	zap.S().Infof("%s", logId)
	return nil
}
