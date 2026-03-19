package scheduler

import (
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/scheduler/jobs/mypackage"
	"github.com/sbigtree/go-package-service/core/scheduler/jobs/test"
	"log"
)

func RegisterTasks() {
	tasks := []struct {
		Spec string
		Job  func()
	}{
		//定时任务在这里注册 按照模块在jobs当中划分
		{"0 0/2 * * * *", test.Test1},                      //测试
		{"0/10 * * * * *", mypackage.FindExpireData},       //背包数据处理
		{"0 0/5 * * * *", mypackage.FindPackTradeTransfer}, //背包数据处理
	}
	//查找mongo表中数据
	go mypackage.FindExpireData()
	//查inventory_pack_trade_transfer_logs表中数据
	mypackage.FindPackTradeTransfer()

	for _, task := range tasks {
		_, err := global.Cron.AddFunc(task.Spec, task.Job)
		if err != nil {
			log.Fatalf("注册任务失败: %v", err)
		}
	}
}
