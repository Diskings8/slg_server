package servers

import (
	"fmt"

	"google.golang.org/grpc"
	"server.slg.com/common/loggers"
	gsi "server.slg.com/common/servers/grpc_server_interfaces"
)

// GrpcBuilder gRPC Server 构造器，只负责构建和注册服务，不管理生命周期
// 启动（Serve）和关闭（GracefulStop）由 Lifecycle 统一管理
type GrpcBuilder struct {
	opts     []grpc.ServerOption
	services []gsi.GRPCServiceI
}

// NewGrpcBuilder 创建 gRPC 构造器
func NewGrpcBuilder() *GrpcBuilder {
	return &GrpcBuilder{}
}

// WithOptions 添加 gRPC ServerOption
func (b *GrpcBuilder) WithOptions(opts ...grpc.ServerOption) *GrpcBuilder {
	b.opts = append(b.opts, opts...)
	return b
}

// WithService 注册 gRPC 服务（需实现 GRPCServiceI 接口）
func (b *GrpcBuilder) WithService(svc gsi.GRPCServiceI) *GrpcBuilder {
	b.services = append(b.services, svc)
	return b
}

// Build 构建 *grpc.Server，注册所有服务后返回
func (b *GrpcBuilder) Build() *grpc.Server {
	srv := grpc.NewServer(b.opts...)
	for _, svc := range b.services {
		svc.Register(srv)
		loggers.Logger.Info(fmt.Sprintf("注册 gRPC 服务: %s", svc.ServiceName()))
	}
	return srv
}
