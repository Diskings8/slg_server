package game_conf

import "server.slg.com/api/game_conf/table"

// tableReg 配置表注册项：JSON 文件名 + 从 GameConf 取回对应域 Conf 的访问器。
type tableReg struct {
	file string                 // JSON 文件名（不含扩展名）
	get  func(gc *GameConf) table.Table
}

// allTables 全量表注册中心（顺序 = 加载顺序）。
//
// 随迁移阶段逐个加入：gacha 依赖 hero/item 跨表校验故放最后。
// 未注册的表不参与 JSON 加载（保持 Go 内嵌占位）。
var allTables = []tableReg{
	{"hero", func(gc *GameConf) table.Table { return gc.Hero }},
	{"skill", func(gc *GameConf) table.Table { return gc.Skill }},
	{"item", func(gc *GameConf) table.Table { return gc.Item }},
	{"troop", func(gc *GameConf) table.Table { return gc.Troop }},
	{"exchange", func(gc *GameConf) table.Table { return gc.Exchange }},
	{"battle", func(gc *GameConf) table.Table { return gc.Battle }},
	{"gacha", func(gc *GameConf) table.Table { return gc.Gacha }},
	{"guard", func(gc *GameConf) table.Table { return gc.Guard }},
	{"resource", func(gc *GameConf) table.Table { return gc.Resource }},
	{"soldier", func(gc *GameConf) table.Table { return gc.Soldier }},
	{"building", func(gc *GameConf) table.Table { return gc.Building }},
}

// battleTables battle 节点子集（battle 规则 + 技能表）。
var battleTables = []tableReg{
	{"battle", func(gc *GameConf) table.Table { return gc.Battle }},
	{"skill", func(gc *GameConf) table.Table { return gc.Skill }},
}
