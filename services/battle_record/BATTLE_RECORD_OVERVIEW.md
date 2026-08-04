# Battle Record — 战斗记录服务

> 路径: `services/battle_record`  
> 模块: `server.slg.com/services/battle_record`

---

## 概述

`battle_record` 是 SLG 服务端的**战斗记录节点**，持久化战报（纯 MySQL），提供 `SaveBattleRecord` / `GetBattleRecord` / `ListBattleRecords` RPC。支持**角色 / 联盟 / 地块**三维查询，战报保留 **14 天**（节点内定时清理）。

与 `game`/`worldmap`/`battle` 按 `instance` **单例配对**：
- **worldmap** 战斗结算后调用 `SaveBattleRecord` 落库
- **game** 玩家查询战报时调用 `ListBattleRecords`

---

## 目录结构

```
battle_record/
├── main.go                                    # 入口：DB Init + AutoMigrate + ETCD + 注册 + 14天清理 goroutine
│
├── battle_record_models/                       # GORM 模型
│   ├── battle.record.go                       # 主表（只存记录，含 BattleResults 二进制）
│   └── battle.record.tag.go                   # 查询索引表（单复合索引覆盖三维）
│
├── battle_handlers/
│   └── battle_servers/                        # BattleRecordHandler（Save/Get/List）
│       ├── battle.record.server.st.go        # BattleRecordServer + SetStore
│       ├── battle.record.server.func.go      # SaveBattleRecord/GetBattleRecord/ListBattleRecords
│       └── battle.record.server_test.go      # bufconn RPC 冒烟（sqlite）
│
└── battle_internals/
    └── battle_records/                         # 存储逻辑
        ├── store.func.go                      # 事务写 / 分页查 / 14天清理 / 编解码
        └── store_test.go                      # 存储单测（sqlite）
```

---

## 核心设计

### 1. 存储：纯 MySQL + 两张表（避免主表多索引放大）

用户担忧"三维 × 攻守 = 6+ 索引"写入放大。方案：**主表只存记录，查询走独立标签索引表**。

**`battle_record` 主表**：`march_id/march_type/map_id/attacker_role_id/attacker_union_id/defender_role_ids(json)/defender_union_ids(json)/attacker_win/is_occupied/building_damage/results(blob: proto.Marshal(BattleResults))/battle_time`。
仅 PK + `battle_time` 索引（清理用）。

**`battle_record_tag` 索引表**：`tag_type(1=role 2=union 3=tile)/tag_id/battle_record_id/battle_time`。
**单复合索引 `(tag_type, tag_id, battle_time)`** 覆盖所有维度查询；另加 `battle_time` 单索引供清理。

一条战报 → 主表 1 行 + tag 表 N 行（攻方 role/union + 每个防守方 role/union + tile，去重）。

### 2. 查询流程

```
ListBattleRecords(tag_type, tag_id, page, page_size)
  ├─ SELECT COUNT(*) FROM battle_record_tag WHERE tag_type=? AND tag_id=?          → total
  ├─ SELECT battle_record_id FROM battle_record_tag WHERE tag_type=? AND tag_id=?
  │     ORDER BY battle_time DESC LIMIT offset, size                               → ids（走复合索引）
  └─ SELECT * FROM battle_record WHERE id IN (ids)                                 → 按 tag 顺序重排返回
```

### 3. 14 天保留

节点内 `time.Ticker` 每小时 goroutine（监听全局 ctx），`DELETE ... WHERE battle_time < now-14d`（两张表）。

### 4. 主战报 + 子战报（parent_id 关联）

`battle_record.parent_id` 指向主战报（0 = 主战报）：
- **主战报**（parent_id=0）：生成角色/联盟/地块 tag → 玩家列表一条
- **子战报**（parent_id≠0）：**不生成 tag**，只通过主战报进入，避免玩家列表重复
- 查询：`ListBattleRecordChildren(record_id)` 分页取主战报的子战报

> ⚠️ **存储能力已实现，车轮战编排未实现**：当前每次结算仍存独立主战报（parent_id=0）。
> 车轮战（一串 n 队连续进攻）需：先建主战报拿 id → 透传 parent_id 到每次结算的 SaveBattleRecord。见 `worldmap_inits/engine.func.go` 的 saveBattleRecord TODO。

### 5. RPC 服务（BattleRecordHandler）

| RPC | 说明 |
|-----|------|
| `SaveBattleRecord` | worldmap 战斗结算后落库（含完整 BattleResults + 攻守角色/联盟） |
| `GetBattleRecord` | 按战报 ID 查询详情 |
| `ListBattleRecords` | 按 tag（角色/联盟/地块）分页查询，battle_time 倒序 |
| `ListBattleRecordChildren` | 查询主战报的子战报（车轮战 n 队整合） |

---

## 调用链

```
worldmap（战斗结算后）
  └─ Engine.settleBattle → 异步 saveBattleRecord（fire-and-forget，不阻塞战斗 tick）
       └─ battle_record.SaveBattleRecord RPC → 事务写主表 + tag 表

game（玩家查询）
  └─ HandlerBattleRecordList（GameBattleRecordList=1000013）
       └─ game_rpc_clients.BattleRecord().ListBattleRecords
            └─ battle_record.ListBattleRecords RPC → tag 分页查询

ETCD 注册：/node:service:battle_record/{instance}/
发现：worldmap / game 通过 rpc_handlers hub → GetBattleRecordHandlerClient
```

- **保存方**：worldmap（`Engine.settleBattle` 拿到 rsp 后异步保存）
- **查询方**：game（玩家查本角色战报；联盟/地块战报由客户端指定 tag）

---

## 外部依赖

- `api/protocol/pb/pb_battle_record` — 协议定义
- `api/protocol/pb/pb_battle` — BattleResults（战报内容）
- `common/conns/dbconn` — MySQL（复用 `DB.Game` 库）
- `common/utils/snowflakes` — 战报主键雪花 ID
- `common/conns/etcdconn` — ETCD 服务注册
- `common/servers` — 生命周期框架
