# Battle — 战斗结算服务

> 路径: `services/battle`  
> 模块: `server.slg.com/services/battle`

---

## 概述

`battle` 是 SLG 服务端的**战斗结算节点**，提供 `BattleSettle` RPC 完成纯战斗计算（战力对比 / 伤亡分配 / 胜负判定 / 攻城）。与 `game`/`worldmap` 按 `instance` **单例配对**：worldmap 行军到达（attack_march）时调用本节点结算，拿到结果后由 worldmap 应用（占城 / 状态 / 伤亡 / 推送）。

**职责边界**：battle 只做**计算**，不持有 DB / 地图引擎 / 行军状态机——那些仍是 worldmap（cores）的职责。

---

## 目录结构

```
battle/
├── main.go                                    # 入口：配置/日志/ETCD/gRPC + 生命周期
│
├── battle_handlers/                            # 协议入口
│   └── battle_servers/                        # Unary RPC 处理器（BattleHandler）
│       ├── battle.server.st.go               # BattleServer 结构体（无状态）
│       └── battle.server.func.go             # BattleSettle：校验 → battle_logics.Settle
│
└── battle_internals/                           # 内部实现
    └── battle_logics/                          # 战斗计算（迁移自 cores/attack_march）
        ├── battle.settle.func.go             # Settle 主入口 + settleTarget（攻城/PvE）
        ├── battle.layer.func.go              # resolveDefendersLayers / fightLayer
        ├── battle.team.func.go               # 队伍快照工具（战力/拆迁/扣损/深拷贝）
        └── battle.settle_test.go             # 纯逻辑单测
```

---

## 核心设计

### 1. 纯计算 + 快照入参

battle 节点只操作 `pb_battle` 快照，不 import `services/internal/cores`：

- 入参 `BattleSettleReq`：攻方 `attacker_team`（`TeamInfo`，内含英雄 + 士兵）+ 防守方分层 `defender_groups`（assist → stay/idle）+ 目标建筑耐久
- 出参 `BattleSettleRsp`：逐层 `results`（`BattleResults`，复用 `battle.proto`）+ `attacker_win` + `occupied` + 被击败防守方 + `building_damage`

### 2. 结算流程（多轮战斗）

```
Settle(req)
  ├─ cloneSlots(攻击方队伍)          ← 深拷贝，层间不互相污染
  ├─ resolveDefendersLayers          ← PvP 逐层（assist → stay/idle）
  │    └─ fightLayer：每层多轮交战（至多 10 轮）
  │        每轮：攻守双方按对方战力占比互相扣损
  │        终止：一方全灭 / 双方无进展（士兵减到下限） / 轮次上限
  │        层胜负：剩余战力高者胜，平局攻方胜
  │        攻方溃败 → 后续层/攻城不再结算
  ├─ settleTarget（攻方未败才进入）
  │    有建筑 → 攻城：broken = 拆迁值 > 建筑耐久（严格大于）
  │    无建筑 → PvE：直接占领
  └─ 组装 rsp（results = 逐轮 OneBattleResult + 攻城/PvE 层）
```

- **每轮**：`attLoss = att*(def/(att+def))`、`defLoss = def*(att/(att+def))`，双方承伤（受伤英雄槽位跳过）。
- **`BattleResults.results` 语义**：一条 = 一轮战斗；防御方战后队伍通过 `DefeatedDefender.TeamAfter` 返回，供 worldmap 写回伤亡。

### 3. 调用链（worldmap → battle）

```
march tick → MarchTickHandler → march_factory.NewMarchDo → attack_march.Do
  → buildBattleSettleReq（cores 构建防守方，依赖 MapManager）
  → mgr.GetBattleSettleFunc()(req)          ← 回调由 worldmap Engine 注入
  → e.settleBattle → BattleHandlerClient.BattleSettle → battle_logics.Settle
  → rsp → applyBattleSettleRsp（cores 应用：伤亡写回/占城/状态/建筑耐久/推送）
```

回调通过 `cores_declarations.BattleSettleFunc` 函数类型注入 `MapManager`，**cores 保持纯逻辑包**（不依赖 rpcconn）。

### 4. 结果回流要点

| 结果 | 写回位置 | 说明 |
|------|---------|------|
| 攻击方伤亡 | `MarchInfo.Team.ApplyTeamInfo` | 用最后一层 attacker.team_info 快照覆盖 Slots |
| 防守方伤亡 | `MarchInfo.Team.ApplyTeamInfo` | 多轮防守方承伤，rsp 的 TeamAfter 写回 |
| 防守方回城 | `MarchState_Back` + `AssistCallBack` | 按 rsp 的 march_id 解析 |
| 建筑耐久 | `BuildingI.ReduceBuildingsHp` | battle 只算伤害值，worldmap 就地扣血 |
| 占领 | `MapInfo.Occupy` | `occupied && attacker_win` 才占 |
| 战败召回 | `SingleMarch.DefeatRecall` | 攻方溃败 → 反转方向返回 TransitMapID（-1 回退 SrcFromMapID）+ 重新入队 ticker；回调未注入 / RPC 失败同样召回 |

---

## 通信

```
worldmap ──Unary──▶ battle（本服务）
  BattleHandler.BattleSettle（结算请求 → 计算结果）

ETCD 注册：/node:service:battle/{instance}/ （与 worldmap 同 instance 配对）
发现：worldmap 通过 rpc_handlers hub → etcd GetNodeTypeServerAddrByInstance(NodeBattleService, instance)
```

- **接收方**：worldmap（attack_march 到达时）
- **服务发现**：`common/conns/rpcconn/rpc_handlers`（`GetBattleHandlerClient`，`client_handler_gen` 生成）

---

## 外部依赖

- `api/protocol/pb/pb_battle` — 协议定义（含 `battle.proto` 数据消息）
- `common/conns/etcdconn` — ETCD 服务注册
- `common/loggers` — 日志
- `common/servers` — 生命周期框架
