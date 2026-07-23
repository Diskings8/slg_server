package attack

import (
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// triggerBattleEvents 触发战斗相关事件
//
// 预留事件钩子，后续 P1/P2 实现：
//   - 攻城事件：通知联盟成员
//   - 被攻击事件：通知防守方联盟
//   - 首占事件：记录首占
func triggerBattleEvents(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, result *BattleResult) {
	// TODO: P1-8 实现战争情报推送
	// TODO: 首占记录
	_ = mgr
	_ = attacker
	_ = result
}
