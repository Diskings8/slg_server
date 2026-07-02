package grpc_server_interfaces

import "google.golang.org/grpc"

type GRPCServiceI interface {
	// ServiceName 服务唯一名称（用于日志、监控标识）
	ServiceName() string

	// Register 注册服务到 gRPC 服务器
	Register(srv *grpc.Server)
}
