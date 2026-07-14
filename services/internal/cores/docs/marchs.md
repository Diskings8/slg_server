# marchs — 行军信息与队伍管理

> 路径: `services/internal/cores/marchs/`  
> 文件: `march.info.st.go` · `march.infomanage.st.go` · `march.infomanager.func.go` · `march.map.st.go` · `march.team.st.go`

---

## 1. MarchInfo — 行军信息

**文件**: `march.info.st.go`

实现了 `cores_declarations.MarchInfoI` 接口。

```go
type MarchInfo struct {
    RwLock          sync.RWMutex                     `gorm:"-"`
    MarchID         cores_declarations.MarchID       `gorm:"primaryKey"`
    Team            *Team                            `gorm:"type:json"`
    FromServerID    uint32
    ToServerID      uint32
    FromRoleID      uint64
    ExecRoleID      uint64
    SrcFromMapID    cores_declarations.MapID
    FromMapID       cores_declarations.MapID
    ToMapID         cores_declarations.MapID
    MarchState      pb_maps_march.MarchState
    StartTimeUx     int64
    EndTimeUx       int64
    FollowMarchID   cores_declarations.MarchID
    UnionID         uint64
    BaseMarchSpeed  uint32
    FinalMarchSpeed uint32
    ActionUse       []cores_declarations.AnyThingUse `gorm:"type:json"`
    Path            []cores_declarations.MapID       `gorm:"type:json"`
    PVPWinCount     uint32
    PVEWinCount     uint32
    VirtualData     uint64
    isVirtual       bool
    isNeedSave      atomic.Bool                      `gorm:"-"`
    isNeedDelete    atomic.Bool                      `gorm:"-"`
    saving          atomic.Bool                      `gorm:"-"`
    marchDoLocker   sync.Mutex                       `gorm:"-"`
    AoiBlock        []cores_declarations.AoiScreenI  `gorm:"-"`
    PassingAoiBlock []cores_declarations.AoiScreenI  `gorm:"-"`
}
```

### 状态标记

| 方法 | 说明 |
|---|---|
| `IsVirtual()` | 是否为虚拟行军（仅 AOI 通行，无战斗） |
| `IsNeedSave()` / `IsNeedDelete()` / `IsSaving()` | 持久化状态标记 |

### 并发控制

| 方法 | 说明 |
|---|---|
| `TryLock()` / `Unlock()` | RWMutex 尝试加写锁 |
| `LockMarchDo()` / `UnlockMarchDo()` | 行军执行专用互斥锁 |

### 数据读取

| 方法 | 说明 |
|---|---|
| `GetMarchID()` | 行军 ID |
| `GetUnionID()` | 联盟 ID |
| `GetFromMapID()` / `GetToMapID()` / `GetSrcFromMapID()` | 起止点地图 ID |
| `GetMapIDs()` | 三地图 ID 批量获取 |
| `GetMarchState()` | 行军状态 |
| `GetFromRoleID()` / `GetExecRoleID()` | 归属者/执行者 |
| `GetFromServerID()` / `GetToServerID()` | 跨服起止服务器 |
| `GetStartTimeUx()` / `GetEndTimeUx()` | 时间戳 |
| `GetMarchStartAndEndTimeUx()` | 时间戳批量获取 |
| `GetMarchTotalTimeUx()` | 总耗时 |
| `GetFollowID()` | 跟随行军 ID |
| `GetTeam()` | 队伍数据 |
| `GetActionUse()` | 行军消耗 |
| `GetTotalWinCount()` / `GetPVPWinCount()` | 战斗胜场 |
| `GetRelocationVal()` | 拆迁值（panic: implement me） |

### AOI 关联

| 方法 | 说明 |
|---|---|
| `AddAOIBlock(i)` / `AddPassingAOIBlock(i)` | 添加 AOI 屏幕关联 |
| `AoiBlock` / `PassingAoiBlock` | AOI 通行记录切片 |

### 工具方法

| 方法 | 说明 |
|---|---|
| `ClearUse()` | 清空 `ActionUse` |
| `TableName()` | 数据库表名（panic: implement me） |

---

## 2. MarchInfoManager — 行军管理器

**文件**: `march.infomanage.st.go` · `march.infomanager.func.go`

```go
type MarchInfoManager struct {
    TickerChan           chan *MarchInfo
    MarchTimeType        cores_declarations.MarchTimeType
    allMarch             map[cores_declarations.MarchID]*MarchInfo
    allMarchLock         sync.RWMutex
    allAssembleMarch     map[cores_declarations.MarchID][]*MarchInfo
    allAssembleMarchLock sync.RWMutex
    mapConfig            cores_declarations.MapConfigI
    tableName            string
    save                 atomic.Bool
}
```

| 字段 | 说明 |
|---|---|
| `TickerChan` | 行军到期通知通道 → `MapManager.loopTickAccept` |
| `allMarch` | 全局行军集合 |
| `allAssembleMarch` | 组合行军集合（跨服/合进行军） |

| 方法 | 说明 |
|---|---|
| `Init(dbc)` | 从数据库初始化：`AutoMigrate` → `Find` → 注册到 `allMarch` + 推入 `TickerChan` |
| `checkAutoMigrate(dbc)` | 自动迁移表结构 |
| `findMarchList(dbc, list)` | 查询全部行军 |
| `addMarchInfo(add)` | 注册到 `allMarch` |
| `GetConfig()` | 获取地图配置 |
| `GetTableName()` | 获取表名 |

---

## 3. MapAttribute — 地图行军属性

**文件**: `march.map.st.go`

管理**单张地图**上的行军集合和驻守队伍。

```go
type MapAttribute struct {
    assistSlice  []*MarchInfo
    assistLocker sync.RWMutex
    marchMap     hashmaps.Map[cores_declarations.MarchID, *MarchInfo]
}
```

| 方法 | 说明 |
|---|---|
| `marchAdd(mi)` | 添加行军 |
| `marchDel(marchID)` | 删除行军 |
| `GetMapMarch(container)` | 获取所有行军到容器中 |
| `RangeMapMarch(f)` | 遍历行军 |
| `GetAllMapMarchLen()` | 行军总数 |

---

## 4. Team — 行军队伍

**文件**: `march.team.st.go`

```go
type Team struct {
    Slots []*pb_battle.TeamSlotInfo
}
```

| 方法 | 说明 |
|---|---|
| `Format2Pb()` | 格式化为 `pb_battle.TeamInfo` |
| `GetAliveSoliderCount()` | 存活士兵总数（排除受伤状态的武将） |
| `GetMaxCount()` | 最大兵力 |
| `CheckCanFight()` | 检查 0 号位武将是否可战斗 |

---

## 设计要点

- **分级并发控制**: `RWMutex` 提供读写分离 + `marchDoLocker` 执行专用锁，避免行军处理影响普通读取
- **GORM JSON 序列化**: `Team` / `Path` / `ActionUse` 以 JSON 列存储
- **延迟标记**: `isNeedSave` / `isNeedDelete` 使用 `atomic.Bool` 标记状态，SaveDo 时批量处理
- **AOI 双向关联**: `AoiBlock` / `PassingAoiBlock` 与 AOI Screen 建立双向引用
- **虚拟行军**: `isVirtual` 标记区分纯路径通行和战斗交互行军
