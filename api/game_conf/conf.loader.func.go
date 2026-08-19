package game_conf

import (
	"fmt"
	"os"
	"path/filepath"

	"server.slg.com/api/game_conf/battle"
	"server.slg.com/api/game_conf/building"
	"server.slg.com/api/game_conf/exchange"
	"server.slg.com/api/game_conf/formation"
	"server.slg.com/api/game_conf/gacha"
	"server.slg.com/api/game_conf/guard"
	"server.slg.com/api/game_conf/hero"
	"server.slg.com/api/game_conf/item"
	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/game_conf/resource"
	"server.slg.com/api/game_conf/review"
	"server.slg.com/api/game_conf/skill"
	"server.slg.com/api/game_conf/soldier"
	"server.slg.com/api/game_conf/table"
	"server.slg.com/api/game_conf/troop"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/utils/util_jsons"
)

// gameconfigFileName 单一配置表产物名（tabtoy 导出，与内嵌 gameconfigjson 同源）。
const gameconfigFileName = "gameconfig.json"

// newEmbedded 构建全域配置（内嵌 gameconfig.json → pb.Table → NewFromPB），供 InitDefault/New("") 使用。
//
// 内嵌数据损坏或校验失败 → panic（数据源与文件路径同源，正常不应发生）。
func newEmbedded() *GameConf {
	t, err := gameconfigjson.Table()
	if err != nil {
		panic(fmt.Sprintf("embedded gameconfig invalid: %v", err))
	}
	gc, err := newFromPB(t, false)
	if err != nil {
		panic(fmt.Sprintf("embedded gameconfig build failed: %v", err))
	}
	return gc
}

// newBattleEmbedded 构建 battle 节点子集（battle+skill，其余 nil）。
func newBattleEmbedded() *GameConf {
	t, err := gameconfigjson.Table()
	if err != nil {
		panic(fmt.Sprintf("embedded gameconfig invalid: %v", err))
	}
	gc, err := newFromPB(t, true)
	if err != nil {
		panic(fmt.Sprintf("embedded gameconfig build failed: %v", err))
	}
	return gc
}

// resolveGameconfig 解析配置入口：config_path 为目录时指向其中的 gameconfig.json，为文件时直接用。
func resolveGameconfig(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat config path %q: %w", path, err)
	}
	if fi.IsDir() {
		return filepath.Join(path, gameconfigFileName), nil
	}
	return path, nil
}

// loadAll 从 config_path 加载全部配置表 + 跨表校验（battleOnly=false）。
//
// 单文件缺失 → 返回 err（fail-fast），替代原「缺表保持内嵌」语义；任一步失败不产生半更新。
func loadAll(dir string) (*GameConf, error) {
	return loadFromPath(dir, false)
}

// loadBattle 从 config_path 加载 battle 节点子集（battle+skill）。
func loadBattle(dir string) (*GameConf, error) {
	return loadFromPath(dir, true)
}

// loadFromPath 从 config_path 读入单一 gameconfig.json → pb.Table → newFromPB 构建。
func loadFromPath(dir string, battleOnly bool) (*GameConf, error) {
	file, err := resolveGameconfig(dir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read gameconfig %q: %w", file, err)
	}
	t := &pb_gameconfig.Table{}
	if err := util_jsons.Unmarshal(data, t); err != nil {
		return nil, fmt.Errorf("unmarshal gameconfig %q: %w", file, err)
	}
	gc, err := newFromPB(t, battleOnly)
	if err != nil {
		return nil, err
	}
	gc.filePath = dir
	gc.battleOnly = battleOnly
	gc.contentHash = table.ContentHash(data)
	gc.globalVersion = nextVersion()
	gc.tableVersions = map[string]string{gameconfigFileName: gc.contentHash}
	return gc, nil
}

// newFromPB 从 pb.Table 构建 GameConf（battleOnly=true 时仅 battle+skill，其余 nil）。
//
// 各域 NewFromPB 均为「局部构建 + 末尾提交 + 校验」，失败返回 err 不产生半更新；
// 全量构建完成后做跨表校验（validateCrossRefs）。
func newFromPB(t *pb_gameconfig.Table, battleOnly bool) (*GameConf, error) {
	gc := &GameConf{pb: t, battleOnly: battleOnly}
	if battleOnly {
		var err error
		if gc.Battle, err = battle.NewFromPB(t); err != nil {
			return nil, fmt.Errorf("battle: %w", err)
		}
		if gc.Skill, err = skill.NewFromPB(t); err != nil {
			return nil, fmt.Errorf("skill: %w", err)
		}
		return gc, nil
	}

	var err error
	if gc.Hero, err = hero.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("hero: %w", err)
	}
	if gc.Skill, err = skill.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("skill: %w", err)
	}
	if gc.Item, err = item.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("item: %w", err)
	}
	if gc.Troop, err = troop.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("troop: %w", err)
	}
	if gc.Exchange, err = exchange.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("exchange: %w", err)
	}
	if gc.Battle, err = battle.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("battle: %w", err)
	}
	if gc.Gacha, err = gacha.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("gacha: %w", err)
	}
	if gc.Guard, err = guard.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("guard: %w", err)
	}
	if gc.Resource, err = resource.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("resource: %w", err)
	}
	if gc.Review, err = review.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("review: %w", err)
	}
	if gc.Soldier, err = soldier.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("soldier: %w", err)
	}
	if gc.Building, err = building.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("building: %w", err)
	}
	if gc.Formation, err = formation.NewFromPB(t); err != nil {
		return nil, fmt.Errorf("formation: %w", err)
	}
	if err := validateCrossRefs(gc); err != nil {
		return nil, err
	}
	return gc, nil
}

// validateCrossRefs 跨表引用校验。
//
// 表内校验在各域 NewFromPB 内完成（Validate），这里只做跨表引用（避免 game_conf ↔ 域包循环依赖）。
// 数据源统一为 gameconfig.json（文件或内嵌），故不再区分「JSON 已加载」——全量构建必然 13 域齐全；
// battle 子集 nil 表跳过（如 skill 引用 item/hero，但 battle 节点不加载）。
func validateCrossRefs(gc *GameConf) error {
	// skill：升级消耗道具 → item；收藏所需英雄卡 → hero
	if gc.Item != nil {
		for _, s := range gc.Skill.AllSkills() {
			if !gc.Item.Has(s.UpgradeCost.ItemID) {
				return fmt.Errorf("skill %d upgrade_cost item %d not found", s.ConfID, s.UpgradeCost.ItemID)
			}
		}
	}
	if gc.Hero != nil {
		for _, cc := range gc.Skill.AllCollections() {
			for _, h := range cc.NeedHeroes {
				if _, ok := gc.Hero.HeroConf(int32(h.ItemID)); !ok {
					return fmt.Errorf("skill collection %d need hero %d not found", cc.SkillConfID, h.ItemID)
				}
			}
		}
	}
	// troop：扩展兵种解锁道具 → item
	if gc.Troop != nil && gc.Item != nil {
		if !gc.Item.Has(pb_confs.ItemID(gc.Troop.UnlockItemConf)) {
			return fmt.Errorf("troop unlock_item_conf %d not found in items", gc.Troop.UnlockItemConf)
		}
	}
	// gacha：英雄掉落/心愿 → hero；道具掉落/抽卡券 → item
	if gc.Gacha != nil {
		if gc.Hero != nil {
			for _, poolID := range gc.Gacha.AllPoolIDs() {
				pool, _ := gc.Gacha.GetPool(poolID)
				for _, g := range pool.DropGroups {
					for _, it := range g.Items {
						if it.IsHero {
							if _, ok := gc.Hero.HeroConf(it.RewardConfID); !ok {
								return fmt.Errorf("gacha pool %d group %d hero %d not found", poolID, g.GroupID, it.RewardConfID)
							}
						}
					}
				}
				for _, h := range pool.WishHeros {
					if _, ok := gc.Hero.HeroConf(h); !ok {
						return fmt.Errorf("gacha pool %d wish hero %d not found", poolID, h)
					}
				}
			}
		}
		if gc.Item != nil {
			for _, poolID := range gc.Gacha.AllPoolIDs() {
				pool, _ := gc.Gacha.GetPool(poolID)
				for _, g := range pool.DropGroups {
					for _, it := range g.Items {
						if !it.IsHero {
							if !gc.Item.Has(pb_confs.ItemID(it.RewardConfID)) {
								return fmt.Errorf("gacha pool %d group %d item %d not found", poolID, g.GroupID, it.RewardConfID)
							}
						}
					}
				}
				if pool.TicketConfID != 0 && !gc.Item.Has(pb_confs.ItemID(pool.TicketConfID)) {
					return fmt.Errorf("gacha pool %d ticket item %d not found", poolID, pool.TicketConfID)
				}
			}
		}
	}
	return nil
}
