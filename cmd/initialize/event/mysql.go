package event

import (
	"database/sql"
	"fmt"
	"go-tmp/cmd/global"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//初始化连接mysql

func InitMysql() {
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", global.AppConfigMaster.DBUSER, global.AppConfigMaster.DBPASSWORD, global.AppConfigMaster.DBHOST, global.AppConfigMaster.DBPORT, global.AppConfigMaster.DBNAME)
	// 2. 先打开底层sql.DB连接（核心：为了配置连接池）
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		errMsg := fmt.Sprintf("MySQL底层连接打开失败：%s", err.Error())
		zap.S().Error(errMsg)
		panic(errMsg)
	}
	// 3. 配置数据库连接池（核心改造点）
	// 3.1 最大打开连接数：根据业务QPS调整，建议CPU核心数×2+有效连接数（示例：50）
	sqlDB.SetMaxOpenConns(50)
	// 3.2 最大空闲连接数：建议等于或略小于MaxOpenConns（示例：20）
	sqlDB.SetMaxIdleConns(20)
	// 3.3 连接最大存活时间：30分钟，需小于数据库wait_timeout（MySQL默认8小时）
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	// 3.4 连接最大空闲时间：10分钟，超时清理空闲连接
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	// 4. 验证连接有效性（避免创建无效连接）
	if err := sqlDB.Ping(); err != nil {
		errMsg := fmt.Sprintf("MySQL连接验证失败：%s", err.Error())
		zap.S().Error(errMsg)
		panic(errMsg)
	}

	global.DB, err = gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB, // 关键：使用配置好连接池的底层连接
	}), &gorm.Config{})
	if err != nil {
		zap.S().Error("MySQL数据库连接失败" + err.Error())
		panic("MySQL数据库连接失败" + err.Error())
	}
	zap.S().Info("MySQL数据库连接成功")
}
