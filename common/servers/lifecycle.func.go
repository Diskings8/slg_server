package servers

import (
	"context"
	"net"
	"sync"

	"server.slg.com/common/loggers"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Lifecycle 服务生命周期管理器
// 管理异步/同步初始化、gRPC/TCP 服务器启动与关闭链
type Lifecycle struct {
	asyncInits []func()
	syncInits  []func()
	shutdowns  []func()

	grpcServer *grpc.Server
	grpcLis    net.Listener

	tcpServer *TcpServer
	tcpLis    net.Listener
}

// LifecycleOption 配置选项
type LifecycleOption func(*Lifecycle)

// WithAsyncInit 注册异步初始化任务（并发执行）
func WithAsyncInit(fns ...func()) LifecycleOption {
	return func(l *Lifecycle) {
		l.asyncInits = append(l.asyncInits, fns...)
	}
}

// WithSyncInit 注册同步初始化任务（异步初始化全部完成后按序执行）
func WithSyncInit(fns ...func()) LifecycleOption {
	return func(l *Lifecycle) {
		l.syncInits = append(l.syncInits, fns...)
	}
}

// WithShutdown 注册关闭回调（按注册逆序执行）
func WithShutdown(fns ...func()) LifecycleOption {
	return func(l *Lifecycle) {
		l.shutdowns = append(l.shutdowns, fns...)
	}
}

// WithGrpcServer 设置 gRPC Server 及其监听器
func WithGrpcServer(srv *grpc.Server, lis net.Listener) LifecycleOption {
	return func(l *Lifecycle) {
		l.grpcServer = srv
		l.grpcLis = lis
	}
}

// WithTcpServer 设置 TCP Server（由 Lifecycle 统一管理 listener 和关闭）
func WithTcpServer(srv *TcpServer) LifecycleOption {
	return func(l *Lifecycle) {
		l.tcpServer = srv
	}
}

// NewLifecycle 创建生命周期管理器
func NewLifecycle(opts ...LifecycleOption) *Lifecycle {
	l := &Lifecycle{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Run 启动服务生命周期，阻塞直到 ctx 被取消，然后执行关闭链后返回
//
// 执行顺序：
//  1. 异步初始化（并发执行）
//  2. 同步初始化（按序执行）
//  3. 启动 gRPC 服务（goroutine）
//  4. 启动 TCP 服务（goroutine）
//  5. 等待 ctx 取消
//  6. 运行关闭回调（逆序）
//  7. 关闭 gRPC 服务（GracefulStop）
//  8. 关闭 TCP 服务（关闭 listener + 断开所有连接）
func (l *Lifecycle) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	// 1. 异步初始化（并发执行）
	if len(l.asyncInits) > 0 {
		loggers.Logger.Info("开始异步初始化...")
		for _, fn := range l.asyncInits {
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
	if len(l.syncInits) > 0 {
		loggers.Logger.Info("开始同步初始化...")
		for _, fn := range l.syncInits {
			fn()
		}
		loggers.Logger.Info("同步初始化完成")
	}

	// 3. 启动 gRPC 服务（goroutine）
	if l.grpcServer != nil && l.grpcLis != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			loggers.Logger.Info("gRPC 服务开始监听", zap.String("addr", l.grpcLis.Addr().String()))
			if err := l.grpcServer.Serve(l.grpcLis); err != nil {
				loggers.Logger.Fatal("gRPC 服务异常退出", zap.Error(err))
			}
		}()
	}

	// 4. 启动 TCP 服务（goroutine）
	if l.tcpServer != nil {
		var err error
		l.tcpLis, err = net.Listen("tcp", l.tcpServer.config.Addr)
		if err != nil {
			loggers.Logger.Fatal("TCP 监听失败", zap.Error(err))
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.tcpServer.Serve(l.tcpLis)
		}()
	}

	// 5. 等待 ctx 取消
	<-ctx.Done()
	loggers.Logger.Info("context 已取消，执行关闭链...")

	// 6. 按逆序执行关闭回调
	for i := len(l.shutdowns) - 1; i >= 0; i-- {
		l.shutdowns[i]()
	}

	// 7. 关闭 gRPC 服务
	if l.grpcServer != nil {
		loggers.Logger.Info("关闭 gRPC 服务...")
		l.grpcServer.GracefulStop()
	}

	// 8. 关闭 TCP 服务
	if l.tcpLis != nil {
		loggers.Logger.Info("关闭 TCP 服务...")
		l.tcpLis.Close()       // 停止 accept，Serve goroutine 退出
		l.tcpServer.CloseAll() // 断开已有连接
	}

	wg.Wait()
	loggers.Logger.Info("服务已完全关闭")
	return nil
}
