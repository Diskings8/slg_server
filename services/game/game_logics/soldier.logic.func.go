package game_logics

import (
	"server.slg.com/api/game_conf"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// soldierLimitOf 计算英雄在指定城市的兵力上限：
// 英雄等级基础兵力 + 归属城市兵营等级累计加成。
// 配置缺失（game_conf 未初始化）时返回 0（调用方按无上限/默认处理）。
func soldierLimitOf(role *game_roles.Role, cityID uint64, heroLevel uint32) uint32 {
	gc := game_conf.Load()
	if gc == nil || gc.Soldier == nil {
		return 0
	}

	barrackLevel := uint32(0)
	if cityID > 0 {
		if b := role.GetBuildings().GetBarrackByCity(cityID); b != nil {
			barrackLevel = b.Level
		}
	}

	return gc.Soldier.SoldierLimit(heroLevel, barrackLevel)
}
