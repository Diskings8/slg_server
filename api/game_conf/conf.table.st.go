package game_conf

import (
	"server.slg.com/api/game_conf/battle"
	"server.slg.com/api/game_conf/building"
	"server.slg.com/api/game_conf/exchange"
	"server.slg.com/api/game_conf/formation"
	"server.slg.com/api/game_conf/gacha"
	"server.slg.com/api/game_conf/guard"
	"server.slg.com/api/game_conf/hero"
	"server.slg.com/api/game_conf/item"
	"server.slg.com/api/game_conf/resource"
	"server.slg.com/api/game_conf/review"
	"server.slg.com/api/game_conf/skill"
	"server.slg.com/api/game_conf/soldier"
	"server.slg.com/api/game_conf/troop"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// GameConf 游戏配置聚合。
//
// 各配置域（Battle/Hero/Skill/...）为 Go typed Conf（访问层，公开接口稳定）；
// 数据源统一为单一 gameconfig.json（文件 Init/ReLoad 或内嵌 InitDefault，同源同构）。
type GameConf struct {
	filePath      string            // config_path（"" = 内嵌）
	globalVersion uint64            // 全局配置版本（每次成功加载/热更 +1；内嵌基线=1）
	tableVersions map[string]string // 内容版本：gameconfig.json → 内容 hash（内嵌为空表）
	contentHash   string            // 单一 gameconfig.json 内容 hash（FNV-32a；内嵌为 ""）
	battleOnly    bool              // battle 节点子集标记（仅加载 battle+skill）
	pb            *pb_gameconfig.Table // 原始配置表（统一数据源，All() 暴露；nil = 未加载）

	Battle    *battle.Conf
	Hero      *hero.Conf
	Skill     *skill.Conf
	Item      *item.Conf
	Troop     *troop.Conf
	Gacha     *gacha.Conf
	Exchange  *exchange.Conf
	Guard     *guard.Conf
	Resource  *resource.Conf
	Review    *review.Conf
	Soldier   *soldier.Conf
	Building  *building.Conf
	Formation *formation.Conf
}

// Version 全局配置版本（内嵌基线=1；JSON 加载/热更每次成功 +1）。
func (c *GameConf) Version() uint64 { return c.globalVersion }

// TableVersions 内容版本（gameconfig.json → 内容 hash；内嵌为空表），返回只读副本。
func (c *GameConf) TableVersions() map[string]string {
	cp := make(map[string]string, len(c.tableVersions))
	for k, v := range c.tableVersions {
		cp[k] = v
	}
	return cp
}

// FilePath config_path（"" = 内嵌）。
func (c *GameConf) FilePath() string { return c.filePath }

// All 原始配置表（pb.Table，全表统一数据源；battle 子集同样含全表数据）。nil = 未加载。
func (c *GameConf) All() *pb_gameconfig.Table { return c.pb }
