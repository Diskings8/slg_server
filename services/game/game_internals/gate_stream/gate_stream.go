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

// defaultManager 包内默认网关流管理器（私有单例）
var defaultManager = &Manager{
	streams: make(map[uint64]*GateConnectStream),
}

// Manager 网关流管理器，持有全局连接状态
type Manager struct {
	streams map[uint64]*GateConnectStream
	lock    sync.RWMutex
	ctx     context.Context
}

// GateConnectStream 网关层连接
type GateConnectStream struct {
	*rpc_streams.GrpcStreamServer
	roleID  uint64
	isRobot bool
}

// Init 初始化，传入全局 context 以便服务关闭时通知所有连接
func Init(ctx context.Context) {
	defaultManager.Init(ctx)
}

// Init 初始化，传入全局 context
func (m *Manager) Init(ctx context.Context) {
	m.ctx = ctx
}

// Gate 获取网关流
func Gate(roleID uint64) (*GateConnectStream, bool) {
	return defaultManager.Gate(roleID)
}

// Gate 获取网关流
func (m *Manager) Gate(roleID uint64) (*GateConnectStream, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	d, ok := m.streams[roleID]
	return d, ok
}

// GateJoin 网关流加入
func GateJoin(roleID uint64, isRobot bool, conn pb_game.GameService_StreamServer, recv func(grpc.ServerStream) error) (*GateConnectStream, error) {
	return defaultManager.GateJoin(roleID, isRobot, conn, recv)
}

// GateJoin 网关流加入
func (m *Manager) GateJoin(roleID uint64, isRobot bool, conn pb_game.GameService_StreamServer, recv func(grpc.ServerStream) error) (*GateConnectStream, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if roleID < 1 {
		return nil, fmt.Errorf("invalid roleID: %d", roleID)
	}

	if _, ok := m.streams[roleID]; ok {
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
		m.ctx,
		rpc_declarations.RpcStreamGate2Game,
		conn,
		opts...,
	)

	m.streams[roleID] = connectTmp
	return connectTmp, nil
}

// GateClose 连接断开后删除连接记录
func GateClose(roleID uint64) {
	defaultManager.GateClose(roleID)
}

// GateClose 连接断开后删除连接记录
func (m *Manager) GateClose(roleID uint64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.streams[roleID]; ok {
		delete(m.streams, roleID)
	}
}

// GateOnlineRoleIDs 获取当前在线的角色ID列表
func GateOnlineRoleIDs() []uint64 {
	return defaultManager.GateOnlineRoleIDs()
}

// GateOnlineRoleIDs 获取当前在线的角色ID列表
func (m *Manager) GateOnlineRoleIDs() []uint64 {
	m.lock.RLock()
	defer m.lock.RUnlock()

	roleIDs := make([]uint64, 0, len(m.streams))
	for roleID := range m.streams {
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs
}

// GateOnlineNum 获取在线人数
func GateOnlineNum() int {
	return defaultManager.GateOnlineNum()
}

// GateOnlineNum 获取在线人数
func (m *Manager) GateOnlineNum() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.streams)
}

// Push 推送消息给指定角色（同步发送，错误仅记日志，不向外返回）
func Push(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) {
	defaultManager.Push(roleID, protocolID, msg)
}

// Push 推送消息给指定角色（同步发送，错误仅记日志，不向外返回）
func (m *Manager) Push(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) {
	err := m.pushFunc(roleID, protocolID, msg)
	if err != nil {
		loggers.Logger.Error("push failed", zap.Uint64("role_id", roleID), zap.Error(err))
	}
}

// pushFunc 推送消息给指定角色
func (m *Manager) pushFunc(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) error {
	roleConn, ok := m.Gate(roleID)
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
	return defaultManager.PushNodePacket(roleID, packet)
}

// PushNodePacket 原样推送 NodePacket 给客户端（worldmap 下推透传，不重新序列化）
func (m *Manager) PushNodePacket(roleID uint64, packet *pb_common.NodePacket) error {
	roleConn, ok := m.Gate(roleID)
	if !ok {
		return nil
	}
	return roleConn.Send(&pb_game.GameServiceNodePacketRsp{
		Packet: packet,
	})
}

// GateCallBackSuccess 调用接口后返回成功数据给客户端
func GateCallBackSuccess(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) error {
	return defaultManager.GateCallBackSuccess(roleID, protocolID, msg)
}

// GateCallBackSuccess 调用接口后返回成功数据给客户端
func (m *Manager) GateCallBackSuccess(roleID uint64, protocolID pb_protocol.MsgID, msg proto.Message) error {
	roleConn, ok := m.Gate(roleID)
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
	return defaultManager.GateCallBackFail(roleID, protocolID, code, devMsg)
}

// GateCallBackFail 调用接口后返回失败数据给客户端
func (m *Manager) GateCallBackFail(roleID uint64, protocolID pb_protocol.MsgID, code pb_error_code.ErrorCode, devMsg string) error {
	roleConn, ok := m.Gate(roleID)
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
	defaultManager.ShutDown()
}

// ShutDown 进程结束时断开所有连接
func (m *Manager) ShutDown() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, gConn := range m.streams {
		gConn.Close()
	}
}
