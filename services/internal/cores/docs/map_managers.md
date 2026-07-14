# map_managers — 地图管理器（核心调度引擎）

> 路径: `services/internal/cores/map_managers/`  
> 文件: `map.manager.st.go` · `map.manager.var.go` · `map.options.st.go` · `manager.func.go` · `manager.tick.func.go` · `manager.march.func.go` · `manager.push.func.go` · `manager.format.func.go` · `manager.hall.func.go`

---

## 1. MapManager — 核心结构

```go
type MapManager struct {
    RoomID             uint64
    MapGroup           cores_declarations.MapGroup
    mapDataManager     *map_datas.MapDataManager
    roleConnectManager *map_connects.RoleConnectManager
    marchManage        *marchs.MarchInfoManager
    timeMarch          map[int64]map[cores_declarations.MarchID]struct{}
    timeMarchLock      sync.Mutex
    timeMap            map[int64]map[cores_declarations.MapID]struct{}
    timeMapLock        sync.Mutex
    marchDoFunc        func(*MapManager, cores_declarations.MarchID)
    marchDoFuncHandle  func(*MapManager, marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI
    mapBlock           *map_blocks.MapBlock
    opts               *options
    waitUpdateMapID    map[cores_declarations.MapID]struct{}
    waitUpdateMapLock  sync.Mutex
}
```

| Getter | 说明 |
|---|---|
| `GetMapDataManager()` | 地图数据管理器 |
| `GetMarchManage()` | 行军管理器 |
| `GetBlock()` | 地图区块 |
| `GetConf()` | 地图配置 |

---

## 2. 生命周期

### NewMapManager

构造函数，策略模式注入：
- `marchDoFunc` — 行军到达回调
- `marchDoHandleFunc` — 行军处理器工厂
- 自动创建 `roleConnectManager` 和 `mapBlock`

### Start

启动两个后台 goroutine：

```
loopTickCheck (100ms/300ms/1s 三级 tick)
    ├── 100ms ticker: 行军到期 → go marchDoFunc
    │                 + upMapAsync() 地图同步推送
    ├── 300ms ticker: clearMapFunc() 地图清理
    └── 1s ticker:    兜底处理（防止轮次丢失）

loopTickAccept
    └── TickerChan ← marchInfo → TickerAddMarch
```

**三级 tick 设计**:
- **100ms**: 精准触发到期的行军
- **300ms**: 地图到期清理
- **1s**: 遍历所有过期时间槽，兜底处理 100ms/300ms 漏掉的条目

### Stop

停止信号（关闭 `opts.stopChan`）。

---

## 3. Tick 管理

**文件**: `manager.tick.func.go`

| 方法 | 说明 |
|---|---|
| `TickerAddMarch(marchID, endTime)` | 注册行军队指定时间槽 |
| `TickerAddMap(mapID, clearTime)` | 注册地图清理 |
| `TickerAddMapList(mapIDList, clearTime)` | 批量注册 |

---

## 4. 行军 AOI 设置

**文件**: `manager.march.func.go`

| 方法 | 说明 |
|---|---|
| `MarchAOISetupSingle(marchInfo)` | 通过起点终点坐标调用 `MarchAOISetup` |
| `MarchAOISetup(marchInfo, startX, startY, endX, endY)` | 设置行军 AOI 通行 |

**AOI 设置策略**:
- **虚拟行军**: 仅路径第一格注册 `MarchAdd`（仅 AOI 通行，无战斗交互）
- **非虚拟行军**: `MovePath` 计算经过的 Screen：
  - 首尾 Screen → `MarchAdd`（驻留）
  - 中间 Screen → `PassingMarchAdd`（经过）

---

## 5. 推送系统

**文件**: `manager.push.func.go`

| 方法 | 说明 |
|---|---|
| `upMapAsync()` | 遍历 `waitUpdateMapID` → AOI 九宫格筛选可见玩家 → 格式化 PB → `PushToRoleID` 逐角色推送 |
| `UpdateMapPush(mapIDs...)` | 将地图 ID 加入待推送队列 |
| `upMarchSync(fromMapID, toMapID, marchPB, receivers...)` | 沿行军路径 AOI 收集可见玩家 + 指定接收者 → 推送 |
| `UpdateMarchPush(marchInfo)` | 格式化行军信息 → `upMarchSync` 推送 |

**upMapAsync 流程**:
1. 加锁拷贝 `waitUpdateMapID` 并清空
2. 对每个 mapID 调用 `AOI.AroundConnects` 收集可见玩家
3. 按角色分组地图 ID（角色 A 可见地图 [1,2,3]，角色 B 可见 [2,4]）
4. 批量格式化 MapInfo 为 PB
5. 逐角色组装 `PushMapInfo` → `PushToRoleID` 发送

---

## 6. 格式转换

**文件**: `manager.format.func.go`

| 方法 | 说明 |
|---|---|
| `FormatMapInfo2Pb(sliceInfo, resp)` | 地图信息 → PB（当前为空，预留扩展点） |
| `FormatMarchInfo2Pb(info)` | 行军信息 → `pb_maps_march.MarchInfo`（含队伍数据） |

**MarchInfo PB 字段**: MarchID, FromRoleID, ExecRoleID, SrcFromMapID, FromMapID, ToMapID, State, StartTime, EndTime, UnionID, TeamInfo

---

## 7. 玩家创建

**文件**: `manager.hall.func.go`

| 方法 | 说明 |
|---|---|
| `CreateRole(roleBrief)` | 创建角色位置：`GetFreeBorn()` 分配出生点 → `SetHall(data, brief)` 设大厅 → `UpdateMapPush` 推送 |

---

## 8. 对象池

**文件**: `map.manager.var.go`

```go
var mapPBPool = &sync.Pool{
    New: func() any { return nil },
}
```

- `MapPBGet()` — 从池中获取 `pb_camera.MapInfo`（调用 Reset）
- `MapPBPut(list...)` — 放回池中复用

---

## 9. 配置选项

**文件**: `map.options.st.go`

函数式选项模式（Functional Options）：

| 选项 | 默认值 |
|---|---|
| `WithStopChan(c chan struct{})` | `make(chan struct{})` |
| `WithStartTime(t int64)` | 0 |
| `WithEndTime(t int64)` | 0 |
| `WithCutNum(n int32)` | `ServerMapBlockCutNum = 25` |

---

## 设计要点

- **策略模式**: `marchDoFunc` / `marchDoHandleFunc` 通过构造函数注入，行军处理与 MapManager 解耦
- **三级定时调度**: 100ms（精准触发）+ 300ms（地图清理）+ 1s（兜底），通过时间槽 `map[int64]map[...]struct{}` 管理
- **异步 goroutine 安全**: 所有 `timeMarch` / `timeMap` / `waitUpdateMapID` 操作均有独立锁保护；行军处理采用 `go marchDoFunc` 并发执行
- **消息推送双通道**: 地图更新（`upMapAsync`）和行军更新（`UpdateMarchPush`）各自独立推送路径
