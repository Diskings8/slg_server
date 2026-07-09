package map_buildings

import "server.slg.com/api/protocol/pb/pb_city"

type NpcCity struct {
	NpcBuilding
	ID             uint32
	CurOccUnionID  uint64
	FirstOccRecord *pb_city.CityFirstOccRecord
}
