package mq_common

import (
	"context"
	"encoding/json"
	"github.com/sbigtree/go-package-service/cmd/global"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
)

// 发送消息到 RocketMQ 通用方法
func SendRocketMQMessage(topic string, body interface{}, tags ...string) (*primitive.SendResult, error) {
	var msgBytes []byte
	var err error

	// 判断 body 类型
	switch v := body.(type) {
	case string:
		msgBytes = []byte(v)
	case []byte:
		msgBytes = v
	default:
		msgBytes, err = json.Marshal(v)
		if err != nil {
			zap.S().Error("RocketMQ 消息序列化失败!",
				zap.String("topic", topic),
				zap.Any("body", body),
				zap.Error(err))
			return nil, err
		}
	}

	// 构造消息
	msg := &primitive.Message{
		Topic: topic,
		Body:  msgBytes,
	}
	if len(tags) > 0 && tags[0] != "" {
		msg.WithTag(tags[0])
	}

	// 发送消息
	response, err := global.RocketMQProducer.SendSync(context.Background(), msg)
	if err != nil {
		zap.S().Error("RocketMQ 消息发送失败!",
			zap.String("topic", topic),
			zap.ByteString("body", msgBytes),
			zap.Error(err))
		return nil, err
	}

	zap.S().Infof("RocketMQ 消息发送成功 topic %s tag %v msgId %s", topic, tags, response.MsgID)

	return response, nil
}
