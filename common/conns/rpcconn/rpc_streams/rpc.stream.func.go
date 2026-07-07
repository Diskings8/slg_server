package rpc_streams

import (
	"context"

	"google.golang.org/grpc"
	"server.slg.com/common/conns/rpcconn/rpc_declarations"
)

func NewGRPCStreamServer(name rpc_declarations.RpcStreamName, stream grpc.ServerStream, opts ...StreamServerOptionFunc) *GrpcStreamServer {
	conn := &GrpcStreamServer{
		name: name,
		conn: stream,
	}
	conn.opts = evalServerOptions(opts)

	if conn.opts.receiveFunc != nil {
		go conn.loop()
	}
	return conn
}

// NewGRPCStreamClient 创建流客户端
func NewGRPCStreamClient(name rpc_declarations.RpcStreamName, connFunc func(ctx context.Context) (grpc.ClientStream, error), opts ...StreamClientOptionFunc) *GrpcStreamClient {
	conn := &GrpcStreamClient{
		name:     name,
		connFunc: connFunc,
	}
	conn.opts = evalClientOptions(opts)
	return conn
}
