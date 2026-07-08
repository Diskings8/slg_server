package map_connects

import (
	"sync"

	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/common/conns/rpcconn/rpc_streams"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// 验证接口
var _ cores_declarations.MapRoleConnectI = new(RoleConnect)

type RoleConnect struct {
	RwLock           sync.RWMutex
	stream           *rpc_streams.GrpcStreamServer
	roleID           uint64                        // 角色id
	cityMapID        cores_declarations.MapID      // 主城地图id
	scaleLevel       cores_declarations.ScaleLevel // 主屏幕视野等级
	curMapID         cores_declarations.MapID      // 当前关注的地图id
	minMapScaleLevel cores_declarations.ScaleLevel // 小地图视野等级
}

func NewRoleConnect(roleID uint64) *RoleConnect {
	rc := &RoleConnect{
		roleID:           roleID,
		cityMapID:        cores_declarations.InvalidMapID,
		scaleLevel:       cores_declarations.ScaleLevel0,
		curMapID:         cores_declarations.InvalidMapID,
		minMapScaleLevel: cores_declarations.ScaleLevel1,
	}
	return rc
}

func (rc *RoleConnect) SetScreenMapID(mapID cores_declarations.MapID) {
	rc.RwLock.Lock()
	defer rc.RwLock.Unlock()
	rc.curMapID = mapID
}

// GetScreenMapID 取得当前屏幕地图id
func (rc *RoleConnect) GetScreenMapID() cores_declarations.MapID {
	rc.RwLock.RLock()
	defer rc.RwLock.RUnlock()
	return rc.curMapID
}

// CheckInCity 检查是否在城内
func (rc *RoleConnect) CheckInCity() bool {
	rc.RwLock.RLock()
	defer rc.RwLock.RUnlock()
	return rc.curMapID == cores_declarations.InvalidMapID
}

func (rc *RoleConnect) GetRoleID() uint64 {
	rc.RwLock.RLock()
	defer rc.RwLock.RUnlock()
	return rc.roleID
}

// Send 发送数据
func (rc *RoleConnect) Send(data *pb_common.NodePacket) error {
	return rc.stream.Send(data)
}

func (rc *RoleConnect) SetStream(stream *rpc_streams.GrpcStreamServer) {
	rc.stream = stream
}

func (rc *RoleConnect) GetStream() (stream *rpc_streams.GrpcStreamServer) {
	return rc.stream
}
