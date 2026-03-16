package appconf

type CMDConfig struct {
	Nacos
	Port int
}

// nacos
type Nacos struct {
	Address string
	IP      string
	Port    int
	User    string
	Pass    string
	DataId  string
	Group   string
	Key     string
}

type AppConfig struct {
	MysqlConf `json:"MysqlConf"`
	RedisConf `json:"RedisConf"`
	ZapConf   `json:"ZapConf"`
}

// mysql
type MysqlConf struct {
	User   string `json:"user"`
	Pass   string `json:"pass"`
	DbName string `json:"dbName"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

// redis

type RedisConf struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Pass string `json:"pass"`
}

// zap
type ZapConf struct {
	Path string `json:"path"`
}

type AppConfigMaster struct {
	DBUSER                     string `json:"DB_USER"`
	DBPASSWORD                 string `json:"DB_PASSWORD"`
	DBNAME                     string `json:"DB_NAME"`
	DBHOST                     string `json:"DB_HOST"`
	DBPORT                     int    `json:"DB_PORT"`
	REDISPASSWORD              string `json:"REDIS_PASSWORD"`
	REDISHOST                  string `json:"REDIS_HOST"`
	REDISPORT                  int    `json:"REDIS_PORT"`
	REDISDB                    int    `json:"REDIS_DB"`
	REDISCLUSTERHOST           string `json:"REDIS_CLUSTER_HOST"`
	REDISCLUSTERPORT           int    `json:"REDIS_CLUSTER_PORT"`
	REDISCLUSTERDB             int    `json:"REDIS_CLUSTER_DB"`
	REDISCLUSTERPASSWORD       string `json:"REDIS_CLUSTER_PASSWORD"`
	AESCRYPTKEY                string `json:"AES_CRYPT_KEY"`
	SECRETKEY                  string `json:"SECRET_KEY"`
	ESNODE                     string `json:"ES_NODE"`
	ESNODES                    string `json:"ES_NODES"`
	ESAPIKEY                   string `json:"ES_APIKEY"`
	YYMAPIHOST                 string `json:"YYM_API_HOST"`
	YYMSUPERPROXY              string `json:"YYM_SUPER_PROXY"`
	CHECKOUTCASHTOKEN          string `json:"CHECKOUT_CASH_TOKEN"`
	C5GAMECLIENTID             string `json:"C5GAME_CLIENT_ID"`
	C5GAMEREDIRECTURI          string `json:"C5GAME_REDIRECT_URI"`
	ALIPAYAPPCERTPATH          string `json:"ALIPAY_APP_CERT_PATH"`
	ALIPAYALIPAYROOTCERTPATH   string `json:"ALIPAY_ALIPAY_ROOT_CERT_PATH"`
	ALIPAYALIPAYPUBLICCERTPATH string `json:"ALIPAY_ALIPAY_PUBLIC_CERT_PATH"`
	YYMWEBHOST                 string `json:"YYM_WEB_HOST"`
	ZAP_LOG_PATH               string `json:"ZAP_LOG_PATH"`
	DB_MONGO_USER              string `json:"DB_MONGO_USER"`
	DB_MONGO_PASSWORD          string `json:"DB_MONGO_PASSWORD"`
	DB_MONGO_HOST              string `json:"DB_MONGO_HOST"`
	DB_MONGO_PORT              int    `json:"DB_MONGO_PORT"`
	ROCKETMQ_HOST              string `json:"ROCKETMQ_HOST"`
	ROCKETMQ_PORT              int    `json:"ROCKETMQ_PORT"`
	GO_STEAM_TOOLS_HOST        string `json:"GO_STEAM_TOOLS_HOST"`
	STATIC_SERVER              string `json:"STATIC_SERVER"`
	GO_CS2TRADE_SERVER_HOST    string `json:"GO_CS2TRADE_SERVER_HOST"`
}
