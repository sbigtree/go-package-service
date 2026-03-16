package event

import (
	"github.com/sbigtree/go-package-service/cmd/global"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

// InitElasticsearchClient 初始化 Elasticsearch 客户端
func InitElasticsearchClient() {
	parts := strings.Split(global.AppConfigMaster.ESNODES, ",")
	Addresses := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		Addresses = append(Addresses, p)
	}
	cfg := elasticsearch.Config{
		Addresses: Addresses,
		APIKey:    global.AppConfigMaster.ESAPIKEY,
	}
	var err error
	global.ESClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		zap.S().Error("创建Elasticsearch客户端失败" + err.Error())
		return
	}
	// 测试连接
	res, err := global.ESClient.Info()
	if err != nil {
		zap.S().Error("获取响应失败" + err.Error())
		return
	}
	defer res.Body.Close()
	zap.S().Info("✅ Elasticsearch客户端连接成功")
}
