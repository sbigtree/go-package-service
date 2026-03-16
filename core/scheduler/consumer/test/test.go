package consumer_test

import (
	"fmt"
	"go-tmp/core/event"
)

func Test1(req event.EventMsg) error {
	fmt.Println("core/schedule/consumer/ 消费者测试方法")
	return nil
}
