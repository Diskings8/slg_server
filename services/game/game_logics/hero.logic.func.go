package game_logics

import (
	"fmt"

	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// HeroLevelUp 英雄升级
//
//   - hero: 目标英雄
//   - 返回: 升级后的等级
//
// TODO: 接入配置表（等级上限、经验需求、每次升级属性点）
// TODO: 接入消耗系统（扣除道具/经验）
func HeroLevelUp(hero *role_heroes.RoleHero) (uint32, error) {
	curLevel := hero.GetLevel()

	// TODO: 从配置表读取等级上限
	const maxLevel uint32 = 100
	if curLevel >= maxLevel {
		return 0, fmt.Errorf("hero already max level: %d", curLevel)
	}

	newLevel := curLevel + 1
	hero.SetLevel(newLevel)
	hero.SetExp(0)

	return newLevel, nil
}

// HeroCultivate 英雄培养
//
//   - hero: 目标英雄
//   - cultivateType: 0=攻击 1=防御 2=智力 3=速度 4=拆迁
//   - 返回: 培养后的属性
//
// TODO: 接入消耗系统（扣除属性点/道具）
func HeroCultivate(hero *role_heroes.RoleHero, cultivateType uint32) (*pb_cultivate.Cultivate, error) {
	if cultivateType > 4 {
		return nil, fmt.Errorf("invalid cultivate type: %d", cultivateType)
	}

	// 补齐5维数组
	cultivates := hero.Cultivates
	for len(cultivates) < 5 {
		cultivates = append(cultivates, &pb_cultivate.Cultivate{})
	}

	attr := cultivates[cultivateType]
	attr.AddValCamp++

	hero.Cultivates = cultivates

	return attr, nil
}
