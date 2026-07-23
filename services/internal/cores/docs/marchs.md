# marchs — 行军信息与队伍管理

> 路径: `services/internal/cores/marchs/`  
> 文件: `march.info.st.go` · `march.infomanage.st.go` · `march.infomanager.func.go` · `march.infomanager.db.func.go` · `map.attribute.st.go` · `map.attribute.func.go` · `march.team.st.go`

---

## 1. MarchInfo — 行军信息

**文件**: `march.info.st.go`

实现了 `cores_declarations.MarchInfoI` 接口。

```go
type MarchInfo struct {
    RwLock          sync.RWMutex                     `gorm:"-"`
    MarchID         cores_declarations.MarchID       `gorm:"primaryKey;COMMENT:行军ID;"`
    MarchType       cores_declarations.MarchType     `gorm:"not null;COMMENT:行军类型;"`
    Team            *Team                            `gorm:"type:json;not null;COMMENT:部队数据;"`
    FromServerID    uint32                           `gorm:"not null;COMMENT:所属服务器;"`
    ToServerID      uint32                           `gorm:"not null;COMMENT:目标服务器;"`
    FromRoleID      uint64                           `gorm:"not null;COMMENT:归属者角色ID;"`
    ExecRoleID      uint64                           `gorm:"not null;COMMENT:执行者角色ID;"`
    SrcFromMapID    cores_declarations.MapID         `gorm:"not null;COMMENT:最开始的起始地图ID;"`
    TransitMapID    cores_declarations.MapID         `gorm:"default:-1;COMMENT:本次行军实际出发地；-1 时回退到 SrcFromMapID;"`
    FromMapID       cores_declarations.MapID         `gorm:"not null;COMMENT:当前行军起始地图ID;"`
    ToMapID         cores_declarations.MapID         `gorm:"not null;COMMENT:当前行军目标地图ID;"`
    MarchState      pb_maps_march.MarchState         `gorm:"not null;COMMENT:行军状态;"`
    StartTimeUx     int64                            `gorm:"not null;COMMENT:行军开始时间;"`
    EndTimeUx       int64                            `gorm:"not null;COMMENT:行军结束时间;"`
    FollowMarchID   cores_declarations.MarchID       `gorm:"not null;COMMENT:跟随的行军;"`
    UnionID         uint64                           `gorm:"not null;COMMENT:同盟ID;"`
    BaseMarchSpeed  uint32                           `gorm:"not null;COMMENT:基础行军速度;"`
    FinalMarchSpeed uint32                           `gorm:"not null;COMMENT:最后行军速度;"`
    ActionUse       []cores_declarations.AnyThingUse `gorm:"type:json;not null;COMMENT:行军消耗;"`
    Path            []cores_declarations.MapID       `gorm:"type:json;not null;COMMENT:路线;"`
    PVPWinCount     uint32                           `gorm:"not null;COMMENT:PVP连胜数量;"`
    PVEWinCount     uint32                           `gorm:"not null;COMMENT:PVE连胜数量;"`
    VirtualData     uint64                           `gorm:"not null;COMMENT:虚拟行军数据;"`
    isVirtual       bool                             `gorm:"not null;COMMENT:是否为虚拟行军;"`
    isNeedSave      atomic.Bool                      `gorm:"-"`
    isNeedDelete    atomic.Bool                      `gorm:"-"`
    isMock          atomic.Bool                      `gorm:"-"`
    saving          atomic.Bool                      `gorm:"-"`
    marchDoLocker   sync.Mutex                       `gorm:"-"`
    AoiBlock        []cores_declarations.AoiScreenI  `gorm:"-"`
    PassingAoiBlock []cores_declarations.AoiScreenI  `gorm:"-"`
}
```

### 关键字段说明

| 字段 | 说明 |
|---|---|
| `MarchType` | 行军类型，标记行军用途（如 `MarchTypeAssist` 驻守） |
| `ExecRoleID` | 当前执行者角色 ID（可能不同于 `FromRoleID` 归属者） |
| `FinalMarchSpeed` | 最终计算后的行军速度（区别于基础速度 `BaseMarchSpeed`） |
| `VirtualData` | 虚拟行军数据（用于虚拟行军场景的附加信息） |
| `isMock` | 假行军标记，标记 `CreateMockMarch` 创建的条目不写入数据库 |

### 状态标记

| 方法 | 说明 |
|---|---|
| `IsVirtual()` | 是否为虚拟行军（仅 AOI 通行，无战斗） |
| `IsMock()` | 是否为假行军（不入库） |
| `IsMarchTypeAssist()` | 是否为驻守行军 |
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
| `GetFromMapID()` / `GetToMapID()` / `GetSrcFromMapID()` / `GetTransitMapID()` | 起止点地图 ID |
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
| `GetRelocationVal()` | 拆迁值（遍历队伍 Slot，排除受伤武将，累加拆迁属性） |

### AOI 关联

| 方法 | 说明 |
|---|---|
| `AddAOIBlock(i)` / `AddPassingAOIBlock(i)` | 添加 AOI 屏幕关联 |
| `AoiBlock` / `PassingAoiBlock` | AOI 通行记录切片 |

### 工具方法

| 方法 | 说明 |
|---|---|
| `ClearUse()` | 清空 `ActionUse` |
| `TableName()` | 数据库表名 → `"MarchInfo"` |

---

## 2. MarchInfoManager — 行军管理器

**文件**: `march.infomanage.st.go` · `march.infomanager.func.go` · `march.infomanager.db.func.go`

```go
type MarchInfoManager struct {
    TickerChan           chan *MarchInfo
    MarchTimeType        cores_declarations.MarchTimeType
    allMarch             map[cores_declarations.MarchID]*MarchInfo
    allMarchLock         sync.RWMutex
    allAssembleMarch     map[cores_declarations.MarchID][]*MarchInfo
    allAssembleMarchLock sync.RWMutex
    mapConfig            cores_declarations.MapConfigI
    mapMarch             []MapAttribute
    tableName            string
    saving               atomic.Bool
}
```

| 字段 | 说明 |
|---|---|
| `TickerChan` | 行军到期通知通道 → `MapManager.loopTickAccept` |
| `allMarch` | 全局行军集合 |
| `allAssembleMarch` | 组合行军集合（跨服/合进行军） |
| `mapMarch` | 地图行军属性列表（`MapAttribute[]`，每张地图一个） |

### 构造与初始化

| 方法 | 说明 |
|---|---|
| `New(tickerChan, tableName, mapConfig, marchTimeType)` | 构造函数，创建管理器并初始化 `mapMarch` 切片 |
| `Init(dbc)` | 从数据库初始化：`AutoMigrate` → `Find` → 挂载到 `MapAttribute` → 恢复驻守 → 推入 `TickerChan` |

### 数据库持久化

| 方法 | 说明 |
|---|---|
| `checkAutoMigrate(dbc)` | 自动迁移表结构 |
| `findMarchList(dbc, list)` | 查询全部行军 |
| `Save(marchInfo)` | 标记行军为待保存 → 触发 `EntitySaveFunc` |
| `SaveDo()` | 定期批量保存所有标记为 `isNeedSave` 的行军（`dbconn.MaxSaveLen` 分批写入） |

### MapAttribute 管理

| 方法 | 说明 |
|---|---|
| `MapAttributeGet(mapID)` | 获取指定地图的行军属性 |
| `MapAttributeMarchCreate(marchInfo)` | 创建行军时将行军挂载到起止点地图（from + to + srcFrom） |
| `MapAttributeMarchDelete(marchInfo)` | 从起止点地图移除行军 |
| `MapAttributeMarchChange(marchInfo, newMapID)` | 修改行军目标位置（旧 from 移除 → from 变 to → to 改新目标） |
| `MapAttributeMarchModToMapID(marchInfo, newToMapID)` | 仅修改行军目标地图 ID |
| `MapAttributeMarchModFormMapID(marchInfo, newMapID, isAllForm)` | 修改行军起始地图（支持是否同步修改 FromMapID） |
| `MapAttributeMarchCallBack(marchInfo)` | 行军返回处理（from 移除 → toMap 添加，toMap 已由调用方设为 TransitMapID/SrcFromMapID） |

### 行军 CRUD

| 方法 | 说明 |
|---|---|
| `CreateMarch(marchInfo)` | 创建行军（入库 + 注册到 allMarch + MapAttribute 挂载） |
| `CreateMarchInBatches(marchInfoList...)` | 批量创建行军（每批 20 条） |
| `CreateMockMarch(marchInfo)` | 创建假行军（仅注册，不入库，设置 `isMock = true`） |
| `DeleteMarch(marchInfo)` | 删除行军（入库删除 + isNeedDelete 标记 + AOI 清理 + MapAttribute 移除） |
| `AllMarch()` | 返回全部行军列表 |
| `GetMarchInfo(marchID)` | 单条行军查询 |
| `GetMarchInfoByType(marchTypes...)` | 按行军类型筛选 |

### 辅助接口

实现了 `common_declarations.AsyncSaveEntityI`：
- `IsDelete()` → `false`
- `Tag()` → `"MarchInfoManager"`
- `Saving()` → `&mm.saving`

---

## 3. MapAttribute — 地图行军属性

**文件**: `map.attribute.st.go` · `map.attribute.func.go`

管理**单张地图**上的行军集合和驻守队伍。

```go
type MapAttribute struct {
    assistSlice  []*MarchInfo
    assistLocker sync.RWMutex
    marchMap     hashmaps.Map[cores_declarations.MarchID, *MarchInfo]
}
```

### 行军管理

| 方法 | 说明 |
|---|---|
| `marchAdd(mi)` | 添加行军到集合 |
| `marchDel(marchID)` | 从集合删除行军 |
| `GetMapMarch(container)` | 获取所有行军到容器中 |
| `RangeMapMarch(f)` | 遍历行军 |
| `GetAllMapMarchLen()` | 行军总数 |

### 驻守管理

| 方法 | 说明 |
|---|---|
| `Assist(assistSlice)` | 返回驻守队伍列表（读锁 + `slices.Clone` 拷贝） |
| `AssistRoleMap(out)` | 返回驻守队伍的角色 ID → MarchInfo 映射 |
| `AssistLen()` | 驻守队伍数量 |
| `AssistRoleID()` | 驻守队伍的角色 ID 列表 |
| `AssistArrive(m)` | 驻守到达（去重后追加到 `assistSlice`） |
| `AssistCallBack(marchID)` | 驻守返回（从 `assistSlice` 删除） |
| `RangeAssist(f)` | 遍历驻守队伍 |

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
- **延迟标记 + 批量持久化**: `isNeedSave` / `isNeedDelete` 使用 `atomic.Bool` 标记状态，`SaveDo()` 分批处理
- **AOI 双向关联**: `AoiBlock` / `PassingAoiBlock` 与 AOI Screen 建立双向引用
- **MapAttribute 全局索引**: `mapMarch []MapAttribute` 以数组下标对应 MapID 实现 O(1) 访问，行军创建/删除时自动维护多地图双向索引
- **驻守管理**: `assistSlice` 搭配读写锁，支持驻守到达/返回/遍历/查询，与行军创建联动（`Init` 中自动恢复驻守状态）
- **虚拟行军**: `isVirtual` 标记区分纯路径通行和战斗交互行军
- **Mock 行军**: `isMock` 标记 + `CreateMockMarch` 实现测试/预告类行军，不写入数据库
