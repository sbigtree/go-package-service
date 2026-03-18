package steam_tools

import (
	steam_client_grpc "github.com/sbigtree/go-protobufs/generator/steam-client-server"
	steam_tools_grpc "github.com/sbigtree/steam-tools-grpc/go/generator"
)

type SteamTools struct {
	steam_tools_grpc.RootClient
	steam_tools_grpc.InventoryCsgoServerClient
	steam_tools_grpc.AccountServerClient
	steam_tools_grpc.Cs2ServerClient
	steam_tools_grpc.OfferServerClient
}

type SteamClient struct {
	steam_client_grpc.SteamClientGrpcClient
}
