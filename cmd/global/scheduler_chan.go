package global

import "go-tmp/core/event"

var (
	TestChannel = make(chan event.EventMsg, 1000) //示例
)
