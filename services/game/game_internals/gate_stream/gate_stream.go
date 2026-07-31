package gate_stream

import (
	"context"
	"fmt"
	"sync"

	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/conns/rpcconn/rpc_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_streams"
	"server.slg.com/common/loggers"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// 方向：gate → game

var (
	gateStream      = make(map[uint64]*GateConnectStream)
	gateConnectLock sync.RWMutex
	globalCtx       context.Context
)

// Init 初始化，传入全局 context 以便服务关闭时通知所有连接
func Init(ctx context.Context) {
	globalCtx = ctx
}

// GateConnectStream 网关层连接
type GateConnectStream struct {
	*rpc_streams.GrpcStreamServer
	roleID  uint64
	isRobot bool
}

// Gate 获取网关流
func Gate(roleID uint64) (*GateConnectStream, bool) {
	gateConnectLock.RLock()
	defer gateConnectLock.RUnlock()
	d, ok := gateStream[roleID]
	return d, ok
}

// GateJoin 网关流加入
func GateJoin(roleID uint64, isRobot bool, conn pb_game.GameService_StreamServer, recv func(grpc.ServerStream) error) (*GateConnectStream, error) {
	gateConnectLock.Lock()
	defer gateConnectLock.Unlock()

	if roleID < 1 {
		return nil, fmt.Errorf("invalid roleID: %d", roleID)
	}

	if _, ok := gateStream[roleID]; ok {
		return nil, status.Error(codes.AlreadyExists, "gate connection already exists")
	}

	connectTmp := &GateConnectStream{
		roleID:  roleID,
		isRobot: isRobot,
	}
	var opts []rpc_streams.StreamServerOptionFunc
	if recv != nil {
		opts = append(opts, rpc_streams.WithServerReceiveFunc(recv))
	}
	connectTmp.GrpcStreamServer = rpc_streams.NewGRPCStreamServer(
		globalCtx,
		rpc_declarations.RpcStreamGate2Game,
		conn,
		opts...,
	)

	gateStream[roleID] = connectTmp
	return connectTmp, nil
}

// GateClose 连接断开后删除连接记录
func GateClose(roleID uint64) {
	gateConnectLock.Lock()
	defer gateConnectLock.Unlock()
	if _, ok := gateStream[roleID]; ok {
		delete(gateStream, roleID)
	}
}

// GateOnlineRoleIDs 获取当前在线的角色ID列表
func GateOnlineRoleIDs() []uint64 {
	gateConnectLock.RLock()
	defer gateConnectLock.RUnlock()

	roleIDs := make([]uint64, 0, len(gateStream))
	for roleID := range gateStream {
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs
}

// GateOnlineNum 获取在线人数
func GateOnlineNum() int {
	gateConnectLock.RLock()
	defer gateConnectLock.RUnlock()
	return len(gateStream)
}

// Push 推送消息给指定角色（异步）
func Push(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) {
	err := pushFunc(roleID, protocolID, msg)
	if err != nil {
		loggers.Logger.Error("push failed", zap.Uint64("role_id", roleID), zap.Error(err))
	}
}

// pushFunc 推送消息给指定角色
func pushFunc(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) error {
	roleConn, ok := Gate(roleID)
	if !ok {
		return nil
	}

	byteData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	return roleConn.Send(&pb_game.GameServiceNodePacketRsp{
		Packet: &pb_common.NodePacket{
			MsgId: protocolID,
			Message: &pb_common.MessagePacket{
				Body:    byteData,
				ErrCode: 0,
			},
		},
	})
}

// PushNodePacket 原样推送 NodePacket 给客户端（worldmap 下推透传，不重新序列化）
func PushNodePacket(roleID uint64, packet *pb_common.NodePacket) error {
	roleConn, ok := Gate(roleID)
	if !ok {
		return nil
	}
	return roleConn.Send(&pb_game.GameServiceNodePacketRsp{
		Packet: packet,
	})
}

// GateCallBackSuccess 调用接口后返回成功数据给客户端
func GateCallBackSuccess(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) error {
	roleConn, ok := Gate(roleID)
	if !ok {
		return nil
	}

	byteData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	err = roleConn.Send(&pb_game.GameServiceNodePacketRsp{
		Packet: &pb_common.NodePacket{
			MsgId: protocolID,
			Message: &pb_common.MessagePacket{
				Body:    byteData,
				ErrCode: 0,
			},
		},
	})
	if status.Code(err) == codes.Canceled {
		return nil
	}
	return err
}

// GateCallBackFail 调用接口后返回失败数据给客户端
func GateCallBackFail(roleID uint64, protocolID pb_protocol.MsgID, code pb_error_code.ErrorCode, devMsg string) error {
	roleConn, ok := Gate(roleID)
	if !ok {
		return nil
	}

	return roleConn.Send(&pb_game.GameServiceNodePacketRsp{
		Packet: &pb_common.NodePacket{
			MsgId: protocolID,
			Message: &pb_common.MessagePacket{
				DevMsg:  devMsg,
				ErrCode: code,
			},
		},
	})
}

// ShutDown 进程结束时断开所有连接
func ShutDown() {
	gateConnectLock.Lock()
	defer gateConnectLock.Unlock()

	for _, gConn := range gateStream {
		gConn.Close()
	}
}
