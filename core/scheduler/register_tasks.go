package scheduler

import (
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/scheduler/jobs/test"
	"log"
)

func RegisterTasks() {
	tasks := []struct {
		Spec string
		Job  func()
	}{
		//定时任务在这里注册 按照模块在jobs当中划分
		{"0 0/2 * * * *", test.Test1}, //测试

	}

	for _, task := range tasks {
		_, err := global.Cron.AddFunc(task.Spec, task.Job)
		if err != nil {
			log.Fatalf("注册任务失败: %v", err)
		}
	}
}
