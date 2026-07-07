package rpc_streams

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/common/conns/rpcconn/rpc_declarations"
	"server.slg.com/common/loggers"
)

type GrpcStreamClient struct {
	conn       grpc.ClientStream
	rwLock     sync.RWMutex
	name       rpc_declarations.RpcStreamName
	connFunc   func(ctx context.Context) (grpc.ClientStream, error)
	cancelFunc context.CancelFunc
	opts       *streamClientOptions
}

func (c *GrpcStreamClient) Name() string {
	return string(c.name)
}

func (c *GrpcStreamClient) SetName(name rpc_declarations.RpcStreamName) {
	c.name = name
}

func (c *GrpcStreamClient) Close() {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()
	if c.conn == nil {
		return
	}
	c.cancelFunc()
}

// Get 取得连接
func (c *GrpcStreamClient) Get() (grpc.ClientStream, error) {
	c.rwLock.RLock()
	conn := c.conn
	if conn != nil {
		c.rwLock.RUnlock()
		return conn, nil
	}
	c.rwLock.RUnlock()

	err := c.Connect()
	if err != nil {
		return nil, err
	}

	if c.conn == nil {
		return nil, fmt.Errorf("没有连接到%s服务", c.Name())
	}
	return c.conn, nil
}

func (c *GrpcStreamClient) Connect() error {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	if c.connFunc == nil {
		return fmt.Errorf("%s无连接函数", c.Name())
	}

	ctx, cancelFunc := context.WithCancel(c.opts.ctx)
	c.cancelFunc = cancelFunc

	var err error
	c.conn, err = c.connFunc(ctx)
	if err != nil {
		return err
	}
	go c.loopHandle()
	loggers.Logger.Debug("服务连接成功", zap.String("name", c.Name()))
	return nil
}

// ReConnect 重连服务
func (c *GrpcStreamClient) ReConnect() {
	if c.opts.reConnectTime > 0 {
	reTry:
		time.Sleep(c.opts.reConnectTime)

		var err error
		select {
		case <-c.opts.closeChan:
			loggers.Logger.Debug("ReConnect 本服务主动关闭", zap.String("name", c.Name()))
			c.Close()
			return
		default:
			err = c.Connect()
			if err == nil {
				loggers.Logger.Info("服务重新连接成功", zap.String("name", c.Name()))
				return
			} else if status.Code(err) == codes.Canceled {
				return
			}

			loggers.Logger.Error("服务重新连接失败", zap.String("name", c.Name()), zap.Error(err))
			goto reTry
		}
	}
}

func (c *GrpcStreamClient) loopHandle() {
	loggers.Logger.Debug("服务检查开始", zap.String("name", c.Name()))

	for {
		select {
		case <-c.opts.closeChan:
			loggers.Logger.Debug("本服务主动关闭", zap.String("name", c.Name()))
			c.Close()
			return
		case <-c.conn.Context().Done():
			loggers.Logger.Info("服务连接断开", zap.String("name", c.Name()))

			c.ReConnect()
			return
		default:
			var err error
			if c.opts.receiveFunc != nil {
				err = c.opts.receiveFunc(c.conn)
			} else {
				err = c.conn.RecvMsg(nil)
			}
			if err != nil {
				if err == io.EOF || status.Code(err) == codes.Canceled || status.Code(err) == codes.Unavailable {
					continue
				} else if status.Code(err) == codes.AlreadyExists {
					loggers.Logger.Warn("已有连接,重试中", zap.String("name", c.Name()), zap.Error(err))
					c.ReConnect()
					return
				}
				loggers.Logger.Error(err.Error())
				continue
			}
		}
	}
}
