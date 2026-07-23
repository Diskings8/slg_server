package attack

import (
	"server.slg.com/services/internal/cores/marchs"
)

// BattleSide 单方战斗结算数据
type BattleSide struct {
	// 关联行军（攻击方的 MarchInfo，或防御方驻军/守军的 MarchInfo）
	MarchInfo *marchs.MarchInfo
	// 战后状态
	IsDefeated     bool   // 是否溃败（队伍全灭）
	AliveSoldiers  uint64 // 战后存活士兵总数
	KilledSoldiers uint64 // 击杀对方士兵总数
	// 连胜计数
	WinCountInc uint32 // 本场胜场增加数
}

// BattleResult 战斗结算结果
type BattleResult struct {
	Attacker *BattleSide // 攻击方
	Defender *BattleSide // 防御方（驻军汇总）

	// 城墙/建筑伤害
	WallDamage   uint64 // 对目标建筑/城墙造成的拆迁值
	IsWallBroken bool   // 城墙是否被攻破（耐久归零）

	// 占领判定
	IsOccupied bool // 攻击方是否占领目标

	// 结算标记
	DefenderMarchUpdates []*marchs.MarchInfo // 需要推送更新的防御方行军（驻军受伤/溃败）
}
