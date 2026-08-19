# marchdos — 行军执行器

> 路径: `services/internal/cores/marchdos/`  
> 文件: `base.march.st.go` · `single.march.st.go` · `multi.march.st.go` · `march.factory.go` · `march.tick.func.go`  
> 子包: `attack_march/` · `assist_march/` · `sweep_march/` · `strategy_march/`

行军到达目的地后执行具体动作的模块，采用**模板方法模式 + 注册式工厂**。

---

## 1. BaseMarch — 基类

```go
type BaseMarch struct {
    marchManage   *marchs.MarchInfoManager
    mgr           *map_managers.MapManager
    fromMapInfo   *map_datas.MapInfo
    toMapInfo     *map_datas.MapInfo
    marchLockOk   bool
    fromMapLockOk bool
    toMapLockOk   bool
    hadInit       bool
    err           error
    prepareOpts   []func(*map_managers.MapManager)
    doOpts        []func(*map_managers.MapManager)
    finishOpts    []func(*map_managers.MapManager)

    // 召回（CallBack）操作链
    prepareCallBackOpts    []func(*map_managers.MapManager)
    callBackOpts           []func(*map_managers.MapManager)
    finishCallBackOpts     []func(*map_managers.MapManager)
    prepareCallBackNowOpts []func(*map_managers.MapManager)
    callBackNowOpts        []func(*map_managers.MapManager)
    finishCallBackNowOpts  []func(*map_managers.MapManager)

    // 召回到达（BackArrive）操作链
    prepareBackArriveOpts []func(*map_managers.MapManager)
    backArriveOpts        []func(*map_managers.MapManager)
    finishBackArriveOpts  []func(*map_managers.MapManager)
}
```

### 三阶段生命周期

```
prepareOpts ──→ doOpts ──→ finishOpts
  (准备)        (执行)      (完成)
```

| 方法 | 说明 |
|---|---|
| `AddPrepareOpt(f)` | 添加准备阶段处理函数（FIFO） |
| `AddDoOpt(f)` | 添加执行阶段处理函数（FIFO） |
| `AddFinishOpt(f)` | 添加完成阶段处理函数（FIFO） |
| `Init()` | 设置三个阶段的空回调，标记 `hadInit = true` |
| `Do() error` | 按序执行三个阶段（实现 `MarchDoFuncHandleI`） |
| `DoWithManager(mm)` | 显式传入管理器执行（兼容内部调用） |
| `SetManager(mm)` | 设置地图管理器 |
| `SetBase(mm)` | 统一设置管理器依赖（mgr + marchManage） |

### CallBack / CallBackNow / BackArrive 模板方法

BaseMarch 对召回和召回到达也采用模板方法模式：

| 方法 | 操作链 | 说明 |
|---|---|---|
| `CallBack()` | prepare → callBack → finish | 召回行军 |
| `CallBackNow()` | prepare → callBackNow → finish | 立即召回 |
| `BackArrive()` | prepare → backArrive → finish | 召回到达处理 |

子类通过 `AddPrepareCallBackOpt` / `AddCallBackOpt` / `AddFinishCallBackOpt` 注入自定义逻辑。

---

## 2. SingleMarch — 单一行军执行器

```go
type SingleMarch struct {
    BaseMarch
    single          *marchs.MarchInfo
    arriveAfterFunc func(*map_managers.MapManager, *marchs.MarchInfo)
}
```

### 状态分流 Do

`SingleMarch.Do()` 根据行军状态分流：

```
MarchState_Back → BackArrive()  (召回到达：清理 + 推送 + 删除行军)
其他状态       → BaseMarch.Do() (正常到达：prepare → do → finish)
```

### CallBack 默认行为

`SingleMarch.Init()` 注册了默认的召回回调链：

```
callBackOpt → callbackSwapDirection(mgr)
```

`callbackSwapDirection` 核心操作：
1. 等比例重算返回时间（走了多久就需多久返回）
2. 交换 From/To 方向
3. 更新 `MapAttribute` 索引
4. 设置 `MarchState = Back`
5. 重算 AOI 路径（清除旧路径，建立返回路径）
6. 重新注册 ticker

`callbackNowInstantReturn` 核心操作：
1. 交换 From/To 方向
2. 设置 `EndTimeUx = now`，tick 立即触发到达处理

### 额外方法

| 方法 | 说明 |
|---|---|
| `CallBackToSrcPoint()` | 强制召回行军到 `SrcFromMapID`（无视 `TransitMapID`） |
| `CallBackNowToSrcPoint()` | 强制立即召回行军到 `SrcFromMapID` |
| `ReTry()` | 召回重试（最多 3 次，间隔 100ms） |
| `BackArrive()` | 召回到达处理（推送 → 删除行军） |
| `MarchInfo()` | 返回行军信息 |
| `GetFromMapInfo()` / `GetToMapInfo()` | 返回来源/目标地块 |

### 分层加锁

`TryLock(marchLock, fromLock, toLock)` → 按序加锁，失败回滚：

```
Step 1: marchLock? → TryLock MarchInfo
                    → 失败 → unlock(), return false
Step 2: fromLock?  → TryLock fromMapInfo
                    → 失败 → unlock(), return false
Step 3: toLock?    → TryLock toMapInfo
                    → 失败 → unlock(), return false
```

`unlock()` → 根据标志位分层解锁（确保只解锁已加锁的层级）。

---

## 3. MultiMarch — 多行军执行器

```go
type MultiMarch struct {
    BaseMarch
    multi           []*marchs.MarchInfo
    markOff         int32
    marchLen        int
    arriveAfterFunc func(*map_managers.MapManager, []*marchs.MarchInfo)
}
```

### 位掩码加锁

`TryLock(marchLock, fromLock, toLock)`：

- 使用 `markOff` 位掩码（`1 << inx`）逐条锁定多条行军
- 任一加锁失败 → 按 `markOff` 位逐项解锁回滚
- 成功后将 `markOff` 置零

```go
markOff |= 1 << inx   // 成功锁定第 inx 条
markOff & 1<<i != 0   // 检查第 i 条是否已锁
```

### Init

在 prepare 阶段自动调用 `SetArriveAfterFunc`。

| 方法 | 说明 |
|---|---|
| `SetArriveAfterFunc(manager, multi)` | 设置到达回调（当前为空实现） |

---

## 4. 注册式工厂

**文件**: `march.factory.go`

```go
var marchFactories = map[MarchType]func(*MapManager, *MarchInfo) MarchDoFuncHandleI{}

func RegisterMarchFactory(mt MarchType, factory func(...) MarchDoFuncHandleI)
func NewMarchDo(mm, marchInfo) MarchDoFuncHandleI
```

各子包在 `init()` 中注册自己：

```go
// attack_march/march.func.go
func init() {
    marchdos.RegisterMarchFactory(MarchTypeAttack, New)
}
```

已注册类型：

| MarchType | 值 | 子包 | 说明 |
|---|---|---|---|
| `MarchTypeAttack` | 10001 | `attack_march/` | 攻击行军：校验 → 战斗 → 占领 → 推送 |
| `MarchTypeAssist` | 10002 | `assist_march/` | 驻守行军：注册驻军 |
| `MarchTypeSweep` | 10003 | `sweep_march/` | 扫荡行军：采集资源（⏸ 待 game 侧新机制） |
| `MarchTypeStrategy` | 10004 | `strategy_march/` | 计略行军 |
| `MarchTypeDevelop` | 10005 | `develop_march/` | 开发行军：打守军 → 升级地块 |

---

## 5. AttackMarch — 攻击行军（P0-2 战斗到达流水线）

**文件**: `attack_march/march.func.go`

攻击行军到达流水线（战斗**计算**迁移到独立 battle 节点，本层负责组装请求 + 应用结果）：

```
Prepare → checkTargetLegality()
    ├─ 校验通过 → continue
    └─ 校验失败 → DefeatRecall()（反转方向返回 TransitMapID）

Do → buildBattleSettleReq()            ← 组装攻守快照（依赖 MapManager 构建防守方）
    ├─ mgr.GetBattleSettleFunc()(req)  ← 调 battle 节点 RPC（worldmap 注入回调）
    │    ├─ 回调未注入 / RPC 失败 → DefeatRecall()（兜底召回）
    │    └─ 成功 → rsp（BattleSettleRsp）
    ├─ applyBattleSettleRsp()
    │    ├─ applyAttackerCasualties()  ← 最后一轮攻方快照覆盖 MarchInfo.Team
    │    ├─ updateAttackerState()      ← 胜→Stay；败→仅重置 PVPWinCount（状态由 DefeatRecall 处理）
    │    ├─ processDefeatedDefenders() ← 防守方伤亡写回 + 回城 + AssistCallBack + push
    │    ├─ applyBuildingDamage()      ← ReduceBuildingsHp 回写
    │    ├─ tryOccupy()                ← 地块转属
    │    └─ updateWinStreak()
    └─ 战败（!rsp.AttackerWin）→ DefeatRecall()
         ├─ 交换 From/To，To 路由到 TransitMapID（-1 回退 SrcFromMapID）
         ├─ 等比例重算返回时间 + TickerAddMarch 重新入队
         └─ MarchState = Back → 回城到站时 BackArrive() 清理

Finish → pushBattleResult() + triggerBattleEvents()
```

> 战败召回 `DefeatRecall()` 复用 `callbackSwapDirection`（`single.march.st.go`），调用方需已持有行军锁。
> `SingleMarch` / `MultiMarch` 的 `BackArrive()` 均已改为不自行加锁（`Do()` 时锁已被 `handle.Lock` 持有），修复了回城到站路径的双重加锁问题。
> BACKARRIVED 事件时机：worldmap 侧在行军返回途中不发事件，真正回城到站（BackArrive 已删除行军）才发。

### 战斗结算结果

本地运行时结构 `BattleResult`/`LayerResult`（`march.result.st.go`）已随战斗计算迁移而**删除**，
结果统一为 battle 节点返回的 `pb_battle.BattleSettleRsp`：

- `results` — 逐层 `OneBattleResult`（攻守双方 `BattleSide` + 是否占领）
- `attacker_win` / `occupied` — 攻击方获胜 / 占领判定
- `defeated_defenders` — 被击败防守方 march_id 列表（cores 据此召回）
- `building_damage` — 攻城伤害（cores 就地扣减建筑耐久）

结算回调通过 `cores_declarations.BattleSettleFunc` 注入 `MapManager`（`SetBattleSettleFunc`），
cores 保持纯逻辑包，不依赖 rpcconn；battle 节点实现见 `services/battle/BATTLE_OVERVIEW.md`。

---

## 6. DevelopMarch — 开发行军（格子争夺：升级地块）

**文件**: `develop_march/march.func.go`

开发行军（10005）语义：**派队 + 打地块守军 → 胜利后地块升级 +3 级、标记已开发（只升级不占归属）**。

```
Prepare → checkDevelopable()
    ├─ 目标为自己的地（ownerID == fromRoleID）
    ├─ 目标等级（当前+3）有守军配置（gameconfig.json 的 guard 表）→ 可开发
    └─ 不合法 → DefeatRecall()

Do → buildDevelopSettleReq()           ← 守军构造成 DefenderGroup（NPC role_id=0，仅一层）
    ├─ mgr.GetBattleSettleFunc()(req)  ← 复用 battle 节点（PvE：无建筑 → occupied=true）
    │    ├─ 回调未注入 / RPC 失败 → DefeatRecall()
    │    └─ 失败（!attacker_win）→ DefeatRecall()
    └─ 胜利 → applyDevelopResult()
         ├─ toMapInfo.AddLevel(3)     ← 地块等级 +3
         └─ toMapInfo.MarkDeveloped() ← 标记已开发（ownerID 不变）

Finish → 去留分流
    ├─ IsStay → MarchState_Stay（驻留地块）
    └─ 否则 → CallBack()（MarchState_Back 反向返回）
```

**守军配置**：`game_conf/guard` 域（数据源 `gameconfig.json` 的 guard/guard_config/guard_slot 表），按地块等级 → 守军队伍（英雄配置ID + 固定兵力）。
守军英雄属性由 worldmap 侧派生（`base + growth×(level-1)`，见 `worldmap_inits/engine.func.go`），
经 `GuardConfigFunc` 回调注入 `MapManager`（`SetGuardConfigFunc`），cores 不依赖 game_conf。

---

## 设计要点

- **模板方法模式**: `BaseMarch.Do()` 定义固定三步流程，子类通过 `Add*Opt` 注入具体逻辑
- **CallBack / BackArrive 模板化**: 召回和召回到达也采用同样的模板方法模式，子类可在前后插入类型特定逻辑
- **注册式工厂**: 子包在 `init()` 中注册，工厂无需导入子包，避免循环依赖
- **状态分流**: `SingleMarch.Do()` 根据 `MarchState` 自动分流到正常到达或召回到达
- **等比例返回**: `callbackSwapDirection` 按已走过的时间等比例计算返回所需时间
- **分层加锁/回滚**: 行军锁 → 来源地图锁 → 目标地图锁，任一失败自动按序回滚
- **marchLocker 互斥**: tick handler 使用 `LockMarchDo` 做并发控制，`RwLock` 留给 handler 内部按需获取，避免死锁
- **异步持久化**: `CallBack` / `CallBackNow` 执行后调用 `Save(m.single)` 触发 `asyncsave_entity.EntitySaveFunc`
- **位掩码优化**: `MultiMarch` 使用 `markOff` 位掩码追踪多条行军的加锁状态
