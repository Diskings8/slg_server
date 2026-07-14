package map_buildings

import "server.slg.com/services/internal/cores/cores_declarations"

type NpcBuilding struct {
	BaseBuilding
}

func (nb *NpcBuilding) BeforeBeAttack(cores_declarations.MarchInfoI) bool {
	return true
}
