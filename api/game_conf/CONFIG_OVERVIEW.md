# 配置表系统管线

> 入口文档：数据源 → 导出 → 运行时加载
> 模块: `server.slg.com/api/game_conf`

---

## 概述

配置系统采用 **xlsx 源表 + tabtoy + protobuf** 管线（对齐 LDL）：策划在 Excel 维护 24 张数据表，
`export_conf.ps1` 一键导出为**单一 `gameconfig.json`**（tabtoy）+ **自动生成 `gameconfig.proto`**
（protoc 编译为强类型 `pb_gameconfig.pb.go`），运行时读取单一 JSON → `pb.Table` →
各域 `NewFromPB` 构建 typed Conf 索引。

数据源**唯一**：不再有 13 个分散 per-table JSON，也没有 Go 内嵌占位/模拟数据。
运行时兜底（`InitDefault` / 各域 `New()`）通过 `go:embed` 嵌入同一份 `gameconfig.json`，
与文件加载同源同构，无双份数据漂移。

---

## 数据流

```
策划维护                          export_conf.ps1                      运行时
────────                          ─────────────                         ──────
excel/*.xlsx ──tabtoy(v3)──▶ json/gameconfig.json ──▶ 文件加载 Init/ReLoad
  24 张数据表                    api/protocol/src/gameconfig.proto      │
  + Index.xlsx                   (+go_package 注入)                     ▼
  + game_attribute.xlsx     ─protoc(build_s.ps1)─▶ pb_gameconfig.pb.go ─┴─▶ pb.Table ──▶ NewFromPB ──▶ typed Conf
  + game_enumeration.xlsx          （强类型）          （单一数据源）     （go:embed 内嵌 → InitDefault/New() 兜底）
```

### 导出命令

```powershell
pwsh -File api/game_conf/scripts/export_conf.ps1
```

产物：
- `api/game_conf/json/gameconfig.json` — tabtoy 单一 JSON（go:embed 源）
- `api/protocol/src/gameconfig.proto` — 生成 + `option go_package` 注入
- `api/protocol/pb/pb_gameconfig/gameconfig.pb.go` — protoc 编译（强类型）

---

## 目录结构

```
api/game_conf/
├── CONFIG_OVERVIEW.md             # 本文档（管线入口）
├── excel/                         # 策划源表（xlsx，唯一编辑入口）
│   ├── Index.xlsx                 #   表注册（tabtoy 入口）
│   ├── game_attribute.xlsx        #   字段注册（中文标识名 → 英文字段名 → 类型）
│   ├── game_enumeration.xlsx      #   枚举定义（按声明顺序自动编号 0,1,2…）
│   └── 24 张数据表 xlsx
├── executor/                      # 导出工具
│   ├── tabtoy.exe                 #   LDL 同源导出器
│   ├── gameconfig.py              #   proto 注入 go_package（tabtoy 重生成会抹掉，必须紧随其后）
│   ├── gameconfig_schema.py       #   表结构 spec（单一事实源：TABLES/STRUCTS/ENUMS）
│   └── data_overrides.py          #   数据修正说明（item 1002/1003、building 103 悬空引用修复）
├── scripts/export_conf.ps1        # 一键导出（tabtoy → gameconfig.py → build_s.ps1）
├── json/
│   ├── gameconfig.json            # tabtoy 单一产物
│   └── gameconfig.embed.go        # go:embed + Table()/Build[T] 惰性解析
├── table/table.func.go            # ContentHash（FNV-32a，热更内容去重）
├── conf.loader.func.go            # 加载器：单一 JSON → pb.Table → NewFromPB + 跨表校验
├── conf.table.st.go               # GameConf 聚合（pb 字段 + All() + 各域 typed Conf）
├── conf.table.vfunc.go            # Init/InitDefault/InitBattle/New/Load/ReLoad/nextVersion
├── conf.watch.func.go             # StartWatch(ctx) 热更监听（mtime+size 轮询）
└── <域>/<域>.conf.st.go           # 各域 typed Conf（公开方法稳定）
    <域>/<域>.conf.func.go         #   校验 Validate() 等
    <域>/<域>.conf.pb.func.go      #   NewFromPB + New() 兜底
```

---

## 表结构（24 张数据表）

| 数据表 | 行数语义 | 索引键 | 域构造器读取 |
|---|---|---|---|
| battle | single | – | `pb.Battle[0]` |
| formation | single | – | `pb.Formation[0]` |
| troop | single | – | `pb.Troop[0]`（transform_cost = repeated cost）|
| hero | single | – | `pb.Hero[0]`（awaken_cost = repeated cost）|
| hero_exp | multi | level | `pb.HeroExp` → ExpNeed（100 级）|
| hero_attr | multi | conf_id | `pb.HeroAttr` → heroes map（base/growth 摊平）|
| item | multi | conf_id | `pb.Item` → configs map |
| exchange | single | – | `pb.Exchange[0]` |
| resource | multi | level | `pb.Resource`（9 级）|
| gacha_pool | multi | pool_id | `pb.GachaPool` |
| gacha_drop_group | multi | pool_id+group_id | `pb.GachaDropGroup` |
| gacha_drop_item | multi | – | `pb.GachaDropItem` |
| guard | single | – | `pb.Guard[0]` |
| guard_config | multi | level | `pb.GuardConfig` |
| guard_slot | multi | – | `pb.GuardSlot` |
| soldier | single | – | `pb.Soldier[0]` |
| soldier_hero_cap | multi | level | `pb.SoldierHeroCap` |
| soldier_barrack_bonus | multi | level | `pb.SoldierBarrackBonus` |
| review | single | – | `pb.Review[0]` |
| review_level | multi | level | `pb.ReviewLevel`（rewards = repeated reward）|
| skill | multi | conf_id | `pb.Skill`（upgrade_cost = 单 cost）|
| skill_collection | multi | skill_conf_id | `pb.SkillCollection`（need_heroes = repeated cost）|
| skill_setting | single | – | `pb.SkillSetting[0]`（槽位标量）|
| building | multi | type | `pb.Building`（cost/queue_nums 为数组）|

共享结构体（`game_attribute` 行组，不进 Index）：`cost`、`reward`、`level_num`。
枚举（`game_enumeration`）：`skilltype`、`targettype`、`effecttype`、`itemeffecttype`、`resourcetype`。
完整字段 spec 见 `executor/gameconfig_schema.py`（单一事实源）。

---

## 运行时加载语义

| 入口 | 行为 |
|---|---|
| `Init(configPath)` | 读 config_path 单一 gameconfig.json → 全量构建 + 跨表校验；**失败返回 err 且保持旧配置**（fail-fast，无「缺表保持内嵌」） |
| `InitDefault()` | config_path 为空时，用 go:embed 内嵌 gameconfig.json 构建（同源同构） |
| `InitBattle()` | battle 节点子集：仅加载 battle+skill，其余 nil（config_path 空走内嵌子集） |
| `Load()` | 全局配置（atomic.Pointer 快照，**永不返回 nil**） |
| `ReLoad()` | 按当前 filePath 重载；内容 hash 全同跳过；失败保持旧配置 |
| `StartWatch(ctx, interval)` | mtime+size 轮询，变化 → ReLoad()；**必须接收全局 ctx**（见 CLAUDE.md 全局 Context 规范） |

构建链路：`loadFromPath` → `util_jsons.Unmarshal`（jsoniter 忽略 `@Tool`/`@Version`）→ `pb.Table`
→ `newFromPB`（各域 `NewFromPB`「局部构建 + 末尾提交 + Validate」）→ `validateCrossRefs`（跨表引用）。

### 各域 NewFromPB 模式

```go
// <域>.conf.pb.func.go
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
    // …从 pb 表行构建 typed Conf
    if err := c.Validate(); err != nil { return nil, fmt.Errorf("<域> validate: %w", err) }
    return c, nil
}
// New() 兜底：gameconfigjson.Build(NewFromPB)（内嵌损坏 panic，正常不应发生）
```

---

## 如何加数据 / 改数据

1. 编辑 `api/game_conf/excel/*.xlsx`（策划直接维护；新增英雄在 hero_attr 加行，新增池在 gacha_* 加行，改经验曲线在 hero_exp 改行）。
2. 运行 `pwsh -File api/game_conf/scripts/export_conf.ps1` 重新导出。
3. `go build ./...` + `go test ./api/game_conf/...` 验证（校验失败会在构建期报错）。
4. 运行时改 `gameconfig.json` 触发 `StartWatch` 热更（校验失败自动回滚保持旧配置）。

> ⚠️ 表结构变更（加字段/加表）需要同步改 `executor/gameconfig_schema.py` 与对应的 `NewFromPB`。
