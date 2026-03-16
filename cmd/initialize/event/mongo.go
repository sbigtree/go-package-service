package event

import (
	"context"
	"fmt"
	"github.com/sbigtree/go-package-service/cmd/global"
	"net/url"
	"time"

	mongo_db "github.com/sbigtree/go-db-model/v2/mongo"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// InitMongoDB 初始化 MongoDB 连接，并赋值给全局变量 global.MongoDB
func InitMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	MongoDBURL := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d",
		global.AppConfigMaster.DB_MONGO_USER,
		url.QueryEscape(global.AppConfigMaster.DB_MONGO_PASSWORD),
		global.AppConfigMaster.DB_MONGO_HOST,
		global.AppConfigMaster.DB_MONGO_PORT,
	)
	// yuanyoumaoabcd

	clientOpts := options.Client().
		ApplyURI(MongoDBURL).
		SetConnectTimeout(30 * time.Second). // 设置连接超时
		SetSocketTimeout(60 * time.Second).  // 设置读写超时
		SetMaxPoolSize(100)                  // 设置连接池最大连接数

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		zap.S().Errorf("连接 MongoDB 失败: %v", err)
		panic("连接 MongoDB 失败: " + err.Error())
	}

	// 检查连接
	if err := client.Ping(ctx, nil); err != nil {
		zap.S().Errorf("MongoDB Ping 失败: %v", err)
		panic("MongoDB Ping 失败: " + err.Error())
	}
	zap.S().Info("✅ MongoDB 连接成功")
	global.MongoDB = client.Database(mongo_db.DataBase) // 记得替换成你实际使用的 DB 名
	mongo_db.Init(global.MongoDB)
	// 使用样例
	//mongo_models.MyInventoryPack{}.Collection(global.MongoDB).InsertOne(ctx, mongo_models.MyInventoryPack{})

}
