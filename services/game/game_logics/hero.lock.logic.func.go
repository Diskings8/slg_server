package game_logics

import (
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// HeroLock 锁定英雄：锁定后不可被消耗（升星消耗卡、分解等）
func HeroLock(hero *role_heroes.RoleHero) {
	hero.SetIsLocked(true)
}

// HeroUnlock 解锁英雄
func HeroUnlock(hero *role_heroes.RoleHero) {
	hero.SetIsLocked(false)
}
