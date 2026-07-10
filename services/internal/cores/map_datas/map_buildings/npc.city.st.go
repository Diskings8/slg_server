package map_buildings

import (
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/services/internal/cores/cores_declarations"
)

type NpcCity struct {
	NpcBuilding
	ID             uint32                          // 城市id
	CurOccUnionID  uint64                          // 当前占领同盟
	FirstOccRecord *pb_city.CityFirstOccRecord     // 首占记录
	CityGarrison   []cores_declarations.MarchInfoI // 城市驻军
}
