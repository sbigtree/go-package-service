package event

import (
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/scheduler"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// 初始化定时任务
func InitScheduler() {
	// 初始化定时任务调度器
	global.Cron = cron.New(cron.WithSeconds())

	// 注册定时任务
	scheduler.RegisterTasks()

	// 启动定时任务调度器
	global.Cron.Start()
	zap.S().Info("✅ 定时任务调度器已启动")
}
