# cores_declarations — 核心声明

> 路径: `services/internal/cores/cores_declarations/`  
> 文件: `core.const.go` · `core.if.go` · `core.st.go`

所有核心包共享的基础类型与接口定义，作为全局类型中心。

---

## 基础类型

| 类型 | 说明 |
|---|---|
| `MarchID` (uint64) | 行军唯一 ID |
| `MapID` (int32) | 地图格子 ID |
| `ScreenID` (int32) | AOI 屏幕格子 ID |
| `BornBlockID` (int32) | 出生块 ID |
| `MarchTimeType` (int) | 行军时间类型 |
| `MarchType` (uint32) | 行军类型（如 110101） |
| `MapGroup` (uint32) | 地图分组 |
| `ScaleLevel` (int) | 屏幕缩放等级 |
| `RoleMainCityState` (int) | 玩家主城状态（Normal / Portable） |
| `MapLevel` (int) | 地图等级 |
| `ElementType` (int) | 地图元素类型 |

---

## 关键常量

| 常量 | 值 | 说明 |
|---|---|---|
| `ServerMapBlockCutNum` | 25 | 地图切块总数 |
| `ServerMapBlockRowCutNum` | 5 | 每行切块数 |
| `ScreenWeight` | 40 | 屏幕格子宽度（AOI 基本单位） |
| `HallLandCover` | 3 | 玩家城边长 |
| `HallCoverCount` | 9 | 玩家城占地格数 |
| `Land1CoverBaseKey` | 1 | 1x1 主位置键索引 |
| `Land3CoverBaseKey` | 4 | 3x3 主位置键索引 |
| `RoleMainCityStateNormalCoverCount` | 9 | 普通主城占用格数 |
| `RoleMainCityStatePortableCoverCount` | 1 | 便携主城占用格数 |
| `TeamSlot_1` ~ `TeamSlot_3` | 1~3 | 队伍槽位编号 |
| `InvalidMapID` | -1 | 无效地图 ID 标记 |

### ElementType 枚举

| 常量 | 说明 |
|---|---|
| `ElementType_None` | 无元素 |
| `ElementType_Resources_1` ~ `_4` | 资源 1~4 |
| `ElementType_Terrain_1` | 地形——山 |
| `ElementType_Terrain_2` | 地形——水 |

### 辅助方法

- `ElementType.IsCantBornUse()` — 检查是否不可用于出生（`Terrain_1`/`Terrain_2` 以外返回 true）

---

## 接口定义

| 接口 | 方法签名 | 说明 |
|---|---|---|
| `AoiScreenI` | 标记接口 | AOI 屏幕标记 |
| `MarchHeroI` | 标记接口 | 行军武将标记 |
| `MarchSoldierI` | `GetCurCount()`, `GetMaxCount()`, `GetInjuredCount()` | 行军士兵接口 |
| `MarchInfoI` | `GetMarchID()`, `GetUnionID()`, `AddPassingAOIBlock(AoiScreenI)`, `AddAOIBlock(AoiScreenI)`, `GetRelocationVal()` | 行军信息接口 |
| `MarchDoFuncHandleI` | `Do()`, `LockDo()`, `CallBack()`, `CallBackNow()`, `Lock()`, `Unlock()`, `Leave()` | 行军执行处理接口，定义完整生命周期 |
| `MapRoleConnectI` | `GetRoleID()`, `GetScreenMapID()`, `SetScreenMapID(MapID)`, `Send(*pb_common.NodePacket)` | 角色连接接口 |
| `BornBlockI` | `Store(BornBlockID, map[int32]struct{})`, `Load(BornBlockID)`, `Use(BornBlockID)`, `Free(BornBlockID)`, `Delete(BornBlockID)`, `Range(f)` | 出生块管理器接口 |
| `MapConfigI` | `MapCount()`, `MapScope()`, `MapID2XY(MapID)`, `XY2MapID(int32, int32)`, `SortByDis(MapID, []MapID)`, `CoverMapIDs(int32, int, any)` | 地图配置接口 |
| `BaseBuildingsConfI` | `GetBuildingsMaxHp(uint32, uint32)`, `GetBuildingsMaxLevel()` | 建筑配置接口 |
| `NpcBuildingsConfI` | 继承 `BaseBuildingsConfI` | NPC 建筑配置接口 |

---

## 结构体

- **`AnyThingUse`** — 通用 KV 容器（`K uint32`, `V uint64`），用于存储行军动作消耗等键值对数据

---

## 设计要点

- 所有类型和方法集中在声明包中，避免循环依赖
- `MarchDoFuncHandleI` 是行军执行器的核心接口，定义了加锁/执行/回调/离开的完整生命周期
- `BornBlockI` 采用接口抽象，支持大地图和大活动地图两种不同实现
- `MapConfigI` 是地图配置的抽象层，屏蔽具体地图尺寸和坐标转换逻辑
