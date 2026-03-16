package tools

import (
	"context"
	"go-tmp/cmd/global"
	"time"

	"go.uber.org/zap"
)

// Lock 尝试获取锁
func Lock(ctx context.Context, key, value string, expiration time.Duration) (bool, error) {
	// 使用 SET 命令尝试获取锁，NX 表示只有当键不存在时才设置，EX 表示设置过期时间
	set, err := global.RedisDB.SetNX(key, value, expiration).Result()
	if err != nil {
		zap.S().Error("Redis lock error", zap.Error(err))
		return false, err
	}
	return set, nil
}

// Unlock 释放锁
func Unlock(ctx context.Context, keys []string, value string) (bool, error) {
	// 使用 Lua 脚本保证原子性，先检查锁的值是否与当前值相同，相同则删除锁
	script :=
		`if redis.call("get", KEYS[1]) == ARGV[1] then
    		return redis.call("del", KEYS[1])
		else
    		return 0
		end`
	result, err := global.RedisDB.Eval(script, keys, value).Result()

	if err != nil {
		zap.S().Error("Redis unlock error", zap.Error(err))
		return false, err
	}
	// 判断返回值是否为 1，表示删除成功
	return result.(int64) == 1, nil
}
