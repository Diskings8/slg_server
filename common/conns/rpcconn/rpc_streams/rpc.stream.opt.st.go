package rpc_streams

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

var reConnectDefaultTime = time.Second

type streamClientOptions struct {
	receiveFunc   func(stream grpc.ClientStream) error
	reConnectTime time.Duration
	closeChan     chan struct{}
	ctx           context.Context
}

type StreamClientOptionFunc func(*streamClientOptions)

func evalClientOptions(ctx context.Context, cliOptFs []StreamClientOptionFunc) *streamClientOptions {
	opt := &streamClientOptions{
		closeChan:     make(chan struct{}),
		ctx:           ctx,
		reConnectTime: reConnectDefaultTime,
	}
	for _, o := range cliOptFs {
		o(opt)
	}
	return opt
}

// WithClientReceiveFunc 接收信息
func WithClientReceiveFunc(f func(grpc.ClientStream) error) StreamClientOptionFunc {
	return func(o *streamClientOptions) {
		o.receiveFunc = f
	}
}

// WithClientReConnectTime 重连时长
func WithClientReConnectTime(t time.Duration) StreamClientOptionFunc {
	return func(o *streamClientOptions) {
		o.reConnectTime = t
	}
}

type StreamServerOptionFunc func(*streamServerOptions)

type streamServerOptions struct {
	ctx         context.Context
	receiveFunc func(stream grpc.ServerStream) error
}

func evalServerOptions(ctx context.Context, cliOptFs []StreamServerOptionFunc) *streamServerOptions {
	opt := &streamServerOptions{
		ctx: ctx,
	}
	for _, o := range cliOptFs {
		o(opt)
	}
	return opt
}

// WithServerReceiveFunc 接收信息
func WithServerReceiveFunc(f func(stream grpc.ServerStream) error) StreamServerOptionFunc {
	return func(o *streamServerOptions) {
		o.receiveFunc = f
	}
}
