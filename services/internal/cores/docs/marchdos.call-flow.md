# marchdos 调用流

从 tick 触发到行军到达处理的完整链路。

---

```ascii
loopTickCheck (100ms ticker)
  │
  ├─ timeMarch[nowTime] → marchIDs
  │
  └─ for marchID := range marchIDs
       │
       ▼  go mm.marchDoFunc(mm, marchID)
          │
          │  mm.marchDoFunc = DefaultMarchTickHandler（注入时指定）
          │
          ▼
          DefaultMarchTickHandler(mm, marchID)
            │
            ├─ GetMarchInfo(marchID)
            │    └─ nil → return（行军已删除）
            │
            ├─ endTime > now → TickerChan ← info（未到期，重新入队等待）
            │
            ├─ LockMarchDo() → false → return（被其他协程处理中）
            │
            ├─ NewMarchDo(mm, info)
            │    │
            │    │  march.factory.go 注册表：
            │    │    MarchTypeAttack  → attack.New
            │    │    MarchTypeAssist  → assist.New
            │    │    MarchTypeSweep   → sweep.New
            │    │    MarchTypeStrategy→ strategy.New
            │    │    MarchTypeDevelop → strategy.newDevelop
            │    │
            │    └─ nil → return（未知行军类型）
            │
            ├─ toMapLock = state != MarchState_Back
            ├─ Lock(true, false, toMapLock)
            │    │
            │    │  SingleMarch.TryLock:
            │    │    1. marchLock → m.single.TryLock()
            │    │    2. fromLock  → false（跳过）
            │    │    3. toLock   → toMapInfo.LockMarchDo()
            │    │
            │    ▼
            ├─ handle.Do()
            │    │
            │    │  SingleMarch.Do():
            │    │    │
            │    │    ├─ MarchState_Back ──► BackArrive()
            │    │    │                       ├─ TryLock (RwLock)
            │    │    │                       ├─ BaseMarch.BackArrive()
            │    │    │                       ├─ UpdateMarchPush
            │    │    │                       └─ DeleteMarch
            │    │    │
            │    │    └─ 其他状态 ──► BaseMarch.Do()
            │    │                     │
            │    │                     │  attack.New 注册的三阶段：
            │    │                     │
            │    │                     ├─ Prepare
            │    │                     │   └─ checkTargetLegality
            │    │                     │       ├─ 合法 → continue
            │    │                     │       └─ 非法 → MarchState = Back
            │    │                     │
            │    │                     ├─ Do（到达处理）
            │    │                     │   ├─ settleBattle
            │    │                     │   │   ├─ calcTeamPower
            │    │                     │   │   ├─ calcDefenderPower
            │    │                     │   │   └─ BattleResult
            │    │                     │   │
            │    │                     │   └─ processBattleResult
            │    │                     │       ├─ 攻击方胜 → occupyTile → Save(tile)
            │    │                     │       └─ 防守方胜 → MarchState = Back
            │    │                     │       └─ Save(march)
            │    │                     │
            │    │                     └─ Finish
            │    │                       ├─ pushBattleResult
            │    │                       │   ├─ UpdateMarchPush
            │    │                       │   └─ UpdateMapPush
            │    │                       └─ triggerBattleEvents（预留）
            │    │
            │    ▼
            ├─ err != nil ──► handle.CallBack()
            │                   ├─ TryLock (RwLock)
            │                   ├─ BaseMarch.CallBack()
            │                   │   └─ callbackSwapDirection
            │                   │       ├─ 等比例重算 EndTimeUx
            │                   │       ├─ 交换 From/To
            │                   │       ├─ MapAttributeMarchCallBack
            │                   │       ├─ MarchState = Back
            │                   │       ├─ 重算 AOI 路径
            │                   │       └─ TickerAddMarch（重新注册 ticker）
            │                   ├─ UpdateMarchPush
            │                   └─ Save(march)
            │
            └─ UnlockMarchDo()
```
