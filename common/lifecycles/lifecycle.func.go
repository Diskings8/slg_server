package lifecycles

import (
	"context"
	"net"
	"sync"

	"server.slg.com/common/loggers"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server 服务生命周期管理器，管理异步/同步初始化、gRPC 启动与关闭链
type Server struct {
	asyncInits []func()
	syncInits  []func()
	shutdowns  []func()

	grpcServer *grpc.Server
	listener   net.Listener
}

// Option 配置选项
type Option func(*Server)

// WithAsyncInit 注册异步初始化任务（并发执行）
func WithAsyncInit(fns ...func()) Option {
	return func(s *Server) {
		s.asyncInits = append(s.asyncInits, fns...)
	}
}

// WithSyncInit 注册同步初始化任务（异步初始化全部完成后按序执行）
func WithSyncInit(fns ...func()) Option {
	return func(s *Server) {
		s.syncInits = append(s.syncInits, fns...)
	}
}

// WithShutdown 注册关闭回调（按注册逆序执行）
func WithShutdown(fns ...func()) Option {
	return func(s *Server) {
		s.shutdowns = append(s.shutdowns, fns...)
	}
}

// WithGrpcServer 设置 gRPC Server
func WithGrpcServer(srv *grpc.Server) Option {
	return func(s *Server) {
		s.grpcServer = srv
	}
}

// WithListener 设置 gRPC 监听器
func WithListener(lis net.Listener) Option {
	return func(s *Server) {
		s.listener = lis
	}
}

// New 创建生命周期管理器
func New(opts ...Option) *Server {
	s := &Server{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run 启动服务，阻塞直到 ctx 被取消，然后执行关闭链后返回
func (s *Server) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	// 1. 异步初始化（并发执行）
	if len(s.asyncInits) > 0 {
		loggers.Logger.Info("开始异步初始化...")
		for _, fn := range s.asyncInits {
			wg.Add(1)
			go func(f func()) {
				defer wg.Done()
				f()
			}(fn)
		}
		wg.Wait()
		loggers.Logger.Info("异步初始化完成")
	}

	// 2. 同步初始化（按序执行）
	if len(s.syncInits) > 0 {
		loggers.Logger.Info("开始同步初始化...")
		for _, fn := range s.syncInits {
			fn()
		}
		loggers.Logger.Info("同步初始化完成")
	}

	// 3. 启动 gRPC 服务
	if s.grpcServer != nil && s.listener != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			loggers.Logger.Info("gRPC 服务开始监听", zap.String("addr", s.listener.Addr().String()))
			if err := s.grpcServer.Serve(s.listener); err != nil {
				loggers.Logger.Fatal("gRPC 服务异常退出", zap.Error(err))
			}
		}()
	}

	// 4. 等待 ctx 取消
	<-ctx.Done()
	loggers.Logger.Info("context 已取消，执行关闭链...")

	// 5. 按逆序执行关闭回调
	for i := len(s.shutdowns) - 1; i >= 0; i-- {
		s.shutdowns[i]()
	}

	// 6. 优雅关闭 gRPC
	if s.grpcServer != nil {
		loggers.Logger.Info("关闭 gRPC...")
		s.grpcServer.GracefulStop()
	}

	wg.Wait()
	loggers.Logger.Info("服务已完全关闭")
	return nil
}
