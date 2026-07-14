# map_aois — AOI 视野管理

> 路径: `services/internal/cores/map_aois/`  
> 文件: `aoi.screen.st.go` · `data.screen.st.go` · `data.screen.func.go`

AOI（Area of Interest）系统管理玩家的视野屏幕格。采用网格划分+九宫格缓存设计，将地图按 `ScreenWeight=40` 切分为等大的 Screen 格子。

---

## Screen[T] — 泛型 AOI 屏幕

```go
type Screen[T cores_declarations.ScreenID] struct {
    ID              T
    connect         hashmaps.Map[uint64, cores_declarations.MapRoleConnectI]
    around          *atomic.Pointer[[]*Screen[T]]
    allMarch        hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
    allPassingMarch hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `connect` | `hashmaps.Map` | 视野内角色连接集合 |
| `around` | `*atomic.Pointer` | 九宫格缓存（周围 8 个邻居 + 自身） |
| `allMarch` | `hashmaps.Map` | 驻留本格的行军 |
| `allPassingMarch` | `hashmaps.Map` | 经过本格的行军 |

### 方法

| 方法 | 说明 |
|---|---|
| `MarchAdd(info)` / `MarchDelete(info)` | 添加/删除行军，自动回调 `info.AddAOIBlock(s)` |
| `MarchRange(f)` | 遍历驻留行军 |
| `PassingMarchAdd(info)` / `PassingMarchDelete(info)` | 添加/删除经过行军，自动回调 `info.AddPassingAOIBlock(s)` |
| `PassingMarchRange(f)` | 遍历经过行军 |
| `Connects(connects)` | 获取视野内角色连接（复用参数优化 GC） |
| `ConnectRoleIDs(connects)` | 获取视野内角色 ID 列表 |

---

## ScreenData — AOI 屏幕数据管理器

```go
type ScreenData struct {
    data            []*Screen[cores_declarations.ScreenID]
    mapConf         cores_declarations.MapConfigI
    mapScope        int32      // 地图每行格子数
    screenScopeHalf int32      // ScreenWeight
    scopeCount      int32      // 一行 Screen 数量
    count           int32      // Screen 总数
}
```

### 构造函数

`NewAoi(mapConfig)` — 按 `MapScope / ScreenWeight` 计算网格总数，初始化全部 Screen 实例。

### Screen 查询

| 方法 | 说明 |
|---|---|
| `GetScreenIDByMapID(mapID)` | MapID → ScreenID 映射 |
| `GetScreenByMapID(mapID)` | MapID → Screen 指针 |
| `GetScreenByScreenID(screenID)` | ScreenID → Screen 指针 |
| `MapID2ScreenID(mapID)` | 同 `GetScreenIDByMapID` |
| `XY2ScreenID(x, y)` | 坐标 → ScreenID |

### 九宫格

| 方法 | 说明 |
|---|---|
| `AroundByScreen(screen)` | 计算九宫格（上下左右 + 四角），首次计算后缓存至 `atomic.Pointer` |
| `Around(mapID)` | 按 MapID 取九宫格 |
| `Cover(mapID, cover)` | 以 mapID 为中心按半径取覆盖范围 |

**边界处理**:
- 最左列（`useScreenID % scopeCount == 1`）：屏蔽左侧邻居
- 最右列（`useScreenID % scopeCount == 0`）：屏蔽右侧邻居
- 越界位置填入 `nil`，保证始终返回 9 个元素

### 视野移动

| 方法 | 说明 |
|---|---|
| `Move(conn, newMapID)` | 玩家视野移动：同格不处理，不同格则从旧 Screen 删除并注册到新 Screen |
| `MovePath(startX, startY, endX, endY, path)` | 沿直线以 `step=ScreenWeight*2` 跳跃计算经过的所有 Screen |

**MovePath 算法**:
1. 计算起点 ScreenID 加入路径
2. 计算两点距离，以 `step=ScreenWeight*2` 逐步采样中间点
3. 若起点终点不同格则加入终点 Screen
4. 用于行军 AOI 设置，区分首尾（`MarchAdd`）和中间（`PassingMarchAdd`）

### 数据注册

| 方法 | 说明 |
|---|---|
| `MapDataAdd(mapID)` | 注册地图数据到 AOI（调用 `GetScreenByMapID`） |
| `ScreenIDs2MapIDs(screenIDs, mapIDs)` | Screen ID 列表 → 展开为所有 MapID |
| `GetMapIDFirstByScreenID(screenID)` | Screen 左上角第一个 MapID |
| `ScreenMapLen()` | 单个 Screen 包含的格子数量 |

### 连接查询

| 方法 | 说明 |
|---|---|
| `GetConnects(mapID, connects)` | 获取单格内玩家连接 |
| `AroundConnects(mapID, connects)` | 获取九宫格内所有玩家连接（自动去重） |

### 退出

`Exit(roleConn)` — 从当前 Screen 的 connect 集合中删除角色。

---

## 设计要点

- **九宫格缓存**: `atomic.Pointer` 缓存计算结果，一次计算永久复用
- **虚拟行军 vs 非虚拟行军**: `MovePath` 区分首尾驻留行军和中间经过行军，`Screen.MarchAdd` 和 `Screen.PassingMarchAdd` 分别处理，与行军信息建立双向 AOI 关联
- **GC 优化**: `Connects(connects)` 和 `ConnectRoleIDs(connects)` 接受外部参数复用，减少内存分配
- **nil 安全**: 所有 Screen 方法在接收者为 nil 时直接返回，避免空指针 panic
