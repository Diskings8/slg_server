package marchs

import "server.slg.com/services/internal/cores/cores_declarations"

type Team struct {
	Heros    []cores_declarations.MarchHero
	Soldiers []cores_declarations.MarchSoldier
}

func (t *Team) GetAliveSoliderCount() uint64 {
	var sum = uint64(0)
	for _, v := range t.Soldiers {
		sum += v.GetCurCount()
	}
	return sum
}

func (t *Team) GetInjuredCount() uint64 {
	var sum = uint64(0)
	for _, v := range t.Soldiers {
		sum += v.GetInjuredCount()
	}
	return sum
}

func (t *Team) GetMaxCount() uint64 {
	var sum = uint64(0)
	for _, v := range t.Soldiers {
		sum += v.GetMaxCount()
	}
	return sum
}

func (t *Team) IsHasInjured() bool {
	for _, v := range t.Soldiers {
		if v.GetCurCount() == 0 {
			return true
		}
	}
	return false
}

func (t *Team) CheckCanFight() bool {
	if t.Soldiers[cores_declarations.HeroPose_0].GetCurCount() != 0 {
		return true
	}
	return false
}
