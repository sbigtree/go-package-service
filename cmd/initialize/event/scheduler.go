package event

import (
	"go-tmp/cmd/global"
	"go-tmp/core/scheduler"

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
