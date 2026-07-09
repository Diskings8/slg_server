package map_buildings

import "server.slg.com/services/internal/cores/cores_declarations"

type NpcBuilding struct {
	BaseBuildings
}

func (nb *NpcBuilding) BeforeBeAttack(cores_declarations.MarchInfoI) bool {
	return true
}

func (nb *NpcBuilding) BeAttack(info cores_declarations.MarchInfoI) (right uint64, isBroken bool) {
	return nb.ReduceBuildingsHp(info.GetRelocationVal())
}
