package event

import (
	steam_client_grpc "github.com/sbigtree/go-protobufs/generator/steam-client-server"
	steam_tools_grpc "github.com/sbigtree/steam-tools-grpc/go/generator"

	"github.com/sbigtree/go-package-service/cmd/appconf/steam_tools"
	"github.com/sbigtree/go-package-service/cmd/global"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func InitSteamToolsClient() {
	//初始化连接箱子交易服务
	// 1.连接

	conn, err := grpc.Dial(global.AppConfigMaster.GO_STEAM_TOOLS_HOST, grpc.WithInsecure())
	if err != nil {
		zap.S().Error("连接steam_tools服务异常： %s\n", err)
	}
	// 2. 实例化gRPC客户端
	global.SteamTools = &steam_tools.SteamTools{
		RootClient:                steam_tools_grpc.NewRootClient(conn),
		InventoryCsgoServerClient: steam_tools_grpc.NewInventoryCsgoServerClient(conn),
		AccountServerClient:       steam_tools_grpc.NewAccountServerClient(conn),
		Cs2ServerClient:           steam_tools_grpc.NewCs2ServerClient(conn),
		OfferServerClient:         steam_tools_grpc.NewOfferServerClient(conn),
	}
	// 2. 实例化gRPC客户端

	conn2, err := grpc.Dial(global.AppConfigMaster.GO_STEAM_CLIENT_HOST, grpc.WithInsecure())
	if err != nil {
		zap.S().Error("连接SteamClient服务异常： %s\n", err)
	}

	global.SteamClient = steam_client_grpc.NewSteamClientGrpcClient(conn2)
	zap.S().Info("✅ init steam_tools success")
}
