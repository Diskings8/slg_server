# map_datas — 地图格子数据（核心数据层）

> 路径: `services/internal/cores/map_datas/`  
> 文件: `map.info.st.go` · `map.datamanager.st.go` · `map.datamanager.func.go` · `datamanager.born.func.go` · `map.union.func.go`  
> 子包: `map_buildings/` · `map_events/`

---

## 1. MapInfo — 单格信息

**文件**: `map.info.st.go`

```go
type MapInfo struct {
    rwLock           sync.RWMutex
    mapID            cores_declarations.MapID
    coreMapID        cores_declarations.MapID
    x, y             int
    serverID         uint32
    ownerID          uint64
    Level            cores_declarations.MapLevel
    configID         uint32
    ElementType      cores_declarations.ElementType
    protectedEndTime int64
    overlayEvent     *map_events.OverlayEvent
    overlayBuilding  *map_buildings.OverlayBuilding
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `mapID` | `MapID` | 格子唯一 ID |
| `coreMapID` | `MapID` | 核心/基础 MapID（多格归属时指向核心格，如主城 9 格的中心） |
| `x, y` | `int` | 格子坐标 |
| `serverID` | `uint32` | 归属服务器 ID |
| `ownerID` | `uint64` | 归属角色 ID |
| `Level` | `MapLevel` | 格子等级 |
| `configID` | `uint32` | 配置 ID |
| `ElementType` | `ElementType` | 元素类型 |
| `protectedEndTime` | `int64` | 保护到期时间戳 |
| `overlayEvent` | `*OverlayEvent` | 叠加的事件 |
| `overlayBuilding` | `*OverlayBuilding` | 叠加的建筑 |

| Getter 方法 | 说明 |
|---|---|
| `GetMapID()` / `GetBaseMapID()` | 格子 ID 和核心格 ID |
| `GetPointX()` / `GetPointY()` | 坐标 |
| `GetServerID()` | 归属服务器 |
| `GetOwnerID()` | 归属角色 ID（读锁） |
| `GetLevel()` | 等级（读锁） |
| `GetElementID()` / `GetElementType()` | 元素配置 ID 和类型（读锁） |
| `GetOverlayBuilding()` | 叠加建筑（读锁） |

| 字段操作方法 | 说明 |
|---|---|
| `Occupy(ownerID)` | 设置地块占领者（调用方需持有写锁） |
| `Free(now)` | 释放地块（重置 ownerID，设置保护时间） |

| 并发控制 | 说明 |
|---|---|
| `TryLock()` / `Unlock()` | 尝试加写锁（非阻塞） |
| `Lock()` / `Unlock()` | 加写锁（阻塞） |
| `LockMarchDo()` / `UnlockMarchDo()` | 行军执行专用互斥锁 |

---

## 2. MapDataManager — 地图数据管理器

**文件**: `map.datamanager.st.go` · `map.datamanager.func.go`

```go
type MapDataManager struct {
    Id        uint64
    waitSave  hashmaps.Map[cores_declarations.MapID, *MapInfo]
    config    cores_declarations.MapConfigI
    tableName string
    saving    atomic.Bool
    AOI       *map_aois.ScreenData
    BornAts   cores_declarations.BornBlockI
    mapData   []MapInfo
}
```

### 初始化与查询

| 方法 | 说明 |
|---|---|
| `Init(mapD)` | 初始化，遍历有效格子注册到 AOI |
| `GetMapInfo(mapID)` | 按 MapID 获取格子指针 + 有效性标志 |
| `GetMapInfoSlice(mapIDs)` | 批量获取 |
| `Range(f)` | 遍历所有有效格子 |
| `GetConfig()` | 获取地图配置 |

**Init 流程**:
1. 保存 `mapData` 切片引用
2. 遍历所有格子，忽略 `InvalidMapID`
3. 对 `coreMapID == mapID` 的格子调用 `AOI.MapDataAdd` 注册

### 并发控制

| 方法 | 说明 |
|---|---|
| `TryLock(mapList)` | 批量加锁，任一失败自动回滚已加的锁 |
| `Unlock(mapList)` | 批量解锁（自动去重） |

**LockMapSlice** 辅助类型：
```go
type LockMapSlice struct {
    data []*MapInfo
    mdm  *MapDataManager
}
func (l LockMapSlice) Unlock() // 自动解锁
func (l LockMapSlice) Data() []*MapInfo // 获取数据
```

### 数据持久化

| 方法 | 说明 |
|---|---|
| `Save(list...)` | 将格子存入 `waitSave` 队列 |
| `SaveDo()` | 定期保存（当前为空实现） |
| `Clear(mapIDs)` | 清理指定格子（panic: implement me） |

### 主城设置与出生点

| 方法 | 说明 |
|---|---|
| `SetRoleMainCity(roleCityState, dataSlice, roleBrief)` | 设置玩家主城，校验格子数 + 地块合法性 |
| `SetHall(data, brief)` | 设置玩家大厅（panic: implement me） |
| `GetFreeBorn()` | 从空闲出生块中查找可用空地，自动上锁并返回 |

**GetFreeBorn 详细流程**:
1. 遍历 `BornAts.Range`（按 `blockSort` 优先级）
2. 对每个出生块 ID，调用 `CoverMapIDs(bornID, 1, HallLandCover/2)` 检查 9 格完整性
3. 批量加锁 `mapSlice`
4. 调用 `CheckRoleBornSiteSafeByMapInfos` 检测地形合法性
5. 成功 → `Use(bornID)` 移入使用池，返回 `LockMapSlice` + `freeBornFunc`
6. 失败 → `Delete(bornID)` 移除该出生块
7. 全部遍历完仍无结果 → 返回"没有空余位置"错误

---

## 3. UnionMemberMapIDs — 联盟成员地图索引

**文件**: `map.union.func.go`

```go
type UnionMemberMapIDs struct {
    unionRoleMapID map[uint64]map[uint64]cores_declarations.MapID
    roleUnionID    map[uint64]uint64
    roleMap        map[uint64]cores_declarations.MapID
    locker         sync.RWMutex
}
```

维护联盟 ID → 角色 ID → 地图格子的三元映射，提供 O(1) 的角色归属查询。

| 方法 | 说明 |
|---|---|
| `Set(unionID, roleID, mapID)` | 设置角色所属联盟和地图位置 |
| `SetUnionID(roleID, unionID)` | 更新角色联盟归属 |
| `SetMapID(roleID, mapID)` | 更新角色地图位置 |
| `Remove(roleID)` | 移除角色（退出/删号） |
| `GetUnionRoleMapIDs(unionID)` | 获取联盟所有成员的地图位置（副本） |
| `GetUnionRoleIDs(unionID)` | 获取联盟所有成员 ID |
| `GetRoleUnionID(roleID)` | 查询角色所属联盟 |
| `GetRoleMapID(roleID)` | 查询角色所在地图 |
| `Len()` | 总角色数 |

---

## 4. CheckRoleBornSiteSafeByMapInfos — 出生点安全性校验

**文件**: `datamanager.born.func.go`

```go
func CheckRoleBornSiteSafeByMapInfos(needLock bool, mapInfos ...*MapInfo) bool
```

- 对每个格子执行 `ElementType.IsCantBornUse()` 检查
- 支持 `needLock` 参数（是否需要在读锁保护下读取 ElementType）
- 支持自定义 `checkFunc` 扩展校验逻辑

---

## 4. 子包 — 建筑系统 (map_buildings)

### BaseBuildings — 建筑基础模块

**文件**: `base.building.st.go`

```go
type BaseBuildings struct {
    BuildingsType          pb_city.BuildingType
    BuildingsMaxHp         uint64
    BuildingsCurHp         uint64
    BuildingsConfID        uint32
    BuildingsLevel         uint32
    BuildingsRecoverHpTime int64
    BuildingsConf          cores_declarations.BaseBuildingsConfI
    buildingsRWLock        sync.RWMutex
}
```

| 方法 | 说明 |
|---|---|
| `NewBaseBuildings(confID, curLv, conf)` | 构造函数，初始满血 |
| `LevelUp()` | 升级后重置恢复时间 |
| `AddBuildingsHp(add)` → `(right uint64)` | 加血，返回实际生效值 |
| `ReduceBuildingsHp(reduce)` → `(right uint64, isBroken bool)` | 扣血，返回实际值和是否损毁 |
| `BeAttack(info)` → `(right uint64, isBroken bool)` | 被攻击（根据 `GetRelocationVal` 扣血） |

**血量逻辑**:
- `AddBuildingsHp`: 当前血量未满时增加，不超过最大值
- `ReduceBuildingsHp`: 当前血量不足时置为 0 并标记损毁

### NpcBuilding — NPC 建筑

**文件**: `npc.building.st.go`

```go
type NpcBuilding struct {
    BaseBuildings
}

func (nb *NpcBuilding) BeforeBeAttack(cores_declarations.MarchInfoI) bool
```

- 嵌入 `BaseBuildings`
- `BeforeBeAttack` — 攻击前回调（当前直接返回 true）

### NpcCity — NPC 城市

**文件**: `npc.city.st.go`

```go
type NpcCity struct {
    NpcBuilding
    ID             uint32
    CurOccUnionID  uint64
    FirstOccRecord *pb_city.CityFirstOccRecord
    CityGarrison   []cores_declarations.MarchInfoI
}
```

| 字段 | 说明 |
|---|---|
| `ID` | 城市 ID |
| `CurOccUnionID` | 当前占领联盟 |
| `FirstOccRecord` | 首占记录 |
| `CityGarrison` | 城市驻军列表 |

---

## 5. 子包 — 事件 (map_events)

**文件**: `map.event.st.go`

- `OverlayEvent` — 空结构体（事件覆盖层占位，待实现）

---

## 设计要点

- **批量加锁/回滚**: `TryLock` 遍历加锁，失败时回滚已锁的格子，避免死锁
- **LockMapSlice**: RAII 风格的自动解锁封装，确保调用链中不会遗漏解锁
- **出生点分配管道**: 出生块 → CoverMapIDs → 加锁 → 地形校验 → Use/Free，完整的事务式生命周期
- **建筑血量三层接口**: `Add`（治疗）→ `Reduce`（伤害）→ `BeAttack`（业务层），职责清晰
- **GORM JSON 序列化**: `Path`/`Team`/`ActionUse` 以 JSON 列存储
