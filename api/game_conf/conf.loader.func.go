package game_conf

import (
	"fmt"
	"os"
	"path/filepath"

	"server.slg.com/api/game_conf/battle"
	"server.slg.com/api/game_conf/exchange"
	"server.slg.com/api/game_conf/gacha"
	"server.slg.com/api/game_conf/hero"
	"server.slg.com/api/game_conf/item"
	"server.slg.com/api/game_conf/skill"
	"server.slg.com/api/game_conf/table"
	"server.slg.com/api/game_conf/troop"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/loggers"
	"go.uber.org/zap"
)

// newEmbedded 构建全域 Go 内嵌配置（InitDefault 用，不校验）。
func newEmbedded() *GameConf {
	return &GameConf{
		Battle:   battle.New(),
		Hero:     hero.New(),
		Skill:    skill.New(),
		Item:     item.New(),
		Troop:    troop.New(),
		Gacha:    gacha.New(),
		Exchange: exchange.New(),
	}
}

// newBattleEmbedded 构建 battle 节点子集（仅 battle+skill，其余 nil）。
func newBattleEmbedded() *GameConf {
	return &GameConf{
		Battle:    battle.New(),
		Skill:     skill.New(),
		battleOnly: true,
	}
}

// loadAll 从 dir 加载全部注册表 + 跨表校验。
//
// 文件缺失 → 告警并保留该域 Go 内嵌；任一步失败 → 返回 err（调用方不替换旧配置）。
func loadAll(dir string) (*GameConf, error) {
	gc := newEmbedded()
	if err := loadTables(gc, dir, allTables); err != nil {
		return nil, err
	}
	if err := validateCrossRefs(gc); err != nil {
		return nil, err
	}
	gc.filePath = dir
	gc.globalVersion = nextVersion()
	gc.tableVersions = collectVersions(gc, allTables)
	return gc, nil
}

// loadTablesFrom 从 dir 加载指定表集（battle 节点子集用）。
func loadTablesFrom(dir string, regs []tableReg) (*GameConf, error) {
	gc := newBattleEmbedded()
	if err := loadTables(gc, dir, regs); err != nil {
		return nil, err
	}
	if err := validateCrossRefs(gc); err != nil {
		return nil, err
	}
	gc.filePath = dir
	gc.battleOnly = true
	gc.globalVersion = nextVersion()
	gc.tableVersions = collectVersions(gc, regs)
	return gc, nil
}

// loadTables 按注册表逐表读取 JSON → Load → Validate。
func loadTables(gc *GameConf, dir string, regs []tableReg) error {
	for _, r := range regs {
		data, err := os.ReadFile(filepath.Join(dir, r.file+".json"))
		if err != nil {
			if os.IsNotExist(err) {
				loggers.Logger.Warn("table json not found, keep embedded", zap.String("table", r.file))
				continue
			}
			return fmt.Errorf("read table %s json: %w", r.file, err)
		}
		if err := loadOne(r.get(gc), data); err != nil {
			return err
		}
	}
	return nil
}

// loadOne 单表加载 + 表内校验。
func loadOne(t table.Table, data []byte) error {
	if err := t.Load(data); err != nil {
		return fmt.Errorf("table %s load: %w", t.FileName(), err)
	}
	if err := t.Validate(); err != nil {
		return fmt.Errorf("table %s validate: %w", t.FileName(), err)
	}
	return nil
}

// collectVersions 收集各表内容版本（仅 JSON 加载的表有 hash）。
func collectVersions(gc *GameConf, regs []tableReg) map[string]string {
	versions := make(map[string]string, len(regs))
	for _, r := range regs {
		if v := r.get(gc).Version(); v != "" {
			versions[r.file] = v
		}
	}
	return versions
}

// versionsEqual 两张版本表是否一致（内容未变则热更跳过原子替换）。
func versionsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// validateCrossRefs 跨表引用校验。
//
// 表内校验在各域 Conf.Validate 完成，这里只做跨表引用（避免 game_conf ↔ 域包循环依赖）。
// 仅当**引用方表为 JSON 加载**（Version() != ""）时校验——内嵌占位视为可信现状（可能含历史不一致）。
// battle 子集 nil 表跳过（如 skill 引用 item，但 battle 节点不加载 item）。
func validateCrossRefs(gc *GameConf) error {
	// skill：升级消耗道具 → item；收藏所需英雄卡 → hero
	if gc.Skill.Version() != "" && gc.Item != nil {
		for _, s := range gc.Skill.AllSkills() {
			if !gc.Item.Has(s.UpgradeCost.ItemID) {
				return fmt.Errorf("skill %d upgrade_cost item %d not found", s.ConfID, s.UpgradeCost.ItemID)
			}
		}
	}
	if gc.Skill.Version() != "" && gc.Hero != nil {
		for _, cc := range gc.Skill.AllCollections() {
			for _, h := range cc.NeedHeroes {
				if _, ok := gc.Hero.HeroConf(int32(h.ItemID)); !ok {
					return fmt.Errorf("skill collection %d need hero %d not found", cc.SkillConfID, h.ItemID)
				}
			}
		}
	}
	// troop：扩展兵种解锁道具 → item
	if gc.Troop != nil && gc.Troop.Version() != "" && gc.Item != nil {
		if !gc.Item.Has(pb_confs.ItemID(gc.Troop.UnlockItemConf)) {
			return fmt.Errorf("troop unlock_item_conf %d not found in items", gc.Troop.UnlockItemConf)
		}
	}
	// gacha：英雄掉落/心愿 → hero；道具掉落/抽卡券 → item
	if gc.Gacha != nil && gc.Gacha.Version() != "" {
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
