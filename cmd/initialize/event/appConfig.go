package event

import (
	"encoding/base64"
	"encoding/json"
	"github.com/sbigtree/go-package-service/cmd/global"
	"github.com/sbigtree/go-package-service/core/tools"
	"log"
	"os"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func InitAppConfig() {
	// 1. 读取原始 YAML 文件内容
	rawYaml, err := os.ReadFile("cmd/appconfig.yaml") // 根据实际路径修改
	if err != nil {
		panic("读取配置文件失败：" + err.Error())
	}

	// 2. 替换 YAML 中的 ${XXX} 为系统环境变量的值
	expanded := os.ExpandEnv(string(rawYaml))
	// 3. 使用 Viper 读取替换后的配置字符串
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(expanded)); err != nil {
		zap.S().Error("viper 读取配置失败: ", err)
		panic(err)
	}

	// 4. 将配置绑定到全局变量 global.CMDConfig.Nacos
	if err := viper.Unmarshal(&global.CMDConfig); err != nil {
		zap.S().Error("解析 Nacos 配置失败: ", err)
		panic(err)
	}
	zap.S().Info("Nacos 配置信息读取成功! 开始运行应用。")

	// 5. 创建 Nacos 客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:         global.CMDConfig.Nacos.Address,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "debug",
		Username:            global.CMDConfig.Nacos.User,
		Password:            global.CMDConfig.Nacos.Pass,
	}

	// 6. 创建 Nacos 服务端配置
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: global.CMDConfig.Nacos.IP,
			Port:   uint64(global.CMDConfig.Nacos.Port),
		},
	}

	// 7. 初始化 Nacos 客户端
	Nacosclient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		zap.S().Error("创建 Nacos 客户端失败: ", err)
		panic(err)
	}

	// 8. 从 Nacos 获取配置内容
	config, err := Nacosclient.GetConfig(vo.ConfigParam{
		DataId: global.CMDConfig.Nacos.DataId,
		Group:  global.CMDConfig.Nacos.Group,
	})
	if err != nil {
		zap.S().Error("获取 Nacos 配置失败: ", err)
		panic(err)
	}

	// 9. 解析 JSON 配置内容
	var configs map[string]interface{}
	if err := json.Unmarshal([]byte(config), &configs); err != nil {
		zap.S().Error("解析 Nacos 配置失败: ", err)
		panic(err)
	}
	log.Printf("configs %+v", configs)
	key, err := base64.StdEncoding.DecodeString(global.CMDConfig.Nacos.Key)
	if err != nil {
		zap.S().Error("解析 Nacos Key 失败: ", err)
		panic(err)
	}

	// 10. 解密带有加密前缀的字段
	for k, v := range configs {
		if str, ok := v.(string); ok && strings.HasPrefix(str, "{e}") {
			plain := tools.RemoveEncryptionPrefix(str)
			decrypted, err := tools.DecryptAES(plain, key, "base64")
			if err != nil {
				zap.S().Error("解密 Nacos 配置失败, key: ", k, ", err: ", err)
				panic(err)
			}
			configs[k] = decrypted
		}
	}

	// 11. 映射解密后的 map 到结构体
	configBytes, err := json.Marshal(configs)
	if err != nil {
		zap.S().Error("序列化解密后的配置失败: ", err)
		panic(err)
	}
	if err := json.Unmarshal(configBytes, &global.AppConfigMaster); err != nil {
		zap.S().Error("反序列化 AppConfigMaster 失败: ", err)
		panic(err)
	}
	log.Printf("AppConfigMaster %+v", global.AppConfigMaster)
	// 使用默认值
	if global.AppConfigMaster.GO_STEAM_TOOLS_HOST == "" {
		global.AppConfigMaster.GO_STEAM_TOOLS_HOST = os.Getenv("GO_STEAM_TOOLS_HOST")
	}
	zap.S().Info("AppConfig 读取成功!", global.AppConfigMaster.DBUSER)
}
