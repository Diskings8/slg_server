# Game 服务对标分析: ldl vs ai4slg

> 生成日期: 2026-07-28  
> 更新日期: 2026-08-04  
> ldl 路径: `C:\workspace\ldl\server\services\game` (1251 文件, 成熟产品)  
> ai4slg 路径: `C:\workspace\a4s\slg_server\services\game` (78 文件, 快速建设中)

---

## 一、整体规模对比

| 维度 | ldl | ai4slg (07-28) | ai4slg (07-30) | ai4slg (08-04) | 进展 |
|------|-----|----------------|----------------|----------------|------|
| Go 源文件数 | 1251 | 40 | 47 | **78** | +31 |
| 实体子模块 | 24 个 | 4 个 | 5 个 | **7 个** (+Buildings/Formations) | +2 🟢 |
| 内部业务逻辑模块 | 50+ | 0 (逻辑在 cores 层) | 2 个 | **6 个** (hero/hero.troop/item/formation/building/march) | +4 🟢 |
| Stream 协议处理器 | ~44 个 | 0 (空骨架) | 2 个 | **16 个** | +14 🟢 |
| RPC Unary 处理器 | 16 个 | 2 个 (仅日志) | 2 个 (统一 Do) | 2 个 (统一 Do) | 🟡 |
| 架构层次 | 6 层完整 | 3 层雏形 | 4 层 | **4 层** (handler→logic→entity→model) | 🟢 |
| 测试文件 | 大量 | 1 个 | 1 个 | **1 个** (role_test, 5 测试) | ✅ |

---

## 二、架构层次对比

### 2.1 整体架构

```
ldl (6 层)                    ai4slg (08-04)
─────────                     ────────────
handler/ (stream+game)  ← →  game_handlers/ + game_servers/  ✅ 16 协议
logic/ (internal/logics) ← →  game_logics/                   ✅ 6 文件
entity/                 ← →  game_entitys/                   ✅ 7个子模块
model/                  ← →  game_models/                    ✅ 5个model
query/ (gorm gen)       ← →  game_generates/                  🟡 gen入口已配置多表映射
internal/               ← →  services/internal/cores/         🟡 独立位置
rpcclient/              ← →  game_internals/game_rpc_clients/ ✅ 按 instance 配对
```

### 2.2 角色实体 (Role) 子模块对比

#### ldl Role 挂载了 24 个子模块:

| 模块 | 说明 | ai4slg (08-04) |
|------|------|--------|
| `Activity` | 活动数据(17+活动类型) | ❌ |
| `ActivityOther` | 活动其他数据 | ❌ |
| `Attr` | 角色属性(serverID, 资源, VIP等) | ❌ |
| **`Builds`** | **建筑(城建系统)** | ✅ **已完成** (role_buildings + BuildingBuild/List) |
| `Arena` | 竞技场 | ❌ |
| `Dungeon` | 关卡 | ❌ |
| **`Equips`** | **装备** | ❌ |
| **`Heroes`** | **英雄** | ✅ **已完成** |
| `Gift` | 礼包 | ❌ |
| **`Items`** | **物品(背包)** | ✅ **已完成** |
| `Privileges` | 特权(月卡/周卡/VIP) | ❌ |
| `Race` | 军备竞赛 | ❌ |
| `Recruit` | 招募 | ❌ |
| `RoleUnion` | 联盟信息 | ❌ |
| `ShopDailyDeal` | 商城:每日特惠 | ❌ |
| `ShopDailyMustHave` | 商城:每日必买 | ❌ |
| `ShopDawnFund` | 商城:黎明基金 | ❌ |
| `ShopWeekDeal` | 商城:每周特惠 | ❌ |
| `SimpleModule` | 简单模块 | ❌ |
| `Stores` | 商店 | ❌ |
| `SystemSet` | 系统设置 | ❌ |
| **`Teams`** | **布阵队伍** | ✅ **已完成** (role_formations + FormationField/Remove/List) |
| **`Tasks`** | **任务** | ❌ |
| `Towers` | 荣耀远征 | ❌ |
| `BuildData` | 建筑相关数据 | ❌ |
| `WorldBoss` | 世界Boss | ❌ |
| `Privacy` | 隐蔽指挥所 | ❌ |

#### ai4slg Role 当前挂载了 7 个子模块:

```go
func (r *Role) New() {
	roleID := r.ID
	r.Heroes = role_heroes.NewRoleHeroes(roleID)               // ✅
	r.Skills = hero_skills.NewHeroSkills(roleID)               // ✅
	r.SkillCollections = hero_skillcollections.NewHeroSkillCollections(roleID) // ✅
	r.CultivateCosts = cultivate_costs.NewCultivateCosts(roleID) // ✅
	r.Items = role_items.NewRoleItems(roleID)                  // ✅
	r.Buildings = role_buildings.NewRoleBuildings(roleID)      // ✅ NEW (08-01)
	r.Formations = role_formations.NewRoleFormations(roleID)   // ✅ NEW (08-01)
}
```

**子模块差距: ldl 24 个 vs ai4slg 7 个** (进展: 0→7, Builds/Teams 两个核心战斗前置已补上)

---

## 三、英雄养成子系统深度对比

### 3.1 数据模型 (model)

| 维度 | ldl | ai4slg (08-04) | 差距 |
|------|-----|----------------|------|
| RoleHero 字段数 | 9 | 8 | 设计不同(ai4slg有Cultivates,无战力) |
| DAO 层 | query.RoleHero (gorm gen) | **手写 CRUD 完整 (7/7 模块)** | 🟢 |
| AutoMigrate | 自动 | **5 模块全部注册** | 🟢 |

### 3.2 实体层 (entity)

| 维度 | ldl | ai4slg (08-04) | 差距 |
|------|-----|----------------|------|
| Heroes | 完整 | **完整** (New/Init/Copy/Format2Pb) | ✅ 接近 |
| Items | 完整 | **完整** (New/Init/Copy/Format2Pb/Add/Reduce/Check) | ✅ |
| Skills | 完整 | **完整** (New/Init/Copy/Format2Pb/DB CRUD) | ✅ |
| SkillCollections | 完整 | **完整** (New/Init/Copy/Format2Pb/DB CRUD) | ✅ |
| CultivateCosts | 完整 | **完整** (New/Init/Copy/Format2Pb/DB CRUD) | ✅ |
| Buildings | 完整 | **完整** (New/Init/Copy/Format2Pb/DB CRUD) | ✅ NEW |
| Formations | 完整 | **完整** (New/Init/Copy/Format2Pb/DB CRUD) | ✅ NEW |
| 属性缓存/战力 | ✅ | ❌ | 缺失 |
| 测试 | 完整 | **1 个文件, 5 个测试** | 🟢 待扩展 |

### 3.3 业务逻辑层 (game_logics)

| 功能 | ldl | ai4slg (08-04) | 差距 |
|------|-----|----------------|------|
| 英雄升级 | `heroes.UpgradeLevel()` | **`HeroAddExp()`** — 升级+连升+每10级属性点 | 🟡 配置表/消耗未接 |
| 英雄升星 | `StreamHeroUpgradeStar` | ❌ | 缺失 |
| 技能升级 | `StreamHeroUpgradeSkill` | ❌ | 缺失 |
| 技能解锁 | `UnlockSkill()` | ❌ | 缺失 |
| 荣誉升级 | `StreamHeroUpgradeHonorLevel` | ❌ | 缺失 |
| 英雄合成 | `StreamHeroSynthetic` | ❌ | 缺失 |
| 英雄培养 | 无专门Cultivate | **`HeroCultivate()`** — 5维加点扣属性点 | ✅ ai4slg 特有 |
| **兵种系统** | — | **`HeroTroopUnlock/Transform`** — 解锁+转换 | 🆕 ai4slg 特有 |
| 编队 | `Teams` | **`FormationFieldHero/RemoveHero`** | ✅ |
| 建筑 | `Builds` | **`BuildingBuild`** | ✅ |
| 道具变更 | 完整 | **完整 `ItemChange()`** 统一入口 | ✅ |
| 出征编队→队伍 | — | **`MarchBuildTeam()`** 英雄快照 | ✅ |
| 行军回城 | `MarchBackArrive()` | ❌ | 缺失 |
| 属性刷新 | `refreshattr.Hero()` | ❌ | 缺失 |

---

## 四、协议层对比

### 4.1 Stream 消息处理器 (16 个)

| 协议号 | MsgID | 处理器 | 模块 |
|--------|-------|--------|------|
| 1000001 | GameHeroList | HeroList | hero_handler |
| 1000002 | GameHeroUpgradeLevel | HeroUpgradeLevel | hero_handler |
| 1000003 | GameHeroCultivate | HeroCultivate | hero_handler |
| 1000005 | GameUseItem | UseItem | item_handler |
| 1000006 | GameMarchCreate | MarchCreate | march_handler |
| 1000007 | GameMapData | MapData | map_handler |
| 1000008 | GameFormationField | FormationField | formation_handler |
| 1000009 | GameFormationRemove | FormationRemove | formation_handler |
| 1000010 | GameFormationList | FormationList | formation_handler |
| 1000011 | GameBuildingBuild | BuildingBuild | building_handler |
| 1000012 | GameBuildingList | BuildingList | building_handler |
| 1000013 | GameHeroTroopTransform | HeroTroopTransform | hero_handler |
| 1000014 | GameHeroTroopUnlock | HeroTroopUnlock | hero_handler |
| 1000015 | GameBattleRecordList | BattleRecordList | battle_record_handler |

> ⚠️ 注: `game.protocol.gen.go` 中 `HandlerHeroTroopTransform`/`HandlerBattleRecordList` 的注释协议号有误(写成 1000013)，真实值见 `api/protocol/src/protocol.proto`：兵种转换=1000013、兵种扩展=1000014、战报列表=1000015。

### 4.2 架构要点 (08-04)

| 维度 | 状态 |
|------|------|
| 协议注册 | registry.go + 泛型 Wrap，手工维护（后续 game_generates 自动生成） |
| 消息路由 | `Recv` 分发（相机消息直转 worldmap 流）+ 统一错误回写 |
| **角色加载分工** | **写 handler 用 `GetRole`（持锁+Save 打脏）；只读 handler 用 `GetCopy`（免锁快照）** |
| gate_stream | 完整 (Join/Close/Push/CallBack/ShutDown) |
| 跨服 RPC | worldmap/battle_record 客户端按 instance 配对 |

---

## 五、Phase 1 完成度评估

### Phase 1 原计划 (来自第8章):

| 步骤 | 内容 | 状态 | 备注 |
|------|------|------|------|
| 1.1 | **game_declarations 填充** | ✅ **按需定义** | 无需提前填充, 使用时自然加入 |
| 1.2 | **Role 挂载子模块** | ✅ **已完成** | 7 个子模块 |
| 1.3 | **补齐缺失的 DB 操作** | ✅ **已完成** | 7/7 模块完整 DBCreate/DBGet/DBSave/DBDelete |
| 1.4 | **补充 AutoMigrate** | ✅ **已完成** | 5/5 模块在 Init() 中注册 |
| 1.5 | **搭建协议路由** | ✅ **已完成** | registry.go + Recv 路由 + 泛型 Wrap |
| 1.6 | **实现英雄协议** | ✅ **已完成** | HeroList (1000001) 等 16 个协议 |
| 1.7 | **补齐 CultivateCost DB** | ✅ **已完成** | cost.db.go CRUD 完整 |

**Phase 1 总进度: ~100%** ✅

### 额外完成项 (08-04):

| 内容 | 原属于 Phase | 说明 |
|------|-------------|------|
| **物品系统 (Items)** | Phase 3 (P0) | 完整 entity+model+DB+handler |
| **建筑系统 (Builds)** | Phase 3 | role_buildings + BuildingBuild/List |
| **编队系统 (Teams)** | Phase 3 | role_formations + FormationField/Remove/List |
| **兵种系统** | Phase 2 扩展 | HeroTroopUnlock/Transform |
| **战报接入** | Phase 4 | BattleRecordList → battle_record 节点 |
| **gate_stream 连接管理** | 基础设施 | 完整 Init/Join/Close/Push/CallBack/ShutDown |
| **game_logics 包** | Phase 2 | 6 个逻辑文件（hero/troop/item/formation/building/march） |
| **game_role_handler** | 基础设施 | GetRole/Do/GetCopy poller 管理辅助 |

---

## 六、当前架构工作流

```
客户端 → Gateway → gRPC Stream
                        ↓
              game_streams.Recv()
                        ↓
              game_handlers.GetProtoHandler(msgID)
                        ↓
              HandlerFunc(ctx, roleID, req, resp)
                        ↓
        ┌───────────────┴──────────────┐
        ▼                              ▼
   写 handler                     只读 handler
   GetRole(持锁)                  GetCopy(免锁快照)
        ↓                              ↓
   game_logics.* 改数据            game_logics.* 查数据
        ↓
   poller.Save() 打脏
        ↓
   gate_stream.GateCallBackSuccess/Fail
```

**2026-08-04 修复**: 原 `Recv()` 预取 `GetRole` 与 handler 内 `GetRole` 造成同一 poller 二元锁二次获取（死锁 → 1s 超时 → SystemBusy）。已移除 `Recv` 预取，角色加载完全下放到各 handler；并补上 `HandlerUseItem` 缺失的 `poller.Save()`（道具扣减此前不落库）。

---

## 七、优势总结

### ai4slg 已完成优势 (08-04):

1. ✅ **独立地图核心引擎** (cores/) - AOI/行军/战斗
2. ✅ **全协议路由框架** - 泛型 Wrap + 注册表 + Recv 分发 (16 协议)
3. ✅ **Role Entity 完整生命周期** - Pool/Copy/Init/DB/Poller
4. ✅ **7 个子模块完整 CRUD** - Hero/Skill/SkillCollection/CultivateCost/Item/Building/Formation
5. ✅ **物品系统完整实现** - Add/Reduce/Check/Format2Pb + 统一 ItemChange 入口
6. ✅ **gate_stream 网关连接管理** - Join/Close/Push/CallBack
7. ✅ **兵种系统** - 解锁/转换 (ai4slg 特有)
8. ✅ **编队+建筑系统** - 上阵下阵/建造列表
9. ✅ **战报查询接入** - 直达 battle_record 节点
10. ✅ **角色加载分工清晰** - 写 GetRole/读 GetCopy，消除双重加锁

### 主要缺失:

1. ❌ Attr 角色属性系统 (资源/VIP/ServerID)
2. ❌ 英雄完整养成: 升星/技能升级/合成/解锁
3. ❌ 战力/属性计算与缓存
4. ❌ 道具使用效果 (HandlerUseItem 的 TODO)
5. ❌ 战斗结果回调处理
6. ❌ 配置表系统 (仅 pb.confs.st.go 占位，needExp 等为硬编码公式)

---

## 八、更新路线图

```
Phase 1 (补齐基础链路) → ≈100% 完成 ✅
         ↓
Phase 2 (核心养成玩法) ← 当前 👈
         ↓
Phase 3 (基础子系统, 已提前完成部分)
         ↓
Phase 4 (战斗打通)
         ↓
Phase 5 (子系统扩展)
         ↓
Phase 6 (运营活动)
```

### 👉 当前阶段: Phase 2 — 核心养成玩法

Phase 2 的 8 个子任务按推荐实现顺序，标注 **08-04 实际进度**:

| 顺序 | 内容 | 前置 | 状态 |
|------|------|------|------|
| **2.1** | **英雄升级 LevelUp** — 接入配置表 + 消耗道具 | `HeroAddExp` 已实现 | 🟡 needExp 为占位公式 `level*100`，未扣消耗 |
| **2.2** | **英雄培养 Cultivate** — 5维属性消耗属性点 | 已有 HeroCultivate | ✅ 完成 |
| **2.3** | **道具使用效果** — 使用道具加经验/加属性 | Items + HandlerUseItem | 🟡 TODO 只扣道具无效果 |
| **2.4** | **技能解锁** — 按等级/条件自动解锁 | Skills 实体已完成 | ❌ |
| **2.5** | **技能升级** — 消耗材料升级技能等级 | 2.4 完成后 | ❌ |
| **2.6** | **英雄合成** — 碎片合成新英雄 | Items 系统(碎片) | ❌ |
| **2.7** | **英雄锁/治疗** — IsLocked 状态管理 | | ❌ |
| **2.8** | **技能收藏激活** — 收藏加成生效 | SkillCollection 实体已完成 | ❌ |

### Phase 3 前置工作状态 (08-04):

| 内容 | 状态 | 备注 |
|------|------|------|
| **Attr 属性系统** | ❌ 未开始 | Role.ServerID/Level/VIPLevel 仍硬编码 TODO |
| **Teams 队伍编成** | ✅ **已提前完成** | role_formations + 3 协议 |
| **Builds 城建系统** | ✅ **已提前完成** | role_buildings + 2 协议 |
| **配置表系统** | ❌ 未开始 | 全项目缺口，needExp/升级消耗等依赖它 |

---

## 九、关键结论

1. **Phase 1 完成** (≈100%) — DB→Entity→Role→gRPC 全链路已通
2. **08-04 新增**: 编队/建筑/兵种/战报 4 个子系统提前完成，协议从 2 → 16
3. **基础设施修复**: 移除 Recv 双重加锁；补上道具 Save 落库
4. **当前主线 Phase 2** — 英雄养成核心驱动力，建议从 **2.3 道具使用效果**（接 2.1 升级，同一链路）切入
5. **配置表是隐藏前置** — 2.1/2.3 的数值（升级经验/消耗）都依赖配置表系统，建议先落一个最小配置表读取
6. **Attr 属性系统** — 3 个硬编码方法待替换，但优先级可延后到养成链路跑通
