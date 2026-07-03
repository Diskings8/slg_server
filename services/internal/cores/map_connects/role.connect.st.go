package map_connects

import (
	"sync"

	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/services/internal/cores/cores_declarations"
)

type RoleConnect struct {
	RwLock           sync.RWMutex
	stream           pb_worldmap.WorldMapNodeService_StreamServer
	roleID           uint64
	cityMapID        cores_declarations.MapID
	scaleLevel       cores_declarations.ScaleLevel
	oldMapID         cores_declarations.MapID
	minMapScaleLevel cores_declarations.ScaleLevel
}

func NewRoleConnect(roleID uint64, cityMapID cores_declarations.MapID, stream pb_worldmap.WorldMapNodeService_StreamServer) *RoleConnect {
	rc := &RoleConnect{
		stream:           stream,
		roleID:           roleID,
		cityMapID:        cityMapID,
		scaleLevel:       cores_declarations.ScaleLevel0,
		oldMapID:         cityMapID,
		minMapScaleLevel: cores_declarations.ScaleLevel1,
	}
	return rc
}
