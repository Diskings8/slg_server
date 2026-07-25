# game 服务架构文档

> 路径: `services/game/` · 模块: `server.slg.com/services/game`
>
> 本文以 `server/services/game`（成熟参考实现）为蓝本，分析当前 game 服务的架构设计与实现状态。

---

## 参考基准

| 项目 | 路径 |
|------|------|
| **参考实现（成熟）** | `E:\gamewrokspace\server\services\game` |
| **本项目（当前）** | `E:\gamewrokspace\ai4slg\slg_server\services\game` |

参考实现是一个完整的 SLG 游戏区服服务，包含 entity 缓存层、model/query 数据层、handler 协议层、internal 业务逻辑层等 8 个目录约 200+ 文件。当前项目处于早期建设阶段，目录结构和功能范围为参考实现的子集。

---

## 目录结构对比

```
参考实现 (server/services/game)         当前项目 (ai4slg/.../services/game)
├── main.go                              ├── main.go
├── entity/          实体+缓存层           │
│   ├── role/         角色聚合根           │
│   ├── system*/      系统实体            │
│   └── union/        联盟实体            │
├── model/            GORM 数据模型       ├── game_models/
│   └── role_*.go     各子系统表模型       │   └── role_model/
│                                          │       ├── db_role_model/   DB 模型
│                                          │       └── std_role_model/ 逻辑模型
├── query/            gorm gen DAO        │
│   └── gen.go        42 个查询对象       │
├── handler/          gRPC + Stream       ├── game_servers/
│   ├── game/          RPC 处理器         │   ├── game_handlers/    RPC 处理器
│   └── stream/       Stream 处理器       │   └── game_streams/    Stream 处理器
├── internal/         核心业务逻辑         (实现在 services/internal/cores/)
│   ├── activity/*    活动系统             │
│   ├── arena/        竞技场               │
│   ├── battle/       战斗                 │
│   ├── heroes/       英雄                 │
│   ├── build/        城建                 │
│   ├── shop/         商店                 │
│   ├── union/        联盟                 │
│   ├── ...           40+ 业务模块         │
│── generate/         代码生成器           │
└── rpcclient/        gRPC 客户端          │
```

---

## 架构层次映射

```
参考实现 (server)                      当前项目 (ai4slg)
─────────────────                     ─────────────────
handler/game/      ←  gRPC 入口  →    game_servers/game_handlers/
handler/stream/    ← Stream 入口  →    game_servers/game_streams/
entity/            ← 实体/缓存  →    game_models/role_model/std_role_model/ (雏形)
model/             ← 数据模型  →    game_models/role_model/db_role_model/
query/             ← DAO 层    →    (待实现，暂无 gen DAO)
internal/          ← 业务逻辑  →    services/internal/cores/ (独立位置)
rpcclient/         ← RPC 客户端 →   (待实现，暂无 rpcclient)
generate/          ← 代码生成  →    (待实现，暂无 generate)
```

---

## 目录职责说明

### 1. `main.go` — 服务入口

**参考实现** 包含完整的生命周期管理：config → 日志 → DB → 游戏配置 → Snowflake → gRPC 注册（20+ 服务）→ ETCD 服务发现（异步 init → 同步 init → shutdown 回调链）。

**当前项目** 为简化版本：
- 解析 `env` + `instance` 参数
- 初始化配置 → 日志 → ETCD 注册
- 构建 gRPC Server，注册 **2 个服务**（`StreamServer` + `HandlerServer`）
- 信号监听优雅关闭

```
参考实现注册的 gRPC 服务 (17+)          当前项目注册的服务 (2)
GameStream             ← Stream →      game_streams.StreamServer
GameService / Trade / Union /          game_handlers.HandlerServer
Article / Battle / Cross / SDK /
Radar / Privacy / Trial / Platform /
Zombie / Mail / CityCompetition / Rank
```

### 2. `game_servers/` — 服务器层（对标参考的 `handler/`）

#### `game_streams/`（对标 `handler/stream/` + `handler/role_stream.go`）

| 文件 | 职责 | 参考实现对应 |
|------|------|-------------|
| `stream.game.st.go` | `StreamServer` 结构体，实现 gRPC Stream 服务接口 | `handler/role_stream.go` (RoleStreamHandler) |
| `stream.game.func.go` | 消息处理函数，路由客户端实时消息 | `handler/stream/*` (约 40 个处理器) |

**参考实现** 的 stream 层按业务领域拆分为约 40 个子包（role/hero/item/build/shop/union 等），消息通过 `protoMap` 协议注册表分发。当前项目为早期阶段，仅骨架结构，尚未拆分领域处理器。

#### `game_handlers/`（对标 `handler/game/`）

| 文件 | 职责 | 参考实现对应 |
|------|------|-------------|
| `handler.game.st.go` | `HandlerServer` 结构体，实现 gRPC Unary 服务 | `handler/game/*` (各 RPC 处理器) |
| `handler.game.func.go` | RPC 请求处理函数 | `handler/game/role.go` 等 |

**参考实现** 包含 15+ 个 RPC 服务处理器（role/trade/union/article/battle/cross/sdk/radar/privacy 等），当前项目为统一 `HandlerServer` 聚合入口。

### 3. `game_models/` — 数据模型层（对标参考的 `model/` + 部分 `entity/`）

**参考实现** 采用实体/模型分离：
- `entity/` — 运行时内存对象（带缓存、脏标记、异步保存），是业务逻辑的主要操作对象
- `model/` — GORM 表结构映射（`*.gen.go` 自动生成，约 40 个模型）

**当前项目** 同样采用分层但更简洁：

```
game_models/role_model/
├── db_role_model/       DB 模型       → 对标参考的 model/  (GORM struct)
│   └── std.db.role.go   RoleDb        GORM 模型，映射 Role 表
└── std_role_model/      逻辑模型      → 对标参考的 entity/role/ (运行时对象)
    └── std.role.go      Role          实现 IHandler，带 dirty 标记
```

| | 参考实现 `model/` | 当前项目 `db_role_model/` |
|--|--|--|
| 生成方式 | gorm.io/gen 自动生成 | 手写 |
| 模型数量 | ~40 个 | 1 个（RoleDb） |
| 字段完整度 | 各子系统完整字段 | 仅 Id |

| | 参考实现 `entity/` | 当前项目 `std_role_model/` |
|--|--|--|
| 接口 | 自定义实体接口 | IHandler (Poller) |
| 脏标记 | 有 | 有 (`dirty atomic.Bool`) |
| 持久化 | 完整实现 | SaveCache/SaveDB 为 panic (待实现) |
| 聚合范围 | 属性/英雄/物品/装备/建筑/任务 等 20+ 子系统 | 仅 Id |

### 4. `game_globals/` — 全局数据

参考实现中的全局数据分散在：
- `entity/unionglobal/` — 全局联盟
- `entity/systemactivity/` — 系统活动
- `internal/global/` — 全局管理器（王国/排行/Boss/跨服）

当前项目以 `db_global_games/` 作为全局数据的统一入口，当前为占位状态。

| 文件 | 说明 |
|------|------|
| `global.db.var.go` | `GGameDbC` 全局数据库连接变量 |
| `global.db.func.go` | 全局数据操作函数包（空） |

### 5. 核心业务逻辑 — `services/internal/cores/`

**与参考实现的关键差异**：当前项目的核心业务逻辑不放在 `services/game/internal/`，而是独立在 `services/internal/cores/`（同一仓库但不同路径）。

| 领域 | 参考实现 `internal/` | 当前项目 `cores/` |
|------|---------------------|-------------------|
| 地图 | 无独立模块 | ✅ `map_datas/map_managers/map_aois/map_connects/map_borns/map_blocks` |
| 行军 | 无独立模块 | ✅ `marchs/marchdos` |
| 角色 | 20+ 子系统 | ✅ `roles` (基础数据管理) |
| 战斗 | `battle/` | ⬜ 待实现 |
| 建筑 | `build/` | ⬜ 待实现 |
| 英雄 | `heroes/` | ⬜ 待实现 |
| 物品 | `item/` | ⬜ 待实现 |
| 装备 | `equips/` | ⬜ 待实现 |
| 科技 | `tech/` | ⬜ 待实现 |
| 任务 | `task/` | ⬜ 待实现 |
| 商店 | `shop/` | ⬜ 待实现 |
| 联盟 | `union/` | ⬜ 待实现 |
| 活动 | `activity/*` (17种) | ⬜ 待实现 |
| 邮件 | `mail/*` | ⬜ 待实现 |
| 竞技场 | `arena/` | ⬜ 待实现 |

---

## 数据流

### 实时消息路径（Stream）

```
客户端 → Gate → gRPC Stream
    → game_streams/StreamServer         (消息入口)
        → services/internal/cores/*     (业务逻辑)
            → game_models/*             (数据持久化)
                → MySQL/Redis
```

### RPC 调用路径（Unary）

```
微服务 → gRPC
    → game_handlers/HandlerServer       (RPC 入口)
        → services/internal/cores/*     (业务逻辑)
```

---

## 当前实现状态总览

| 层次 | 组件 | 状态 | 说明 |
|------|------|------|------|
| 入口 | `main.go` | ✅ 完成 | ETCD 注册 + gRPC 服务 + 优雅关闭 |
| 协议层 | `game_streams/` | ⚠️ 骨架 | 仅入口结构，未拆分领域处理器 |
| 协议层 | `game_handlers/` | ⚠️ 骨架 | 统一入口，未拆分具体 RPC |
| 模型层 | `db_role_model/` | ✅ 基础 | GORM 模型定义完成 |
| 模型层 | `std_role_model/` | ⚠️ 部分 | 逻辑模型框架完成，持久化待实现 |
| 全局层 | `db_global_games/` | ⬜ 占位 | 变量声明，函数为空 |
| DAO 层 | `query/` | ❌ 缺失 | 无 gorm gen 生成的查询层 |
| RPC 客户端 | `rpcclient/` | ❌ 缺失 | 无跨服务调用封装 |
| 代码生成 | `generate/` | ❌ 缺失 | 无协议/模型生成器 |
| 核心逻辑 | `internal/cores/` | 🟡 开发中 | 地图/AOI/行军/角色基础完成，战斗/城建/英雄/活动等缺失 |

---

## 与参考实现的差距分析

### 已覆盖的架构理念

| 理念 | 参考实现 | 当前项目 |
|------|---------|---------|
| gRPC Stream + Unary 双入口 | ✅ | ✅ |
| 实体/模型分离 | entity vs model | std_role_model vs db_role_model |
| 脏标记异步保存 | entity 内置 | std_role_model 内置 |
| ETCD 服务注册 | ✅ | ✅ |
| 面向接口编程 | ✅ | IHandler 接口 |

### 主要差距

1. **业务逻辑位置不同** — 参考实现将业务逻辑内聚在 `services/game/internal/`，当前项目放在 `services/internal/cores/`
2. **DAO 层缺失** — 参考实现用 gorm.io/gen 生成类型安全的 query 层，当前项目暂无
3. **协议注册表** — 参考实现通过 `protoMap` + `protocol_gen.go` 自动管理协议号与处理函数的映射，当前项目未实现
4. **领域覆盖** — 参考实现了 40+ 业务模块，当前 cores 仅覆盖地图/行军/角色基础，战斗/英雄/城建/联盟/活动等均待实现
5. **代码生成** — 参考实现通过 `generate/` 自动化协议注册和模型代码，当前项目手写
6. **跨服通信** — 参考实现有 `rpcclient/` 封装对其他微服务的 gRPC 调用，当前项目未实现

---

## 命名规范

当前项目遵循项目级命名规范 `{领域}.{概念}.{类型后缀}.go`：

| 后缀 | 含义 | 示例 |
|------|------|------|
| `.st.go` | Struct/Type 结构体定义 | `stream.game.st.go` |
| `.func.go` | Function 方法/逻辑实现 | `stream.game.func.go` |
| `.var.go` | Variable 包级变量 | `global.db.var.go` |
| `.db.func.go` | 数据库操作 | `global.db.func.go` |

> 参考实现使用 `snake_case` 包名分层（`handler/stream/role`、`internal/heroes`），当前项目使用 `_` 连接的大包名（`game_servers`、`game_models`）。

---

## 启动流程

```
main()
  ├── parseFlagVar()                    — 解析 env/instance
  ├── configs.LoadEnvConf()             — 加载环境配置
  ├── loggers.Init()                    — 日志初始化
  ├── etcdconn.RegisterServiceByNodeType() — ETCD 注册 (NodeGameService)
  ├── servers.BuildRpcServer()          — 构建 gRPC Server
  ├── RegisterServices(                 — 注册服务
  │     &game_streams.StreamServer{},
  │     &game_handlers.HandlerServer{},
  │   )
  ├── srv.Run()                         — 启动 gRPC 监听
  └── signal.Notify()                   — 等待关闭信号
```

---

## 相关文档

| 文档 | 路径 |
|------|------|
| 项目总览 | `CLAUDE.md` |
| 核心逻辑架构 | `services/internal/cores/CORES_OVERVIEW.md` |
| 命名规范 | `docs/naming-convention.md` |
| 参考实现 | `E:\gamewrokspace\server\services\game\main.go` |
