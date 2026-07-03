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
├── aois/                  # AOI 视野管理
├── cores_declarations/    # 核心类型与接口声明
├── map_blocks/            # 地图区块划分
├── map_datas/             # 地图格子数据
│   ├── map_declarations/  # 地图要素类型常量
│   ├── map_buildings/     # 建筑覆盖层
│   └── map_events/        # 事件覆盖层
├── map_managers/          # 地图管理器（核心调度引擎）
├── marchdos/              # 行军执行器（单/多行军动作）
└── marchs/                # 行军信息与队伍
```

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
- `MarchHero` — 行军武将接口（标记接口）
- `MarchSoldier` — 行军士兵接口（`GetCurCount`, `GetMaxCount`, `GetInjuredCount`）
- `MarchInfoI` — 行军信息接口（`GetMarchID`, `AddPassingAOIBlock`, `AddAOIBlock`）
- `MarchDoFuncHandleI` — **行军执行处理接口**，定义行军动作的完整生命周期：
  - `Do()` / `LockDo()` / `CallBack()` / `CallBackNow()` — 核心动作
  - `Lock()` / `Unlock()` — 并发锁
  - `Leave()` — 行军离开

**结构体**:
- `AnyThingUse` — 通用 KV 容器（`K uint32`, `V uint64`），用于存储行军动作消耗等键值对

---

### 2. `aois` — AOI 视野管理

**文件**: `aoi.screen.st.go` · `data.screen.st.go`

AOI（Area of Interest）系统管理的"感兴趣区域"，即玩家的视野屏幕格子。

- **`Screen[T int32 | uint32]`** — 泛型 AOI 屏幕，支持 `MarchAdd`（行军加入）和 `CrossMarchAdd`（行军经过）操作，当前均为 `panic("implement me")`。
- **`ScreenData`** — AOI 屏幕数据管理器：
  - `MapDataAdd(mapID)` — 向 AOI 注册地图格子
  - `GetScreen(id)` — 获取指定行军的 Screen
  - `MovePath(x,y, x2,y2)` — 计算两点之间的路径跨越的所有 Screen

---

### 3. `map_blocks` — 地图区块

**文件**: `map.block.st.go` · `map.block.func.go`

- **`MapBlock`** — 地图块结构体（当前为空结构），用于将大地图按区块进行分区管理。
- `NewMapBlock(mapData)` — 通过 `MapDataManager` 创建地图块。

---

### 4. `map_datas` — 地图格子数据（最核心数据层）

#### 4.1 接口

**`MapConfigI`** — 地图配置接口：
- `MapCount() uint32` — 地图格子总数
- `MapID2XY(id)` — 将 MapID 映射为 (x, y) 坐标

#### 4.2 `MapInfo` — 单格信息

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

提供 `TryLock` / `Unlock` 并发安全访问，`Free()` 用于重置。

#### 4.3 `MapDataManager` — 地图数据管理器

**文件**: `map.datamanager.st.go` · `map.datamanager.func.go`

**核心职责**：管理所有地图格子的生命周期。

- **`Init(mapD)`** — 初始化地图数据，遍历所有格子，将有效格子的基础坐标注册到 AOI。
- **`GetMapInfo(mapID)`** — 根据 MapID 获取格子指针 + 是否有效。
- **`GetMapInfoSlice(mapIDs)`** — 批量获取格子。
- **`Range(f)`** — 遍历所有有效格子。
- **`TryLock(mapList)` / `Unlock(mapList)`** — 批量加锁/解锁（失败时自动回滚已锁的格子）。
- **`Save(list...)`** — 将修改的格子存入等待保存队列。
- **`Clear(mapIDs)`** — 清理指定格子（TODO）。
- **`SetRoleMainCity(roleCityState, dataSlice, roleBrief)`** — 设置玩家主城：
  - 根据主城状态（Normal/Portable）校验格子数量
  - 在 `Land1CoverBaseKey` 或 `Land3CoverBaseKey` 处计算核心位置
  - 写入 serverID、ownerID、coreMapID
  - 触发 AOI 更新

#### 4.4 子包

| 子包 | 文件 | 说明 |
|---|---|---|
| `map_declarations` | `map.info.const.go` | `MapLevel` 和 `ElementType` 的枚举值 |
| `map_buildings` | `map.building.st.go` | `OverlayBuilding` 空结构（建筑覆盖层占位） |
| `map_events` | `map.event.st.go` | `OverlayEvent` 空结构（事件覆盖层占位） |

---

### 5. `map_managers` — 地图管理器（核心调度引擎）

**文件**: `map.manager.st.go` · `manager.func.go` · `manager.march.func.go` · `manager.tick.func.go` · `map.manager.var.go` · `map.options.st.go`

#### 5.1 `MapManager` — 核心结构

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

#### 5.2 生命周期

- **`NewMapManager(...)`** — 构造函数，通过函数注入（`marchDoFunc`, `marchDoHandleFunc`）支持策略模式，使行军处理逻辑可扩展。
- **`Start()`** — 启动两个后台 goroutine：
  - **`loopTickCheck()`** — 三个定时器层级：
    - **100ms ticker** — 行军到期处理 + 地图同步推送
    - **300ms ticker** — 地图清理
    - **1s ticker** — 兜底清理（防止 100ms/300ms 轮次丢失）
  - **`loopTickAccept()`** — 监听 `marchManage.TickerChan`，将到期的行军注册到 `timeMarch`

#### 5.3 行军 AOI 设置 (`manager.march.func.go`)

- **`MarchAOISetupSingle(marchInfo)`** — 通过起点和终点的坐标调用 `MarchAOISetup`。
- **`MarchAOISetup(marchInfo, startX, startY, endX, endY)`** — 根据行军是否为虚拟：
  - **虚拟行军**：在路径的每个位置注册行军（仅 AOI 通行，不产生战斗交互）
  - **非虚拟行军**：通过 `AOI.MovePath` 计算路径经过的 Screen，首尾调用 `MarchAdd`，中间调用 `CrossMarchAdd`

#### 5.4 Tick 管理 (`manager.tick.func.go`)

- `TickerAddMarch(marchID, endTime)` — 注册行军到指定时间槽
- `TickerAddMap(mapID, clearTime)` — 注册地图清理到指定时间槽
- `TickerAddMapList(mapIDList, clearTime)` — 批量注册

#### 5.5 对象池 (`map.manager.var.go`)

- `MapPBGet()` / `MapPBPut()` — 用于 protobuf 序列化对象的 sync.Pool 管理（当前返回 nil/TODO）

#### 5.6 配置选项 (`map.options.st.go`)

函数式选项模式（Functional Options Pattern）：
- `WithStopChan` — 停止信号
- `WithStartTime` / `WithEndTime` — 起止时间
- `WithCutNum` — 地图切块数量

---

### 6. `marchs` — 行军信息与队伍

#### 6.1 `MarchInfo` — 行军信息

**文件**: `march.info.st.go`

行军的完整状态快照，实现了 `MarchInfoI` 接口。

| 字段 | 说明 |
|---|---|
| `MarchID` | 行军唯一 ID |
| `Team` | 出征队伍（武将+士兵） |
| `FromServerID` / `ToServerID` | 跨服行军起止服务器 |
| `FromRoleID` / `ToRoleID` | 发起/目标角色 |
| `SrcFromMapID` / `FromMapID` / `ToMapID` | 原始/当前起点/终点地图 |
| `MarchState` | 行军状态 |
| `StartTimeUx` / `EndTimeUx` / `BaseEndTimeUx` | 时间戳 |
| `FollowMarchID` | 跟随的行军 ID |
| `UnionID` | 联盟 ID |
| `BaseMarchSpeed` | 基础行军速度 |
| `ActionUse` | 行军消耗项（`[]AnyThingUse`） |
| `Path` | 行军路径（`[]MarchID`，JSON 序列化到数据库） |
| `PVPWinCount` / `PVEWinCount` | 战斗胜场计数 |
| `isVirtual` | 是否为虚拟行军 |
| `AoiBlock` / `PassingAoiBlock` | AOI 通行记录 |

提供丰富的 Getter 方法和 `TryLock()` / `Unlock()` / `LockMarchDo()` 并发控制。

#### 6.2 `MarchInfoManager` — 行军管理器

**文件**: `march.infomanage.st.go` · `march.infomanager.func.go`

- `TickerChan` — 行军到期通知通道（向 MapManager 的 tick 系统投递消息）
- `allMarch` — 全局行军集合（MarchID → MarchInfo）
- `allAssembleMarch` — 组合行军集合
- `Init(dbc)` — 从数据库加载所有行军：自动迁移表结构 → 查询全部行军 → 逐条注册到 allMarch 并推送到 TickerChan

#### 6.3 `MapAttribute` — 地图行军属性

**文件**: `march.map.st.go`

管理**单张地图**上的：
- `assistSlice` — 驻守队伍列表
- `marchMap` — 经过的行军集合
- 提供 `marchAdd` / `marchDel` / `GetMapMarch` / `RangeMapMarch` / `GetAllMapMarchLen`

#### 6.4 `Team` — 行军队伍

**文件**: `march.team.st.go`

- `Heros` — 武将列表（`[]MarchHero`）
- `Soldiers` — 士兵列表（`[]MarchSoldier`）
- `GetAliveSoliderCount()` — 存活士兵总数
- `GetInjuredCount()` — 受伤士兵总数
- `GetMaxCount()` — 最大兵力
- `IsHasInjured()` — 是否有受伤
- `CheckCanFight()` — 检查 0 号位的武将是否还有兵力（核心战斗条件检查）

---

### 7. `marchdos` — 行军执行器

**文件**: `base.march.st.go` · `single.march.st.go` · `multi.march.st.go`

行军到达目的地后，执行具体动作的模块，采用**模板方法模式**。

#### 7.1 `BaseMarch` — 基类

- **生命周期阶段**（三个阶段，按顺序执行）：
  1. `prepareOpts` — 准备阶段（锁定资源、验证条件）
  2. `doOpts` — 执行阶段（实际动作）
  3. `finishOpts` — 完成阶段（清理、回调通知）
- 方法：`AddPrepareOpt` / `AddDoOpt` / `AddFinishOpt` 添加各阶段的处理函数
- `Do(mapManager)` — 依次执行三个阶段
- **要求**：必须先调用 `Init()` 再调用 `Do()`，否则 panic

#### 7.2 `SingleMarch` — 单一行军执行器

- 持有单条 `MarchInfo`（`single` 字段）
- `TryLock(marchLock, fromLock, toLock)` — 分层加锁：先行军锁，再来源地图锁，再目标地图锁，任一失败则回滚已加的锁
- `arriveAfterFunc` — 到达后的回调函数

#### 7.3 `MultiMarch` — 多行军执行器

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
| `aois` (Screen / ScreenData) | ⬜ 骨架实现，`MarchAdd` 等方法为 `panic("implement me")` |
| `map_blocks` | ⬜ 骨架实现 |
| `map_datas` (MapInfo / MapDataManager) | ✅ 核心逻辑完成，`Clear()` 为 TODO |
| `map_managers` (MapManager) | ✅ 核心调度完成，`upMapSync()` / `clearMapFunc()` 为空 |
| `map.manager.var.go` (对象池) | ⬜ 注释掉的 protobuf 逻辑，返回 nil |
| `marchs` (MarchInfo / Team / MapAttribute) | ✅ 核心逻辑完成，`TableName()` 为 panic |
| `marchdos` (BaseMarch / SingleMarch / MultiMarch) | ⬜ Init / SetArriveAfterFunc 中空回调 |
| 跨服行军 | ⬜ `allAssembleMarch` 相关逻辑尚未实现 |
| 数据库持久化 | ✅ 自动迁移 + 查询完成，定期保存逻辑 `SaveDo()` 为空 |

---

## 外部依赖

- `server.slg.com/common/utils/hashmaps` — 自定义哈希表实现
- `server.slg.com/api/protocol/pb/pb_role` — protobuf 角色协议
- `server.slg.com/common/conns/dbconn/dbconn_interface` — 数据库连接抽象
- GORM — ORM 框架（`AutoMigrate`, `Find`）
