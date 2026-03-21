package global

import "github.com/sbigtree/go-package-service/core/event"

var (
	TestChannel        = make(chan event.EventMsg, 1000) //示例
	ExpireDataChannel1 = make(chan event.EventMsg, 1000)
	ExpireDataChannel2 = make(chan event.EventMsg, 1000)
	ExpireDataChannel3 = make(chan event.EventMsg, 1000)
)

const InventoryPackTable = "my_inventory_pack"
