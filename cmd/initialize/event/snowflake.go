package event

import (
	"go-tmp/cmd/global"

	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

// 初始化 Snowflake 节点
func InitSnowflake(nodeID int64) {
	global.SnowflakeOnce.Do(func() {
		var err error
		global.SnowflakeNode, err = snowflake.NewNode(nodeID)
		if err != nil {
			zap.S().Error("初始化 Snowflake 节点失败" + err.Error())
			return
		}
	})
	zap.S().Info("Snowflake 节点初始化成功")
}
