package mysql_methods

import (
	"context"
	"fmt"
	"github.com/sbigtree/go-package-service/cmd/global"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 模拟mysql方法
// 定义一个通用的事务执行函数，接受一个处理函数作为参数
func ExecuteTransaction(ctx context.Context, f func(tx *gorm.DB) error) (err error) {
	db := global.DB
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r) // 保持原样抛出
		}
	}()

	if err = f(tx); err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			zap.S().Errorf("事务回滚失败: %v, 原始错误: %v", rbErr, err)
			return fmt.Errorf("rollback failed: %v, original error: %v", rbErr, err)
		}
		return err
	}

	if commitErr := tx.Commit().Error; commitErr != nil {
		zap.S().Error("事务提交失败:", commitErr)
		return commitErr
	}

	return nil
}

//通用
