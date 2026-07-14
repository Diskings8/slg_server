# Cores — SLG 服务端核心功能模块

> 生成日期: 2026-07-13  
> 路径: `services/internal/cores`  
> 模块: `server.slg.com/services/internal/cores`

## 概述

`cores` 是 SLG（策略类游戏）服务端的**核心逻辑包**，负责地图数据管理、行军系统、AOI（Area of Interest）视野管理、Tick 定时调度、角色连接管理、角色数据轮询、出生块管理等基础游戏机制。它是整个游戏服务器中支撑地图玩法的心脏模块。

---

## 包结构

```
cores/
├── cores_declarations/    # 核心类型与接口声明
├── map_aois/              # AOI 视野管理
├── map_blocks/            # 地图区块划分（骨架）
├── map_borns/             # 出生块管理
├── map_connects/          # 角色连接与会话管理
├── map_datas/             # 地图格子数据
│   ├── map_buildings/     # 建筑覆盖层
│   └── map_events/        # 事件覆盖层
├── map_managers/          # 地图管理器（核心调度引擎）
├── marchdos/              # 行军执行器
├── marchs/                # 行军信息与队伍
└── roles/                 # 角色数据管理
```

---

## 包文档索引

| 包 | 详细文档 | 状态 |
|---|---|---|
| `cores_declarations` | [docs/cores_declarations.md](docs/cores_declarations.md) | ✅ 完成 |
| `map_aois` | [docs/map_aois.md](docs/map_aois.md) | ✅ 完成 |
| `map_blocks` | [docs/map_blocks.md](docs/map_blocks.md) | ⬜ 骨架 |
| `map_borns` | [docs/map_borns.md](docs/map_borns.md) | ✅ 完成 |
| `map_connects` | [docs/map_connects.md](docs/map_connects.md) | ✅ 完成 |
| `map_datas` | [docs/map_datas.md](docs/map_datas.md) | ✅ 核心逻辑完成 |
| `map_managers` | [docs/map_managers.md](docs/map_managers.md) | ✅ 核心调度完成 |
| `marchs` | [docs/marchs.md](docs/marchs.md) | ✅ 核心逻辑完成 |
| `marchdos` | [docs/marchdos.md](docs/marchdos.md) | ✅ 完成 |
| `roles` | [docs/roles.md](docs/roles.md) | ✅ 完成 |

---

## 核心架构关系图

```
                                         ┌─────────────────────────────────┐
                                         │          MapManager              │
                                         │   (核心调度引擎)                   │
                                         │                                  │
                                         │  ┌─ loopTickCheck() (100ms定时)   │
                                         │  │   ├─ 行军到期处理              │
                                         │  │   ├─ upMapAsync() 地图推送     │
                                         │  │   └─ 兜底清理(1s)              │
                                         │  │                                │
                                         │  └─ loopTickAccept()              │
                                         │      └─ TickerChan ← MarchInfo    │
                                         └──────┬───────────────────────────┘
                                                │
        ┌───────────┬───────────┬───────────┬───┼──────┬───────────┬───────────┐
        │           │           │           │   │      │           │           │
        ▼           ▼           ▼           ▼   │      ▼           ▼           ▼
 ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────┐│┌──────────┐ ┌──────────┐ ┌──────────┐
 │MapData   │ │MarchInfo │ │ MarchDo  │ │Role  │││Roles     │ │Map       │ │Map       │
 │Manager   │ │Manager   │ │Handle    │ │Conn  │││(角色数据) │ │Connects  │ │Blocks    │
 │          │ │          │ │工厂       │ │Mgr   │││          │ │管理器    │ │(骨架)    │
 │ MapInfo[]│ │ allMarch │ │ ┌─Single │ │AOI   │││Poller    │ │gRPC流   │ │          │
 │ AOI      │ │ Ticker   │ │ ├─Multi  │ │推送  │││Copy-On-  │ │推送      │ │          │
 │ Save队列 │ │Chan      │ │ └─Base   │ │系统  │││Write     │ │系统      │ │          │
 │ BornAts  │ │Assemble  │ │          │ │      │││Queue     │ │          │ │          │
 └─────┬────┘ └─────┬────┘ └──────────┘ └──────┘│└──────────┘ └──────────┘ └──────────┘
       │            │                           │
       ▼            ▼                           ▼
 ┌──────────┐ ┌──────────┐             ┌──────────────┐
 │ MapInfo  │ │ MarchInfo│             │  Loader      │
 │ (单格)   │ │ (行军)   │             │  DB CRUD     │
 │          │ │          │             │  Cache       │
 │坐标/等级/│ │ 起止点/  │             └──────────────┘
 │归属/建筑 │ │ 时间/队伍│
 │ 事件     │ │ 路径/AOI │
 └──────────┘ │ 战斗/   │
              │ 状态标记 │
              └────┬─────┘
                   │
                   ▼
             ┌──────────┐
             │   Team   │
             │ 行军队伍  │
             │          │
             │武将+士兵  │
             │存活/战斗  │
             └──────────┘
```

---

## 关键设计模式

| 模式 | 使用位置 | 说明 |
|---|---|---|
| **函数式选项** | `MapManager` 配置 | `WithStopChan`, `WithCutNum` 等 |
| **策略模式** | `MapManager.marchDoFunc/Handle` | 行军到达处理与核心调度解耦 |
| **模板方法** | `BaseMarch.Do()` | prepare → do → finish 三步固定流程 |
| **分层并发控制** | `SingleMarch` / `MultiMarch` | 行军锁→来源地图锁→目标地图锁，失败回滚 |
| **写时复制 (COW)** | `roles.Data.Copy()` | 读操作走源数据，写操作触发热拷贝 |
| **对象池** | `MapPBGet/MapPBPut` | `sync.Pool` 复用 `pb_camera.MapInfo` |

---

## 当前状态总览

| 组件 | 状态 |
|---|---|
| `cores_declarations` | ✅ 基本完成 |
| `map_aois` | ✅ 功能完善 |
| `map_blocks` | ⬜ 骨架实现 |
| `map_borns` | ✅ 功能完善 |
| `map_connects` | ✅ 功能完善 |
| `map_datas` | ✅ 核心逻辑完成（`Clear`/`SetHall` panic） |
| `map_buildings` | ✅ 建筑血量/等级/战斗逻辑完成 |
| `map_events` | ⬜ 空结构占位 |
| `map_managers` | ✅ 核心调度完成（`clearMapFunc`/`FormatMapInfo2Pb` 空） |
| `marchs` | ✅ 核心逻辑完成（`GetRelocationVal`/`TableName` panic） |
| `marchdos` | ✅ 模板方法 + 分层加锁完成 |
| `roles` | ✅ 数据管理层完成（`Save` panic） |
| 组合行军 `allAssembleMarch` | ⬜ 集合已定义，逻辑未实现 |
| 数据库持久化 | ✅ AutoMigrate + Find 完成，`SaveDo` 空 |

---

## 外部依赖

- `server.slg.com/common/utils/hashmaps` — 自定义并发哈希表
- `server.slg.com/common/utils/maths` — 数值计算
- `server.slg.com/common/utils/s2s` — 切片辅助
- `server.slg.com/common/pollers` — 数据轮询管理器
- `server.slg.com/common/conns/dbconn` — 数据库连接
- `server.slg.com/common/conns/rpcconn/rpc_streams` — gRPC 流管理
- `github.com/go4org/hashtriemap` — 并发哈希 Trie 图
- `github.com/patrickmn/go-cache` — 内存缓存
- `go.uber.org/zap` — 日志
- GORM — ORM 框架
- gRPC + protobuf — 网络通信
