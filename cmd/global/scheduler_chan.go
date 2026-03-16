package global

import "github.com/sbigtree/go-package-service/core/event"

var (
	TestChannel = make(chan event.EventMsg, 1000) //示例
)
