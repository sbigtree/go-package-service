package event

import (
	"encoding/json"
	"errors"
)

type EventMsg struct {
	Event  int32           `json:"event"`
	Retry  int32           `json:"retry"` // 重试次数
	Params json.RawMessage `json:"params"`
}

var ErrLocked = errors.New("获取锁失败")

func NewEventMsg(params map[string]interface{}) EventMsg {
	msg := EventMsg{}
	b, _ := json.Marshal(params)
	msg.Params = b
	return msg
}
