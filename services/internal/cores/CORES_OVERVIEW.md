# Cores — SLG 服务端核心功能模块

> 生成日期: 2026-07-03  
> 路径: `services/internal/cores`  
> 模块: `server.slg.com/services/internal/cores`

## 概述

`cores` 是 SLG（策略类游戏）服务端的**核心逻辑包**，负责地图数据管理、行军系统、AOI（Area of Interest）视野管理、Tick 定时调度等基础游戏机制。它是整个游戏服务器中支撑地图玩法的心脏模块。

---

## 包结构

```
cores/
├── cores_declarations/    # 核心类型与接口声明
├── map_aois/              # AOI 视野管理（Screen / ScreenData）
├── map_blocks/            # 地图区块划分
├── map_borns/             # 出生块管理
├── map_connects/          # 角色连接与会话管理
├── map_datas/             # 地图格子数据
│   ├── map_buildings/     # 建筑覆盖层
│   └── map_events/        # 事件覆盖层
├── map_managers/          # 地图管理器（核心调度引擎）
├── marchdos/              # 行军执行器（单/多行军动作）
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

## 包详解

### 1. `cores_declarations` — 核心声明

**文件**: `core.const.go` · `core.if.go` · `core.st.go`

所有核心包共享的基础类型定义，作为全局类型中心。

| 类型 | 说明 |
|---|---|
| `MarchID` (uint64) | 行军唯一 ID |
| `MapID` (int32) | 地图格子 ID |
| `MarchType` (uint32) | 行军类型（如 110101） |
| `MarchState` (uint32) | 行军状态（如 Idle） |
| `MapGroup` (uint32) | 地图分组 |
| `RoleMainCityState` | 玩家主城状态（Normal / Portable） |
| `MapLevel` (int) | 地图等级 |
| `ElementType` (int) | 地图元素类型（None / 1~4） |

**关键常量**:
- `ServerMapBlockCutNum = 25` — 地图切块总数
- `ServerMapBlockRowCutNum = 5` — 每行切块数
- `Land1CoverBaseKey = 1` / `Land3CoverBaseKey = 4` — 地块覆盖关键索引
- `RoleMainCityStateNormalCoverCount = 9` — 普通主城占用 9 格
- `RoleMainCityStatePortableCoverCount = 1` — 便携主城占用 1 格

**接口定义**:
- `AoiScreenI` — AOI 屏幕接口（标记接口）
- `MarchHeroI` — 行军武将接口（标记接口）
- `MarchSoldierI` — 行军士兵接口（`GetCurCount`, `GetMaxCount`, `GetInjuredCount`）
- `MarchInfoI` — 行军信息接口（`GetMarchID`, `AddPassingAOIBlock`, `AddAOIBlock`）
- `MarchDoFuncHandleI` — **行军执行处理接口**，定义行军动作的完整生命周期：
  - `Do()` / `LockDo()` / `CallBack()` / `CallBackNow()` — 核心动作
  - `Lock()` / `UnLock()` — 并发锁
  - `Leave()` — 行军离开

**结构体**:
- `AnyThingUse` — 通用 KV 容器（`K uint32`, `V uint64`），用于存储行军动作消耗等键值对

---

### 2. `map_aois` — AOI 视野管理

**文件**: `aoi.screen.st.go` · `data.screen.st.go` · `data.screen.func.go`

AOI（Area of Interest）系统管理的"感兴趣区域"，即玩家的视野屏幕格子。采用网格划分+九宫格缓存的设计，将地图按 `ScreenWeight=40` 切分为等大的 Screen 格子。

- **`Screen[T cores_declarations.ScreenID]`** — 泛型 AOI 屏幕：
  - **连接管理**：`ConnectRoleIDs`、`Connects`（遍历格子内玩家）
  - **行军管理**：`MarchAdd`/`MarchDelete`/`MarchRange`（行军）、`PassingMarchAdd`/`PassingMarchDelete`/`PassingMarchRange`（经过行军）
  - 行军注册时自动回调 `info.AddAOIBlock(s)` / `info.AddPassingAOIBlock(s)`，建立双向关联
- **`ScreenData`** — AOI 屏幕数据管理器：
  - `AroundByScreen(screen)` — 计算并缓存九宫格（四周 8 个邻居 + 自身），首次计算后以 `atomic.Pointer` 缓存
  - `Cover(mapID, cover)` — 以 mapID 为中心，按 cover 半径（Screen 粒度）取覆盖范围
  - `Move(conn, newMapID)` — 玩家视野移动，自动从旧 Screen 删除并注册到新 Screen
  - `MovePath(x,y, x2,y2)` — 沿直线方向以 `step=screenScopeHalf*2` 跳跃计算经过的所有 Screen
  - `Exit(roleConn)` — 玩家离开 AOI 系统
  - `AroundConnects(mapID, connects)` — 获取九宫格内所有玩家连接

---

### 3. `map_blocks` — 地图区块

**文件**: `map.block.st.go` · `map.block.func.go`

- **`MapBlock`** — 地图块结构体（当前为空结构），用于将大地图按区块进行分区管理。
- `NewMapBlock(mapData)` — 通过 `MapDataManager` 创建地图块。

---

### 4. `map_borns` — 出生块管理

**文件**: `bigmap.born.st.go` · `sort.born.go` · `temp.born.st.go`

实现了 `cores_declarations.BornBlockI` 接口，管理玩家出生点的分配与回收。

**`BigMapBornBlockManager`** — 大地图出生块管理器，双池状态机（空闲池 ↔ 使用池）：
- `Store(bornID, data)` / `Load(bornID)` / `Use(bornID)` / `Free(bornID)` — 出生块生命周期
- `Range(f)` — 按 `blockSort` 优先级遍历空闲块

**`TempBornBlockManager`** — 临时出生块管理器（简化版，适用于临时地图）

详见 [docs/map_borns.md](docs/map_borns.md)

---

### 6. `map_datas` — 地图格子数据（最核心数据层）

#### 6.1 接口

**`MapConfigI`** — 地图配置接口：
- `MapCount() uint32` — 地图格子总数
- `MapID2XY(id)` — 将 MapID 映射为 (x, y) 坐标

#### 6.2 `MapInfo` — 单格信息

**文件**: `map.info.st.go`

地图上每一个格子的完整数据：

| 字段 | 类型 | 说明 |
|---|---|---|
| `mapID` | `MapID` | 格子唯一 ID |
| `coreMapID` | `MapID` | 核心/基础 MapID（多格子归属时指向核心格） |
| `x, y` | `int` | 格子坐标 |
| `serverID` | `uint32` | 归属服务器 ID |
| `ownerID` | `uint64` | 归属角色 ID |
| `level` | `MapLevel` | 格子等级 |
| `configID` | `uint32` | 配置 ID |
| `elementType` | `ElementType` | 元素类型 |
| `protectedEndTime` | `int64` | 保护到期时间戳 |
| `overlayEvent` | `*OverlayEvent` | 叠加的事件 |
| `overlayBuilding` | `*OverlayBuilding` | 叠加的建筑 |

提供 `TryLock` / `UnLock` 并发安全访问，`Free()` 用于重置。

#### 6.3 `MapDataManager` — 地图数据管理器

**文件**: `map.datamanager.st.go` · `map.datamanager.func.go`

**核心职责**：管理所有地图格子的生命周期。

- **`Init(mapD)`** — 初始化地图数据，遍历所有格子，将有效格子的基础坐标注册到 AOI。
- **`GetMapInfo(mapID)`** — 根据 MapID 获取格子指针 + 是否有效。
- **`GetMapInfoSlice(mapIDs)`** — 批量获取格子。
- **`Range(f)`** — 遍历所有有效格子。
- **`TryLock(mapList)` / `UnLock(mapList)`** — 批量加锁/解锁（失败时自动回滚已锁的格子）。
- **`Save(list...)`** — 将修改的格子存入等待保存队列。
- **`Clear(mapIDs)`** — 清理指定格子（TODO）。
- **`SetRoleMainCity(roleCityState, dataSlice, roleBrief)`** — 设置玩家主城：
  - 根据主城状态（Normal/Portable）校验格子数量
  - 在 `Land1CoverBaseKey` 或 `Land3CoverBaseKey` 处计算核心位置
  - 写入 serverID、ownerID、coreMapID
  - 触发 AOI 更新
- **`GetFreeBorn()`** — 从空闲出生块中查找可用空地，自动上锁并返回 `LockMapSlice` + `freeBornFunc`

#### 6.4 子包

| 子包 | 文件 | 说明 |
|---|---|---|
| `map_buildings` | `map.building.st.go` | 建筑覆盖层（BaseBuildings/NpcBuilding/NpcCity） |
| `map_events` | `map.event.st.go` | `OverlayEvent` 空结构（事件覆盖层占位） |

详见 [docs/map_datas.md](docs/map_datas.md)

---

### 7. `map_connects` — 角色连接与会话管理

**文件**: `connect.manager.st.go` · `map.connects.var.go` · `role.connect.st.go`

管理玩家与服务器的 gRPC 流连接，提供基于 AOI 九宫格的消息推送。

**`RoleConnectManager`** — 连接管理器：
- 连接生命周期：`NewRoleConnect` / `CloseRoleConnect` / `LoadRoleConnect`
- 视野管理：`SetRoleScreen(roleID, mapID)`
- 消息推送：`PushToScreen`（AOI 九宫格自动去重）、`PushToRoleID`、`PushToRoleIDs`
- 自动断线清理：gRPC 错误码检测自动关闭连接

详见 [docs/map_connects.md](docs/map_connects.md)

---

### 8. `map_managers` — 地图管理器（核心调度引擎）

**文件**: `map.manager.st.go` · `manager.func.go` · `manager.march.func.go` · `manager.tick.func.go` · `map.manager.var.go` · `map.options.st.go`

#### 8.1 `MapManager` — 核心结构

| 字段 | 说明 |
|---|---|
| `RoomID` | 房间 ID |
| `MapGroup` | 地图分组 |
| `mapDataManager` | 地图数据管理器 |
| `marchManage` | 行军管理器 |
| `timeMarch` | 时间 → 行军 ID 集合（用于定时触发行军处理） |
| `timeMap` | 时间 → 地图格子 ID 集合（用于定时地图清理） |
| `marchDoFunc` | 行军执行回调函数 |
| `marchDoFuncHandle` | 行军执行处理器工厂函数 |
| `mapBlock` | 地图区块 |
| `opts` | 配置参数（stopChan, start/endTime, cutNum 等） |
| `waitUpdateMapID` | 待推送更新的地图集合 |

#### 8.2 生命周期

- **`NewMapManager(...)`** — 构造函数，通过函数注入（`marchDoFunc`, `marchDoHandleFunc`）支持策略模式，使行军处理逻辑可扩展。
- **`Start()`** — 启动两个后台 goroutine：
  - **`loopTickCheck()`** — 三个定时器层级：
    - **100ms ticker** — 行军到期处理 + 地图同步推送
    - **300ms ticker** — 地图清理
    - **1s ticker** — 兜底清理（防止 100ms/300ms 轮次丢失）
  - **`loopTickAccept()`** — 监听 `marchManage.TickerChan`，将到期的行军注册到 `timeMarch`

#### 8.3 行军 AOI 设置 (`manager.march.func.go`)

- **`MarchAOISetupSingle(marchInfo)`** — 通过起点和终点的坐标调用 `MarchAOISetup`。
- **`MarchAOISetup(marchInfo, startX, startY, endX, endY)`** — 根据行军是否为虚拟：
  - **虚拟行军**：在路径的每个位置注册行军（仅 AOI 通行，不产生战斗交互）
  - **非虚拟行军**：通过 `AOI.MovePath` 计算路径经过的 Screen，首尾调用 `MarchAdd`，中间调用 `CrossMarchAdd`

#### 8.4 Tick 管理 (`manager.tick.func.go`)

- `TickerAddMarch(marchID, endTime)` — 注册行军到指定时间槽
- `TickerAddMap(mapID, clearTime)` — 注册地图清理到指定时间槽
- `TickerAddMapList(mapIDList, clearTime)` — 批量注册

#### 8.5 对象池 (`map.manager.var.go`)

- `MapPBGet()` / `MapPBPut()` — 用于 protobuf 序列化对象的 sync.Pool 管理（当前返回 nil/TODO）

#### 8.6 配置选项 (`map.options.st.go`)

函数式选项模式（Functional Options Pattern）：
- `WithStopChan` — 停止信号
- `WithStartTime` / `WithEndTime` — 起止时间
- `WithCutNum` — 地图切块数量

---

### 9. `marchs` — 行军信息与队伍

#### 9.1 `MarchInfo` — 行军信息

**文件**: `march.info.st.go`

行军的完整状态快照，实现了 `MarchInfoI` 接口。

| 字段 | 说明 |
|---|---|
| `MarchID` | 行军唯一 ID |
| `MarchType` | 行军类型（如 110101） |
| `Team` | 出征队伍（武将+士兵） |
| `FromServerID` / `ToServerID` | 跨服行军起止服务器 |
| `FromRoleID` / `ExecRoleID` | 归属者/执行者角色 ID |
| `SrcFromMapID` / `FromMapID` / `ToMapID` | 原始/当前起点/终点地图 |
| `MarchState` | 行军状态 |
| `StartTimeUx` / `EndTimeUx` | 时间戳 |
| `FollowMarchID` | 跟随的行军 ID |
| `UnionID` | 联盟 ID |
| `BaseMarchSpeed` / `FinalMarchSpeed` | 基础/最终行军速度 |
| `ActionUse` | 行军消耗项（`[]AnyThingUse`） |
| `Path` | 行军路径（`[]MarchID`，JSON 序列化到数据库） |
| `PVPWinCount` / `PVEWinCount` | 战斗胜场计数 |
| `VirtualData` | 虚拟行军数据 |
| `isVirtual` / `isMock` | 虚拟行军 / 假行军标记 |
| `AoiBlock` / `PassingAoiBlock` | AOI 通行记录 |

提供丰富的 Getter 方法和 `TryLock()` / `UnLock()` / `LockMarchDo()` 并发控制。

#### 9.2 `MarchInfoManager` — 行军管理器

**文件**: `march.infomanage.st.go` · `march.infomanager.func.go` · `march.infomanager.db.func.go`

| 字段 | 说明 |
|---|---|
| `TickerChan` | 行军到期通知通道（向 MapManager 的 tick 系统投递消息） |
| `allMarch` | 全局行军集合（MarchID → MarchInfo） |
| `allAssembleMarch` | 组合行军集合 |
| `mapMarch` | 地图行军属性列表（`[]MapAttribute`） |

| 方法 | 说明 |
|---|---|
| `New(tickerChan, tableName, mapConfig, marchTimeType)` | 构造函数 |
| `Init(dbc)` | 从数据库加载所有行军（含 MapAttribute 挂载 + 驻守恢复） |
| `MapAttributeGet(mapID)` | 获取指定地图的行军属性 |
| `MapAttributeMarchCreate(marchInfo)` | 创建行军时挂载到起止点地图 |
| `MapAttributeMarchDelete(marchInfo)` | 删除行军时从起止点地图移除 |
| `MapAttributeMarchChange(marchInfo, newMapID)` | 行军改目标位置 |
| `MapAttributeMarchModToMapID(marchInfo, newToMapID)` | 修改行军目标地图 |
| `MapAttributeMarchModFormMapID(marchInfo, newMapID, isAllForm)` | 修改行军起始地图 |
| `MapAttributeMarchCallBack(marchInfo)` | 行军返回处理 |
| `CreateMarch(marchInfo)` | 创建行军（入库 + 注册） |
| `CreateMarchInBatches(marchInfoList...)` | 批量创建行军 |
| `CreateMockMarch(marchInfo)` | 创建假行军（不入库） |
| `DeleteMarch(marchInfo)` | 删除行军（入库删除 + AOI清理） |
| `AllMarch()` | 获取全部行军 |
| `GetMarchInfo(marchID)` | 单条行军查询 |
| `GetMarchInfoByType(marchTypes...)` | 按类型查询行军 |

#### 9.3 `MapAttribute` — 地图行军属性

**文件**: `map.attribute.st.go` · `map.attribute.func.go`

管理**单张地图**上的：
- `assistSlice` — 驻守队伍列表
- `marchMap` — 经过的行军集合
- 提供 `marchAdd` / `marchDel` / `GetMapMarch` / `RangeMapMarch` / `GetAllMapMarchLen` / `AssistArrive`

#### 9.4 `Team` — 行军队伍

**文件**: `march.team.st.go`

- `Slots` — 槽位列表（`[]*pb_battle.TeamSlotInfo`）
- `Format2Pb()` — 格式化为 `pb_battle.TeamInfo`
- `GetAliveSoliderCount()` — 存活士兵总数
- `GetMaxCount()` — 最大兵力
- `CheckCanFight()` — 检查 0 号位武将是否可战斗

---

### 10. `roles` — 角色数据管理

**文件**: `role.data.st.go` · `role.data.copy.func.go` · `role.data.db.func.go` · `role.brief.st.go` · `role.poller.func.go` · `role.poller.var.go` · `role.queue.st.go`

角色数据管理模块，采用**写时复制（Copy-on-Write）** + **轮询器（Poller）** 模式管理玩家数据。

**`Data`** — 角色数据：Queue（生成队列）、Brief（简略信息）、LastConnectTime
- 实现了 `common_declarations.DataI` 接口
- `Copy(rw)` — 创建副本，延迟深拷贝
- `GetBrief()` / `GetQueue()` — 副本模式触发延迟拷贝
- `AddQueue()` / `ReleaseRoleQueue()` — 队列管理

**轮询器系统**：三级轮询周期（30s / 1min / 半天）
- `Get(id)` → `(data, freeFunc, releaseFunc, err)` — 标准获取模式
- `GetCopy(id)` — 获取副本（COW 模式）

详见 [docs/roles.md](docs/roles.md)

---

### 11. `marchdos` — 行军执行器

**文件**: `base.march.st.go` · `single.march.st.go` · `multi.march.st.go`

行军到达目的地后，执行具体动作的模块，采用**模板方法模式**。

#### 11.1 `BaseMarch` — 基类

- **生命周期阶段**（三个阶段，按顺序执行）：
  1. `prepareOpts` — 准备阶段（锁定资源、验证条件）
  2. `doOpts` — 执行阶段（实际动作）
  3. `finishOpts` — 完成阶段（清理、回调通知）
- 方法：`AddPrepareOpt` / `AddDoOpt` / `AddFinishOpt` 添加各阶段的处理函数
- `Do(mapManager)` — 依次执行三个阶段
- **要求**：必须先调用 `Init()` 再调用 `Do()`，否则 panic

#### 11.2 `SingleMarch` — 单一行军执行器

- 持有单条 `MarchInfo`（`single` 字段）
- `TryLock(marchLock, fromLock, toLock)` — 分层加锁：先行军锁，再来源地图锁，再目标地图锁，任一失败则回滚已加的锁
- `arriveAfterFunc` — 到达后的回调函数

#### 11.3 `MultiMarch` — 多行军执行器

- 持有多条 `MarchInfo`（`multi` 切片）
- `TryLock(...)` — 使用位掩码 `markOff` 逐条锁定多条行军，任一条加锁失败则回滚
- `SetArriveAfterFunc` — 设置所有行军到达后的回调

---

## 核心架构关系图

```
                        ┌─────────────────────────────────┐
                        │          MapManager              │
                        │   (核心调度引擎)                   │
                        │                                  │
                        │  ┌─ loopTickCheck() (100ms定时)   │
                        │  │   ├─ 行军到期处理              │
                        │  │   ├─ 地图同步推送              │
                        │  │   └─ 兜底清理(1s)              │
                        │  │                                │
                        │  └─ loopTickAccept()              │
                        │      └─ TickerChan ← MarchInfo    │
                        └──────┬───────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────────────┐
        │                      │                              │
        ▼                      ▼                              ▼
┌───────────────┐   ┌──────────────────┐   ┌─────────────────────┐
│ MapDataManager│   │ MarchInfoManager │   │   MarchDoFuncHandle │
│ (地图格子数据)  │   │  (行军信息管理)   │   │  (行军执行器工厂)    │
│               │   │                  │   │                     │
│  MapInfo[]    │   │  allMarch        │   │  ┌─ SingleMarch    │
│  AOI Screen   │   │  TickerChan      │   │  ├─ MultiMarch     │
│  Save Queue   │   │  allAssembleMarch│   │  └─ BaseMarch(基类) │
└───────┬───────┘   └────────┬─────────┘   └─────────────────────┘
        │                    │
        ▼                    ▼
┌───────────────┐   ┌──────────────────┐
│   MapInfo     │   │    MarchInfo     │
│ (单个格子)     │   │  (行军信息状态)   │
│               │   │                  │
│ 坐标/等级/    │   │  起止点/时间/队伍 │
│ 归属/建筑/事件 │   │  路径/AOI/战斗   │
└───────────────┘   └──────────────────┘
                            │
                            ▼
                    ┌──────────────────┐
                    │      Team        │
                    │  (行军队伍)       │
                    │                  │
                    │  武将 + 士兵      │
                    │  存活/受伤/战斗  │
                    └──────────────────┘
```

---

## 关键设计模式与亮点

1. **函数式选项模式 (Functional Options)**
   - `MapManager` 的配置使用 `Option` 模式，支持灵活扩展（`WithStopChan`, `WithCutNum` 等）。

2. **策略模式 (Strategy)**
   - `marchDoFunc` 和 `marchDoHandleFunc` 通过构造函数注入，使行军到达处理逻辑与 `MapManager` 解耦。

3. **模板方法模式 (Template Method)**
   - `BaseMarch.Do()` 定义了行军执行的固定三步流程（prepare → do → finish），子类只需注入各阶段的处理函数。

4. **分层并发控制**
   - 三级加锁策略：行军锁 → 来源地图锁 → 目标地图锁，任一失败自动回滚。
   - `MultiMarch` 使用位掩码（`markOff` 和 `1 << inx`）实现高效的批量加锁状态追踪。

5. **定时调度系统**
   - 三级 tick（100ms / 300ms / 1s），100ms 负责行军到期精准触发，1s 负责兜底清理，防止 tick 轮次丢失导致的行军"卡死"。

6. **AOI 分离**
   - 虚拟行军 vs 非虚拟行军：虚拟行军仅做路径 AOI 注册（不触发战斗），非虚拟行军通过 `MovePath` 精确计算 Screen 通行范围。

7. **ORM 集成**
   - `MarchInfo` 通过 GORM 标签（`gorm:"type:json;serializer:json"`）将行军路径 `Path` 序列化为 JSON 存储。

---

## 当前状态与 TODO

该模块整体处于**开发中/部分实现**状态：

| 组件 | 状态 |
|---|---|
| `cores_declarations` | ✅ 基本完成 |
| `map_aois` (Screen / ScreenData) | ✅ 功能完善，按业务场景裁剪（无缩放层级、无空白地块管理，与 ldl 版对齐） |
| `map_blocks` | ⬜ 骨架实现 |
| `map_borns` (BigMapBornBlockManager / TempBornBlockManager) | ✅ 功能完善 |
| `map_connects` (RoleConnectManager) | ✅ 功能完善 |
| `map_datas` (MapInfo / MapDataManager) | ✅ 核心逻辑完成，`Clear()` 为 TODO |
| `map_buildings` (BaseBuildings / NpcBuilding / NpcCity) | ✅ 建筑血量/等级/战斗逻辑完成 |
| `map_events` | ⬜ 空结构占位 |
| `map_managers` (MapManager) | ✅ 核心调度完成，`clearMapFunc()` 为空 |
| `map.manager.var.go` (对象池) | ⬜ 返回 nil/TODO |
| `marchs` (MarchInfo / Team / MapAttribute) | ✅ 核心逻辑完成，定期保存 `SaveDo()` 为空 |
| `marchdos` (BaseMarch / SingleMarch / MultiMarch) | ⬜ 到达回调为空实现 |
| `roles` (Data / Poller) | ✅ 数据管理层完成，`Save()` 为 panic |
| 跨服行军 | ⬜ `allAssembleMarch` 相关逻辑尚未实现 |
| 数据库持久化 | ✅ 自动迁移 + 查询 + 创建/删除完成，定期保存 `SaveDo()` 为空 |

---

## 外部依赖

- `server.slg.com/common/utils/hashmaps` — 自定义哈希表实现
- `server.slg.com/common/utils/asyncsave_entity` — 异步实体持久化接口
- `server.slg.com/common/pollers` — 数据轮询管理器
- `server.slg.com/common/conns/dbconn` — 数据库连接（读写分离）
- `server.slg.com/common/conns/rpcconn/rpc_streams` — gRPC 流管理
- `server.slg.com/api/protocol/pb/*` — protobuf 协议（pb_role, pb_maps_march, pb_battle, pb_city, pb_camera 等）
- `github.com/go4org/hashtriemap` — 并发哈希 Trie 图
- `github.com/patrickmn/go-cache` — 内存缓存
- `go.uber.org/zap` — 日志
- GORM — ORM 框架（`AutoMigrate`, `Create`, `Find`, `Delete`）
