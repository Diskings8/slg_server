package rpc_streams

import (
	"io"
	"sync/atomic"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"server.slg.com/common/conns/rpcconn/rpc_declarations"
	"server.slg.com/common/loggers"
)

type GrpcStreamServer struct {
	name   rpc_declarations.RpcStreamName
	opts   *streamServerOptions
	conn   grpc.ServerStream
	closed atomic.Int32
}

func (s *GrpcStreamServer) Name() string {
	return string(s.name)
}

func (s *GrpcStreamServer) Get() grpc.ServerStream {
	return s.conn
}

func (s *GrpcStreamServer) Send(msg any) error {
	return s.conn.SendMsg(msg)
}

func (s *GrpcStreamServer) Close() {
	s.closed.CompareAndSwap(0, 1)
}

func (s *GrpcStreamServer) WaitDone() {
	select {
	case <-s.opts.ctx.Done():
		loggers.Logger.Info(" 本服务主动断开", zap.String("name", s.Name()))
	case <-s.conn.Context().Done():
		loggers.Logger.Info(" 服务链接断开", zap.String("name", s.Name()))
	}
}

func (s *GrpcStreamServer) loop() {
	var err error
	for {
		select {
		case <-s.opts.ctx.Done():
			loggers.Logger.Info(s.Name() + " 本服务主动断开")
			return
		case <-s.conn.Context().Done():
			loggers.Logger.Info(s.Name() + " 服务链接断开")
			return
		default:
			err = s.opts.receiveFunc(s.conn)
			if err != nil {
				if err == io.EOF {
					s.Close()
					return
				}
				loggers.Logger.Error(s.Name() + err.Error())
			}
			continue
		}
	}
}
