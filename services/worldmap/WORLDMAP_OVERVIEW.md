# Worldmap — 地图引擎服务

> 路径: `services/worldmap`  
> 模块: `server.slg.com/services/worldmap`

---

## 概述

`worldmap` 是 SLG 服务端的**地图引擎节点**，持有 `services/internal/cores` 引擎（AOI 视野 / 行军 / 地块状态），负责大地图的视野管理与行军状态机。战斗**计算**委托独立 `battle` 节点（见 `services/battle/BATTLE_OVERVIEW.md`），worldmap 在行军到达时通过注入回调调用并应用结果。与 `game`（区服业务层）按 `instance` **单例配对**：一个 game 服对应一个 worldmap 节点。

---

## 目录结构

```
worldmap/
├── main.go                                    # 入口：配置/日志/DB/ETCD/gRPC + 生命周期
│
├── worldmap_handlers/                          # 协议入口
│   ├── worldmap_servers/                      # Unary RPC 处理器（WorldMapHandler）
│   │   └── worldmap.server.func.go           # CreateMarch/CancelMarch/MarchInfo/MapData
│   └── worldmap_streams/                      # Stream 处理器（WorldMapService）
│       ├── worldmap.stream.st.go             # WorldMapStream 结构体 + SetEngine
│       └── worldmap.stream.func.go           # Stream() 握手 + recv() 相机移动 + 视野下推
│
└── worldmap_internals/                         # 内部实现
    └── worldmap_inits/                        # cores 引擎聚合 + 地图配置
        ├── engine.func.go                    # Engine 聚合 + 行军结算 + 事件发布
        ├── map.config.func.go                # DefaultMapConfig（1000×1000）
        └── map.generate.func.go              # 地图元素生成（种子确定性）
```

> 注：`worldmap_streams/` 目录内文件 Go 包名为 `worldmap_handlers`（与目录名不同，main.go 以包名引用）。

---

## 核心设计

### 1. Engine 聚合（`worldmap_inits.NewEngine(ctx)`）

统一持有 cores 各管理器，Handler 通过 `SetEngine(engine)` 注入：

| 字段 | 类型 | 职责 |
|------|------|------|
| `Config` | `*DefaultMapConfig` | 1000×1000 格，坐标 ↔ MapID 换算 |
| `MapDataManager` | `*map_datas.MapDataManager` | 地块数据 + AOI（`ScreenData`） |
| `MarchInfoManager` | `*marchs.MarchInfoManager` | 行军信息管理 |
| `MapManager` | `*map_managers.MapManager` | 地图管理器（AOI 连接、视野 push） |

初始化时 `InitMapElements` 按种子确定性生成地图元素（保证视野有数据、出生点可诞生）。

### 2. 行军结算 → 事件发布

```
march tick → MarchTickHandler
  → 到点锁定行军 → march_factory.NewMarchDo（按 MarchType 分派 attack/develop/assist）
  → handle.Do() 结算（战斗/采集/驻守）
    ├─ 失败 → CallBack() 召回
    └─ 成功 → OnMarchArrived(marchInfo)
               └─ 按 state 分派事件类型 → publishMarchEvent → Redis Stream
```

| 事件类型 | 含义 | 触发 |
|---------|------|------|
| `MARCH_EVENT_ARRIVED` | 目标点结算（占领/采集/驻守） | state ≠ Back |
| `MARCH_EVENT_BACKARRIVED` | 行军回城到站 | state = Back |
| `MARCH_EVENT_CANCELED` | 行军取消 | `OnMarchCanceled` |

发布走 `common/redisstream.ProtoXAdd(ctx, StreamKeyMarchEvents, data)`，key 全局统一声明。

### 3. 视野流（WorldMapService.Stream）

game 为每个玩家建立到 worldmap 的双向视野流：

```
握手（WorldMapConnectReq: roleID + mapID）
  → RoleConnectManager.NewRoleConnect 注册到 AOI
  → receiveF 循环：
      相机移动（MsgID_GameCameraMove）→ SetRoleScreen(roleID, mapID)
        → buildCameraMoveResp：AOI 九宫格内地块 + 行军
        → PushToRoleID 下推
  → 断开 → 清理 AOI 连接
```

### 4. Unary RPC（WorldMapHandler）

| RPC | 说明 |
|-----|------|
| `CreateMarch` | 创建行军（含出征队伍 team_slots），返回 march_id + end_time |
| `CancelMarch` | 取消行军，触发 `MARCH_EVENT_CANCELED` |
| `MarchInfo` | 查询行军信息 |
| `MapData` | AOI 视野查询：`range<=0` 九宫格 / 否则 cover 范围，返回地块 + 行军 |

---

## 通信

```
                    ┌────────────────────────────────┐
                    │  Redis Stream（发布给 game）    │
                    │  StreamKeyMarchEvents           │
                    └────────────────────────────────┘
                            ↑ ProtoXAdd

game ──Unary/Stream──▶ worldmap（本服务）
  WorldMapHandler（CreateMarch/CancelMarch/MarchInfo/MapData）
  WorldMapService.Stream（玩家视野流）

ETCD 注册：/node:service:worldmap/{instance}/ （与 game 同 instance 配对）
```

- **接收 game**：`WorldMapHandler`（Unary）+ `WorldMapService`（Stream）
- **发布事件**：行军到达/回城/取消 → Redis Stream，game 的 `stream_consumers` 消费
- **发现**：game 通过 `etcdconn.GetNodeTypeServerAddrByInstance(NodeWorldMapService, instance)` 精确定位本节点

---

## 外部依赖

- `services/internal/cores` — 地图引擎（AOI/行军/战斗）
- `common/redisstream` — Redis Stream 发布
- `common/conns/etcdconn` — ETCD 服务注册
- `common/conns/dbconn` — MySQL（行军/地块持久化）
- `api/protocol/pb/pb_worldmap` — 协议定义
