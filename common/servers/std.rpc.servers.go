package servers

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"server.slg.com/common/loggers"
	gsi "server.slg.com/common/servers/grpc_server_interfaces"
)

type RpcServer struct {
	server   *grpc.Server
	config   Config
	services []gsi.GRPCServiceI
	ctx      context.Context
}

func BuildRpcServer(ctx context.Context, cfg Config) *RpcServer {
	opts := []grpc.ServerOption{
		grpc.ConnectionTimeout(cfg.Timeout),
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
	}
	s := grpc.NewServer(opts...)
	return &RpcServer{
		server:   s,
		ctx:      ctx,
		config:   cfg,
		services: make([]gsi.GRPCServiceI, 0),
	}
}

func (s *RpcServer) RegisterServices(services ...gsi.GRPCServiceI) {
	s.services = append(s.services, services...)
	for _, svc := range services {
		svc.Register(s.server) // 调用服务自身注册逻辑
		loggers.Log.Info(fmt.Sprintf("注册 gRPC 服务: %s", svc.ServiceName()))
	}
}

func (s *RpcServer) Run() error {
	lis, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return err
	}

	loggers.Log.Info(fmt.Sprintf("gRPC 服务启动成功: %s", s.config.Addr))

	// 优雅关闭监听
	go func() {
		_ = s.server.Serve(lis)
	}()

	go func() {
		select {
		case <-s.ctx.Done():
			s.server.GracefulStop()
		}
	}()
	return nil
}
