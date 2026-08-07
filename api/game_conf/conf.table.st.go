package game_conf

import (
	"server.slg.com/api/game_conf/battle"
	"server.slg.com/api/game_conf/exchange"
	"server.slg.com/api/game_conf/gacha"
	"server.slg.com/api/game_conf/guard"
	"server.slg.com/api/game_conf/hero"
	"server.slg.com/api/game_conf/item"
	"server.slg.com/api/game_conf/skill"
	"server.slg.com/api/game_conf/soldier"
	"server.slg.com/api/game_conf/troop"
)

// GameConf 游戏配置聚合。
//
// 各配置域（Battle/Hero/Skill/...）为 Go typed Conf（访问层，公开接口稳定）；
// 数据源支持 Go 内嵌占位（InitDefault）与 JSON 配置表（Init/ReLoad 走 registry 加载）。
type GameConf struct {
	filePath      string            // JSON 配置目录（"" = Go 内嵌）
	globalVersion uint64            // 全局配置版本（每次成功加载/热更 +1；内嵌基线=1）
	tableVersions map[string]string // 表名 → 内容 hash（仅 JSON 加载的表有值）
	battleOnly    bool              // battle 节点子集标记（仅加载 battle+skill）

	Battle   *battle.Conf
	Hero     *hero.Conf
	Skill    *skill.Conf
	Item     *item.Conf
	Troop    *troop.Conf
	Gacha    *gacha.Conf
	Exchange *exchange.Conf
	Guard    *guard.Conf
	Soldier  *soldier.Conf
}

// Version 全局配置版本（内嵌基线=1；JSON 加载/热更每次成功 +1）。
func (c *GameConf) Version() uint64 { return c.globalVersion }

// TableVersions 各表内容版本（表名 → 内容 hash；内嵌表不在此），返回只读副本。
func (c *GameConf) TableVersions() map[string]string {
	cp := make(map[string]string, len(c.tableVersions))
	for k, v := range c.tableVersions {
		cp[k] = v
	}
	return cp
}

// FilePath JSON 配置目录（"" = Go 内嵌）。
func (c *GameConf) FilePath() string { return c.filePath }
