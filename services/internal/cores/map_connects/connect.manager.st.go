package map_connects

import (
	"errors"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/conns/rpcconn/rpc_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_streams"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/aois"
	"server.slg.com/services/internal/cores/cores_declarations"
)

type RoleConnectManager struct {
	connects map[uint64]*RoleConnect
	aoi      *aois.ScreenData
	sync.RWMutex
}

func NewRoleConnectManager(aoi *aois.ScreenData) *RoleConnectManager {
	manager := &RoleConnectManager{
		connects: make(map[uint64]*RoleConnect),
		aoi:      aoi,
	}

	return manager
}

func (rcm *RoleConnectManager) NewRoleConnect(name rpc_declarations.RpcStreamName, roleID uint64, mapID cores_declarations.MapID, roleConn grpc.ServerStream, receiveF func(grpc.ServerStream) error) (*RoleConnect, error) {
	if roleID < 1 {
		return nil, errors.New("invalid roleID")
	}
	if defaultAllConnectManager.isStop.Load() {
		return nil, errors.New("all connect stop")
	}
	rcm.Lock()
	defer rcm.Unlock()
	if _, ok := rcm.connects[roleID]; ok {
		return nil, errors.New("role connect already exists")
	}
	conn := NewRoleConnect(roleID)
	var opts []rpc_streams.StreamServerOptionFunc
	if receiveF != nil {
		opts = append(opts, rpc_streams.WithServerReceiveFunc(receiveF))
	}

	conn.SetStream(rpc_streams.NewGRPCStreamServer(defaultAllConnectManager.ctx, name, roleConn, opts...))

	rcm.aoi.Move(conn, mapID)
	rcm.connects[roleID] = conn
	return conn, nil
}

func (rcm *RoleConnectManager) CloseRoleConnect(roleID uint64) {
	rcm.Lock()
	defer rcm.Unlock()
	if c, ok := rcm.connects[roleID]; ok {
		rcm.aoi.Exit(c)
	}
	delete(rcm.connects, roleID)
}

func (rcm *RoleConnectManager) LoadRoleConnect(roleID uint64) (*RoleConnect, bool) {
	rcm.RLock()
	defer rcm.RUnlock()
	d, ok := rcm.connects[roleID]
	return d, ok
}

func (rcm *RoleConnectManager) WaitDone(conn *RoleConnect) {
	conn.GetStream().WaitDone()
	rcm.CloseRoleConnect(conn.GetRoleID())
}

func (rcm *RoleConnectManager) SetRoleScreen(roleID uint64, mapID cores_declarations.MapID) {
	conn, ok := rcm.LoadRoleConnect(roleID)
	if ok {
		conn.SetScreenMapID(mapID)
	}
}

// Range 当前所有链接执行
func (rcm *RoleConnectManager) Range(f func(conn *RoleConnect) bool) {
	rcm.RLock()
	list := make([]*RoleConnect, 0, len(rcm.connects))
	for _, v := range rcm.connects {
		list = append(list, v)
	}
	rcm.RUnlock()
	for _, v := range list {
		if !f(v) {
			return
		}
	}
}

// CloseAll 当前所有链接执行断开
func (rcm *RoleConnectManager) CloseAll() {
	rcm.Range(func(conn *RoleConnect) bool {
		conn.GetStream().Close()
		return true
	})
}

// GetConnectRoleIDs 当前所有链接的角色ID
func (rcm *RoleConnectManager) GetConnectRoleIDs() []uint64 {
	rcm.RLock()
	list := make([]uint64, 0, len(rcm.connects))
	for roleID := range rcm.connects {
		list = append(list, roleID)
	}
	rcm.RUnlock()
	return list
}

//-----------------------Push---------------------//

func (rcm *RoleConnectManager) PushToScreen(messageID pb_protocol.MsgID, msgNotify proto.Message, mapIDList ...cores_declarations.MapID) {
	if len(mapIDList) < 1 {
		loggers.Logger.Error("send in empty mapIDList")
		return
	}
	byteData, err := proto.Marshal(msgNotify)
	if err != nil {
		loggers.Logger.Error(err.Error())
		return
	}
	var removeDuplicates = make(map[uint64]cores_declarations.MapRoleConnectI)
	var needSendMsg = &pb_common.NodePacket{
		MsgId: messageID,
		Message: &pb_common.MessagePacket{
			Body:    byteData,
			DevMsg:  "",
			ErrCode: pb_protocol.ErrorCode_NoneErr,
		},
	}
	for _, mapID := range mapIDList {
		removeDuplicates = rcm.aoi.AroundConnects(mapID, removeDuplicates)
	}
	for roleID, roleConnect := range removeDuplicates {
		sendErr := roleConnect.Send(needSendMsg)
		if sendErr != nil {
			code := status.Code(sendErr)
			if code == codes.Canceled || code == codes.Unavailable || code == codes.Unknown {
				rcm.CloseRoleConnect(roleID)
				continue
			}
			loggers.Logger.Error(sendErr.Error())
		}
	}
}

func (rcm *RoleConnectManager) pushMsgDataToRole(protocolID pb_protocol.MsgID, byteData []byte, roleID uint64) {
	c, ok := rcm.LoadRoleConnect(roleID)
	if ok {
		err := c.Send(&pb_common.NodePacket{
			MsgId: protocolID,
			// Sn:       sn,
			Message: &pb_common.MessagePacket{
				Body:    byteData,
				ErrCode: pb_protocol.ErrorCode_NoneErr,
			},
		})
		if err != nil {
			code := status.Code(err)
			if code == codes.Canceled || code == codes.Unavailable || code == codes.Unknown {
				// gRPC 传输层已关闭，主动清理 AOI 中的残留连接
				rcm.CloseRoleConnect(roleID)
			} else {
				loggers.Logger.Error(err.Error(), zap.Uint64("role_id", roleID))
			}
		}
	}
}

func (rcm *RoleConnectManager) PushToRoleIDs(messageID pb_protocol.MsgID, msgNotify proto.Message, roleIDList ...uint64) {
	if len(roleIDList) < 1 {
		loggers.Logger.Error("send in empty mapIDList")
		return
	}
	byteData, err := proto.Marshal(msgNotify)
	if err != nil {
		loggers.Logger.Error(err.Error())
		return
	}

	for _, roleID := range roleIDList {
		rcm.pushMsgDataToRole(messageID, byteData, roleID)
	}
}

func (rcm *RoleConnectManager) PushToRoleID(messageID pb_protocol.MsgID, msgNotify proto.Message, roleID uint64) {
	byteData, err := proto.Marshal(msgNotify)
	if err != nil {
		loggers.Logger.Error(err.Error())
		return
	}
	rcm.pushMsgDataToRole(messageID, byteData, roleID)
}
