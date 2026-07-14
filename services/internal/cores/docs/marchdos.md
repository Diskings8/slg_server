# marchdos — 行军执行器

> 路径: `services/internal/cores/marchdos/`  
> 文件: `base.march.st.go` · `single.march.st.go` · `multi.march.st.go`

行军到达目的地后执行具体动作的模块，采用**模板方法模式**。

---

## 1. BaseMarch — 基类

```go
type BaseMarch struct {
    marchManage   *marchs.MarchInfoManager
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
| `Do(mapManager)` | 按序执行三个阶段，未 Init 则 panic |

---

## 2. SingleMarch — 单一行军执行器

```go
type SingleMarch struct {
    BaseMarch
    single          *marchs.MarchInfo
    MarchType       cores_declarations.MarchType
    arriveAfterFunc func(*map_managers.MapManager, *marchs.MarchInfo)
}
```

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

### Init

注入默认空回调后调用 `BaseMarch.Init()`。

---

## 3. MultiMarch — 多行军执行器

```go
type MultiMarch struct {
    BaseMarch
    multi           []*marchs.MarchInfo
    markOff         int32
    marchLen        int
    MarchType       cores_declarations.MarchType
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

## 设计要点

- **模板方法模式**: `BaseMarch.Do()` 定义固定三步流程，子类通过 `Add*Opt` 注入具体逻辑
- **分层加锁/回滚**: 行军锁 → 来源地图锁 → 目标地图锁，任一失败自动按序回滚
- **位掩码优化**: `MultiMarch` 使用 `markOff` 位掩码追踪多条行军的加锁状态，避免每次遍历判断
- **安全防护**: `Do()` 前必须调用 `Init()`，否则 panic 防止未初始化的执行
