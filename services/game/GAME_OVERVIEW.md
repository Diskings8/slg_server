# Game — SLG 游戏业务层

> 生成日期: 2026-07-30  
> 路径: `services/game`  
> 模块: `server.slg.com/services/game`

---

## 概述

`game` 是 SLG 服务端的**游戏业务层**，负责角色数据管理、道具背包、英雄养成、协议路由等业务逻辑。与 `services/internal/cores`（地图引擎层）配合，构成完整的游戏服务。

---

## 目录结构

```
game/
├── main.go                              # 入口：gRPC 服务启动 + 生命周期
│
├── game_entitys/                        # 实体层（数据 + 内存索引）
│   └── game_roles/                      # Role 聚合根
│       ├── game.role.st.go             # Role 结构体（挂载所有子模块）
│       ├── game.role.db.go             # DB CRUD（反射遍历子模块）
│       ├── game.role.copy.go           # Copy-on-Write 实现
│       ├── game.role.logic.go          # 角色业务逻辑（CheckItem/AddItem/ReduceItem）
│       ├── game.role.poller.go         # Poller 管理器初始化
│       ├── role_items/                  # 道具背包子模块
│       │   ├── item.st.go             # 结构体
│       │   ├── item.func.go           # Init/Copy/Format2Pb/AddItem/ReduceItem
│       │   ├── item.func.gen.go       # Getter/Setter
│       │   └── item.db.go             # DB CRUD
│       ├── role_heroes/                # 英雄子模块
│       ├── hero_skills/                # 技能子模块
│       ├── hero_skillcollections/      # 技能收藏子模块
│       └── cultivate_costs/            # 养成消耗子模块
│
├── game_handlers/                       # 协议路由层
│   ├── game_servers/                   # Unary RPC 处理器
│   │   ├── game.server.st.go          # GameServer 结构体 + gRPC 注册
│   │   └── game.server.func.go        # Do() — 统一协议入口(NodePacket → 路由)
│   ├── game_streams/                   # Stream 处理器
│   │   ├── game.stream.st.go          # GameStream 结构体 + Stream() + gateConnectDo()
│   │   └── game.server.func.go        # Recv() — 流消息路由 + Offline()
│   ├── game.protocol.registry.go       # 协议注册表(RegisterProto/GetProtoHandler/Wrap)
│   └── game.protocol.gen.go            # init() 注册协议号 ↔ 处理器映射
│
├── game_logics/                         # 业务逻辑层
│   ├── game.item.logic.func.go         # ItemChange() — 统一道具变更入口
│   ├── hero.logic.func.go              # HeroLevelUp/HeroCultivate（TODO: 接入消耗）
│   ├── formation.logic.func.go         # 上阵/下阵英雄 + 编队查询
│   ├── building.logic.func.go          # BuildingBuild/BuildingGetPb/BuildingListPb
│   └── march.logic.func.go             # MarchBuildTeam — 编队 → 出征队伍（英雄快照）
│
├── game_internals/                      # 内部基础设施
│   ├── game.internals.init.go          # Init(ctx)/ShutDown 聚合各子模块
│   ├── gate_stream/                    # 网关连接管理（Manager 结构体 + 私有单例）
│   │   └── gate_stream.go             # GateJoin/Gate/Push/GateCallBack*/ShutDown
│   ├── game_rpc_clients/               # 出站 RPC 客户端（包装 rpcconn 生成 hub）
│   │   ├── game.rpc.client.func.go    # hub 门面 + WorldMap()/WorldMapByInstance() 访问器 + ShutDown
│   │   ├── game.rpc.conns.st.go       # GameRpcClientHandler 单例（默认本服 instance，支持按 instance 多连接）
│   │   └── worldmap_client/            # worldmap 客户端门面（按 instance 配对）
│   │       ├── worldmap.client.func.go # Unary 业务方法（CreateMarch/MapData...）
│   │       └── worldmap.client.stream.go # 角色视野流管理（RoleStream）
│   └── stream_consumers/               # Redis Stream 消费者（行军事件）
│       ├── consumer.init.func.go      # Init → redisstream.MultiConsume
│       └── march.consumer.func.go     # 行军事件处理（到达/回城/取消）
│
├── game_models/                         # GORM 数据模型
│   ├── model.role_item.go             # RoleItem
│   ├── model.hero.go                  # RoleHero
│   ├── model.hero.skill.go            # HeroSkill
│   ├── model.hero.skillcollection.go  # HeroSkillCollection
│   └── model.cultivate.cost.go        # CultivateCost
│
└── GAME_OVERVIEW.md                     # 本文档
```

---

## 各层职责

| 层 | 包 | 职责 | 关键约束 |
|---|---|---|---|
| **模型** | `game_models/` | GORM 模型定义，纯数据，含 gorm tag + TableName | 不引入业务逻辑 |
| **实体** | `game_entitys/game_roles/` | 内存数据容器 + 索引(Mem map) + Init/Copy/Format2Pb | 不包含业务决策逻辑 |
| **逻辑** | `game_logics/` | 业务编排（ChangeItem/HeroLevelUp 等），以 `*Role` 为首参 | 操作实体数据，不直接处理协议 |
| **协议路由** | `game_handlers/` | 协议注册表 + 消息分发(Do/Recv) + 处理器 | 路由层，不包含业务逻辑 |
| **基础设施** | `game_internals/` | 连接管理、推送等通用能力 | 不依赖 game 业务 |
| **协议处理器** | `game_handlers/*` | HandlerHeroList 等具体协议处理 | 调 logic 层完成业务 |

---

## 请求处理流程

```
                         Stream 模式                           Unary 模式
                         ───────────                           ──────────
客户端 ──长连──▶ Gateway ──▶ GameStream.Stream()             客户端 ──▶ GameHandler.Do()
                              │                                          │
                          gateConnectDo()                            GetProtoHandler()
                              │                                          │
                          GateJoin(Recv)                              handler.F()
                              │
                    ┌─────────┴──────────┐
                    ▼                    ▼
              业务协议路由          相机移动等
              Recv()              转发到 WorldMap
                    │
              GetProtoHandler()
                    │
              handler.F() → game_logics.*
                    │
              GateCallBackSuccess() 推回客户端
```

---

## 核心设计

### 1. 双协议入口

| 方式 | 服务 | 用途 |
|------|------|------|
| `GameService.Stream` | `game_streams/` | Gateway 长连接 → 客户端请求转发 + 服务端推送 |
| `GameHandler.Do` | `game_servers/` | 内部服务/网关 Unary 调用 |

两者共享同一套 `protoRegistry`，处理器只需注册一次。

### 2. 共享协议注册表

```go
// game.protocol.registry.go
RegisterProto(msgID, &ProtoHandler{F: handler, Req: req, Resp: resp})

// Stream / Do 通用
GetProtoHandler(msgID) → 反序列化 → handler.F() → 回响应
```

### 3. 写时复制 (Copy-on-Write)

`Role.Copy(rw)` 创建轻量副本，子模块在首次 `Get*()` 时通过 `copyLock` 触发深拷贝，读多写少场景减少锁竞争。

### 4. 数据持久化 (Poller)

所有角色数据通过 `PollerManager[*Role]` 统一管理：

| 环境 | 缓存写入 | DB 写入 | 缓存 TTL |
|------|---------|---------|---------|
| 开发/测试 | 每 1s | 每 10s | 6h |
| 正式 | 每 30s | 每 1min | 12h |

### 5. 全局 Context

所有长驻模块必须通过 `Init(ctx)` 接收全局 context，监听 `ctx.Done()` 响应服务关闭。详见 CLAUDE.md 规范。

---

## 跨服务通信

game 与 worldmap（地图引擎节点）按 `instance` **单例配对**（同一区服的 game 服 ↔ 它的 worldmap）。

```
            ┌─────────────────────────────────────────────┐
            │  Redis Stream（worldmap → game，异步事件）    │
            │  StreamKeyMarchEvents: ARRIVED/BACKARRIVED/CANCELED
            │        ↑ worldmap.publishMarchEvent          │
            │        ↓ stream_consumers.MultiConsume       │
            └─────────────────────────────────────────────┘

game ──Unary/Stream──▶ worldmap
   game_rpc_clients ──▶ rpcconn 生成 hub（instance 感知发现）
                        → etcd NodeWorldMapService{instance}
```

| 方向 | 通道 | 链路 |
|------|------|------|
| game → worldmap（Unary） | gRPC `WorldMapHandler` | `game_rpc_clients.WorldMap()` → `rpc_handlers.ClientHandler` → `rpc_conns` 连接池 |
| game → worldmap（流） | gRPC `WorldMapService.Stream` | `ConnectRoleStream`（握手 → 相机移动上行 / 视野下推） |
| worldmap → game（事件） | Redis Stream | `common/redisstream` `ProtoXAdd` → `stream_consumers` 消费 |

基础连接维护统一在 `common/conns/rpcconn`：
- `rpc_conns` — gRPC 连接池（按地址复用、生命周期管理）
- `rpc_handlers` — 代码生成的 typed client hub（`client_handler_gen` 生成，绑定 instance）
- `etcdconn.GetNodeTypeServerAddrByInstance` — 按 nodeType + instance 精确发现

---

## 外部依赖

- `google.golang.org/protobuf` — protobuf 序列化
- `google.golang.org/grpc` — gRPC 通信
- `gorm.io/gorm` / `gorm.io/driver/mysql` — ORM
- `github.com/bwmarrin/snowflake` — 雪花 ID 生成
- `github.com/robfig/cron/v3` — 定时调度
- `go.etcd.io/etcd/client/v3` — 服务注册发现
- `server.slg.com/services/internal/cores` — 地图引擎层

---

## 当前状态

| 子系统 | 状态 | 说明 |
|--------|------|------|
| Role 聚合 + 子模块挂载 | ✅ | heroes/items/skills/costs 已挂载 |
| 道具背包 (Items) | ✅ | model/entity/db/logic 完整 |
| 协议路由框架 | ✅ | registry + Do + Recv 完整 |
| Stream 连接管理 | ✅ | gate_stream（Manager 结构体）+ gateConnectDo |
| 出站 RPC 客户端 | ✅ | game_rpc_clients → rpcconn hub，instance 感知发现 |
| 行军事件消费 | ✅ | stream_consumers 消费 Redis Stream（到达/回城/取消） |
| 士兵模型 | ✅ | 上阵默认100兵 + 兵力上限（英雄等级+兵营，soldier 配置驱动）+ 回城战损写回 formation；补兵机制⏸ 后续 |
| 兵营建筑 | ✅ | BuildingType.RoleBarracks + CityID 归属城市，读等级算兵力加成 |
| 城市内建筑 | ✅ | 建造/升级体系（building 配置域 + 资源扣除 + 建造时长惰性结算 + 校场队列同步）；分城归 worldmap OverlayEvent ⏸ |
| 配置表系统 | ✅ | api/game_conf 对齐 LDL：24 张 excel 源表 → tabtoy 导出单一 gameconfig.json → protoc pb.go 强类型；运行时 NewFromPB + 跨表校验 + mtime 热更；go:embed 同源兜底（详见 [CONFIG_OVERVIEW.md](../../api/game_conf/CONFIG_OVERVIEW.md)） |
| 英雄基础 | ⚠️ | 实体/模型完成，逻辑层待接入消耗 |
| 协议处理器 | ⬜ | 空模板，待按需补充 |
