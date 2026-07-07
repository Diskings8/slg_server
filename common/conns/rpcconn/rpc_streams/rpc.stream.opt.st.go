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

func evalClientOptions(cliOptFuncs []StreamClientOptionFunc) *streamClientOptions {
	opt := &streamClientOptions{
		closeChan:     make(chan struct{}),
		ctx:           context.Background(),
		reConnectTime: reConnectDefaultTime,
	}
	for _, o := range cliOptFuncs {
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

// WithClientCloseChan 断开通道
func WithClientCloseChan(ch chan struct{}) StreamClientOptionFunc {
	return func(o *streamClientOptions) {
		o.closeChan = ch
	}
}

// WithClientContext 携带context
func WithClientContext(ctx context.Context) StreamClientOptionFunc {
	return func(o *streamClientOptions) {
		o.ctx = ctx
	}
}

type StreamServerOptionFunc func(*streamServerOptions)

type streamServerOptions struct {
	receiveFunc func(stream grpc.ServerStream) error
	closeChan   chan struct{}
}

func evalServerOptions(cliOptFuncs []StreamServerOptionFunc) *streamServerOptions {
	opt := &streamServerOptions{
		closeChan: make(chan struct{}),
	}
	for _, o := range cliOptFuncs {
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

// WithServerCloseChan 断开通道
func WithServerCloseChan(ch chan struct{}) StreamServerOptionFunc {
	return func(o *streamServerOptions) {
		o.closeChan = ch
	}
}
