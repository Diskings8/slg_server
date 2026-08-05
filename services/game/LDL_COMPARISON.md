# Game 服务对标分析: ldl vs ai4slg

> 生成日期: 2026-07-28  
> 更新日期: 2026-08-05  
> ldl 路径: `C:\workspace\ldl\server\services\game` (1251 文件, 成熟产品)  
> ai4slg 路径: `C:\workspace\a4s\slg_server\services\game` (85 文件, 快速建设中)

---

## 一、整体规模对比

| 维度 | ldl | ai4slg (07-28) | ai4slg (07-30) | ai4slg (08-04) | 进展 |
|------|-----|----------------|----------------|----------------|------|
| Go 源文件数 | 1251 | 40 | 47 | **85** | +45 |
| 实体子模块 | 24 个 | 4 个 | 5 个 | **7 个** (+Buildings/Formations) | +3 🟢 |
| 内部业务逻辑模块 | 50+ | 0 (逻辑在 cores 层) | 2 个 | **8 个** (hero/troop/skill/star/item/formation/building/march) | +8 🟢 |
| Stream 协议处理器 | ~44 个 | 0 (空骨架) | 2 个 | **20 个** | +20 🟢 |
| RPC Unary 处理器 | 16 个 | 2 个 (仅日志) | 2 个 (统一 Do) | 2 个 (统一 Do) | 🟡 |
| 架构层次 | 6 层完整 | 3 层雏形 | 4 层 | **4 层** (handler→logic→entity→model) | 🟢 |
| 测试文件 | 大量 | 1 个 | 1 个 | **4 个** (role/currency/skill/star) | 🟢 |

---

## 二、架构层次对比

### 2.1 整体架构

```
ldl (6 层)                    ai4slg (08-04)
─────────                     ────────────
handler/ (stream+game)  ← →  game_handlers/ + game_servers/  ✅ 20 协议
logic/ (internal/logics) ← →  game_logics/                   ✅ 8 文件
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
| **`Attr`** | **角色属性(serverID, VIP, 登录统计等)** | ✅ **已完成** (role_attrs + GameAttrList 协议 + 桩替换) |
| **`Builds`** | **建筑(城建系统)** | ✅ **已完成** (role_buildings + BuildingBuild/List) |
| `Arena` | 竞技场 | ❌ |
| `Dungeon` | 关卡 | ❌ |
| **`Equips`** | **装备** | ❌ |
| **`Heroes`** | **英雄** | ✅ **已完成** |
| `Gift` | 礼包 | ❌ |
| **`Items`** | **物品(背包)** | ✅ **已完成** (含一级/二级货币) |
| `Privileges` | 特权(月卡/周卡/VIP) | ❌ |
| `Race` | 军备竞赛 | ❌ |
| **`Recruit`** | **招募** | ✅ **已完成** (role_recruits + 抽卡池/单抽十连/心愿 3 协议) |
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

#### ai4slg Role 当前挂载了 9 个子模块:

```go
func (r *Role) New() {
	roleID := r.ID
	r.Heroes = role_heroes.NewRoleHeroes(roleID)               // ✅
	r.Skills = hero_skills.NewHeroSkills(roleID)               // ✅
	r.SkillCollections = hero_skillcollections.NewHeroSkillCollections(roleID) // ✅
	r.CultivateCosts = cultivate_costs.NewCultivateCosts(roleID) // ✅
	r.Items = role_items.NewRoleItems(roleID)                  // ✅
	r.Buildings = role_buildings.NewRoleBuildings(roleID)      // ✅
	r.Formations = role_formations.NewRoleFormations(roleID)   // ✅
	r.Recruits = role_recruits.NewRoleRecruits(roleID)          // ✅
	r.Attr = role_attrs.NewRoleAttrs(roleID)                    // ✅
}
```

**子模块差距: ldl 24 个 vs ai4slg 9 个** (Builds/Teams/Recruit/Attr 已补上；Equips 等未开始)

---

## 三、英雄养成子系统深度对比

### 3.1 数据模型 (model)

| 维度 | ldl | ai4slg (08-04) | 差距 |
|------|-----|----------------|------|
| RoleHero 字段数 | 9 | 10 (+StarStage/IsAwakened) | 设计不同(ai4slg有Cultivates,无战力) |
| RoleHero 技能槽 | 英雄技能列表 | **EquipSkills 长度3** (index0默认/index1,2装配槽) | ✅ |
| hero_skill 字段 | 无 | **+EquipHeroID/UseCountLimit/UsedCount** | 🆕 装配次数模型 |
| DAO 层 | query.RoleHero (gorm gen) | **手写 CRUD 完整 (7/7 模块)** | 🟢 |
| AutoMigrate | 自动 | **5 模块全部注册** | 🟢 |

### 3.2 实体层 (entity)

| 维度 | ldl | ai4slg (08-04) | 差距 |
|------|-----|----------------|------|
| Heroes | 完整 | **完整** + RemoveHero/DBDeleteHero | ✅ 接近 |
| Items | 完整 | **完整** + 货币类型支持 | ✅ |
| Skills | 完整 | **完整** + GetSkillByConfID/AddSkill/EquipTo/Unequip | ✅ |
| SkillCollections | 完整 | 完整 (New/Init/Copy/Format2Pb/DB CRUD) | ✅ |
| CultivateCosts | 完整 | **完整** + AddCost | ✅ |
| Buildings | 完整 | 完整 | ✅ |
| Formations | 完整 | **完整** + FormationHasHero | ✅ |
| 属性缓存/战力 | ✅ | ❌ | 缺失 |
| 测试 | 完整 | **4 文件** (role/currency/skill/star) | 🟢 待扩展 |

### 3.3 业务逻辑层 (game_logics)

| 功能 | ldl | ai4slg (08-04) | 差距 |
|------|-----|----------------|------|
| 英雄升级 | `heroes.UpgradeLevel()` | **`HeroAddExp()`** — 升级+连升+每10级属性点 | ✅ needExp 接逐级配置表；前端直升协议已移除，exp 仅来自道具effect/战斗 |
| 英雄升星 | `StreamHeroUpgradeStar` | **`HeroUpgradeStar()`** — 消耗同config英雄卡，防误删校验 | ✅ |
| 技能升级 | `StreamHeroUpgradeSkill` | **`HeroSkillUpgrade()`** — 升级英雄身上槽位技能 | ✅ |
| 技能槽/装配/拆卸 | — | **`HeroEquipSkill/UnequipSkill`** — 槽位条件+装配次数+返还 | 🆕 |
| 英雄培养 | 无专门Cultivate | **`HeroCultivate()`** — 5维加点扣属性点 | ✅ ai4slg 特有 |
| **兵种系统** | — | **`HeroTroopUnlock/Transform`** — 解锁+转换 | 🆕 ai4slg 特有 |
| 编队 | `Teams` | **`FormationFieldHero/RemoveHero`** | ✅ |
| 建筑 | `Builds` | **`BuildingBuild`** | ✅ |
| 道具变更 | 完整 | **`ItemChange()`** 统一入口 + 货币类型 | ✅ |
| 出征编队→队伍 | — | **`MarchBuildTeam()`** 英雄快照 | ✅ |
| 荣誉升级/合成 | `...HonorLevel/Synthetic` | ❌ | 缺失(设计不同:升星替代合成) |
| 行军回城 | `MarchBackArrive()` | ❌ | 缺失 |
| 属性刷新 | `refreshattr.Hero()` | ❌ | 缺失 |

---

## 四、协议层对比

### 4.1 Stream 消息处理器 (19 个)

| 协议号 | MsgID | 处理器 | 模块 |
|--------|-------|--------|------|
| 1000001 | GameHeroList | HeroList | hero_handler |
| 1000003 | GameHeroCultivate | HeroCultivate | hero_handler |
| 1000004 | GameHeroSkillUpgrade | HeroSkillUpgrade | hero_handler |
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
| 1000016 | GameHeroEquipSkill | HeroEquipSkill | hero_handler |
| 1000017 | GameHeroUnequipSkill | HeroUnequipSkill | hero_handler |
| 1000018 | GameHeroUpgradeStar | HeroUpgradeStar | hero_handler |

> ⚠️ 注: `game.protocol.gen.go` 中 `HandlerBattleRecordList` 注释协议号有误(写成 1000013)，真实值 1000015。

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
| 1.6 | **实现英雄协议** | ✅ **已完成** | HeroList (1000001) 等 20 个协议 |
| 1.7 | **补齐 CultivateCost DB** | ✅ **已完成** | cost.db.go CRUD 完整 |

**Phase 1 总进度: ~100%** ✅

### 额外完成项 (08-04):

| 内容 | 原属于 Phase | 说明 |
|------|-------------|------|
| **物品系统 (Items)** | Phase 3 (P0) | 完整 entity+model+DB+handler |
| **货币类型** | Phase 2 扩展 | ItemType 一级/二级货币，统一走背包 |
| **建筑系统 (Builds)** | Phase 3 | role_buildings + BuildingBuild/List |
| **编队系统 (Teams)** | Phase 3 | role_formations + FormationField/Remove/List |
| **兵种系统** | Phase 2 扩展 | HeroTroopUnlock/Transform |
| **技能系统** | Phase 2 | 技能槽/装配/拆卸/升级 |
| **英雄升星** | Phase 2 | HeroUpgradeStar（替代合成设计） |
| **战报接入** | Phase 4 | BattleRecordList → battle_record 节点 |
| **gate_stream 连接管理** | 基础设施 | 完整 Init/Join/Close/Push/CallBack/ShutDown |
| **game_logics 包** | Phase 2 | 8 个逻辑文件 |
| **game_roles.GetRole/GetCopy** | 基础设施 | 取数门面（GetRole/GetCopy/Do 收归 game_roles，PollerI 接口在 game_declarations） |

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

**2026-08-04 修复**: ① 移除 `Recv()` 预取的 `GetRole`（与 handler 内 `GetRole` 造成同一 poller 二元锁二次获取死锁 → 1s 超时 SystemBusy）；② 补 `HandlerUseItem` 缺失的 `poller.Save()`（道具扣减不落库）；③ 实现 `rpc_results.Reset()`（原 panic("implement me")，对象池复用即崩，导致所有 `rpc_results.Error` 调用 panic）；④ `ItemChange` 成功路径返回 nil（原返回非 nil `ErrorCode(NoneErr)` 使成功误报，影响道具/兵种/技能消耗）。

---

## 七、优势总结

### ai4slg 已完成优势 (08-04):

1. ✅ **独立地图核心引擎** (cores/) - AOI/行军/战斗
2. ✅ **全协议路由框架** - 泛型 Wrap + 注册表 + Recv 分发 (20 协议)
3. ✅ **Role Entity 完整生命周期** - Pool/Copy/Init/DB/Poller
4. ✅ **7 个子模块完整 CRUD** - Hero/Skill/SkillCollection/CultivateCost/Item/Building/Formation
5. ✅ **物品系统完整** - Add/Reduce/Check + 一级/二级货币统一背包 + ItemChange 入口
6. ✅ **技能系统** - 技能槽/装配/拆卸/升级，装配次数模型 + 拆卸按级返还
7. ✅ **英雄升星** - 消耗同配置卡，防误删校验（编队/技能/养成）
8. ✅ **兵种系统** - 解锁/转换 (ai4slg 特有)
9. ✅ **编队+建筑系统** - 上阵下阵/建造列表
10. ✅ **战报查询接入** - 直达 battle_record 节点
11. ✅ **角色加载分工清晰** - 写 GetRole/读 GetCopy，消除双重加锁

### 主要缺失:

1. ✅ Attr 角色属性系统 (role_attrs: ServerID/VIP/登录统计；ServerID()/VIPLevel()/Offline() 桩已替换)
2. ✅ 道具使用效果 (ApplyItemEffect 已实现: 经验/货币/道具三类效果 + 前置校验)
3. ✅ 英雄升级接配置表 (needExp 读逐级表；前端直升协议 1000002 已移除，exp 仅来自道具effect/战斗)
4. ✅ 技能获得途径 (收藏兑换: 消耗英雄卡 → 收集进度 → 达标发放技能到技能库)
5. 🟡 英雄锁/治疗 (HeroLock/Unlock 已实现，治疗未做)
6. ✅ 技能收藏激活 (收藏兑换消耗客户端选定英雄卡，解锁即发放技能，可装配)
7. ✅ 英雄属性计算已接入（hero.conf 基础/成长/星级/加点 → CalcHeroAttr，参与战斗公式）；战力聚合数字不接入
8. ✅ 战斗结果回调（每场经验 = 敌方平均等级×击杀×系数÷参战英雄；逐场升级；战报记录；MarchEvent 回传 game HeroAddExp）
9. ❌ 配置表系统 (仅占位; needExp/技能/升级消耗等硬编码)
10. ❌ 货币兑换 (一级→二级, 比例占位)
11. ❌ 产销日志落库 (ItemChange 收集但未写 DB)

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

Phase 2 的 8 个子任务（含实际调整），标注 **08-04 实际进度**:

| 顺序 | 内容 | 状态 |
|------|------|------|
| **2.1** | **英雄升级** — 接配置表 + 收紧 exp 来源 | ✅ needExp 读逐级配置表；前端直升协议已移除，exp 仅来自道具effect/战斗（08-05） |
| **2.2** | **英雄培养** — 5维属性消耗属性点 | ✅ 完成 |
| **2.3** | **道具使用效果** — 使用道具加经验/加属性 | ✅ ApplyItemEffect 已实现（经验/货币/道具三类效果） |
| **2.4** | **技能系统** — 技能槽/装配/拆卸/升级 | ✅ 完成 |
| **2.5** | **英雄升星** — 消耗同配置英雄卡（替代合成） | ✅ 完成 |
| **2.6** | **英雄锁/治疗** — IsLocked 状态管理 | 🟡 英雄锁已实现（HeroLock/Unlock），治疗未做 |
| **2.7** | **技能收藏激活** — 收藏兑换消耗英雄卡 | ✅ 消耗客户端选定英雄卡 → 达标解锁并发放技能（08-05） |
| **2.8** | **货币兑换** — 一级→二级 | ❌ |

### Phase 3 前置工作状态 (08-04):

| 内容 | 状态 | 备注 |
|------|------|------|
| **Attr 属性系统** | ✅ **已提前完成** | role_attrs 子模块 + GameAttrList 协议；ServerID/VIPLevel/Offline 桩已替换；玩家等级仍派生于建筑 |
| **Teams 队伍编成** | ✅ **已提前完成** | role_formations + 3 协议 |
| **Builds 城建系统** | ✅ **已提前完成** | role_buildings + 2 协议 |
| **配置表系统** | ❌ 未开始 | 全项目缺口，needExp/升级消耗/技能数值依赖 |

---

## 九、关键结论

1. **Phase 1 完成** (≈100%) — DB→Entity→Role→gRPC 全链路已通
2. **08-04 大爆发**: 技能系统（槽/装配/拆卸/升级）、英雄升星、货币类型 3 大块完成，协议 16 → 20，测试 4 文件
3. **基础设施修复 4 项**: Recv 双重加锁、道具 Save 遗漏、rpc_results.Reset panic、ItemChange 成功误报
4. **当前主线 Phase 2** — 2.1 英雄升级接配置表、2.3 道具使用效果、2.6 英雄锁、2.7 技能收藏兑换均已完成；剩余 **2.8 货币兑换**
5. **配置表是隐藏前置** — 2.3/技能数值等仍依赖配置表系统
6. **Attr 属性系统** — ✅ 已完成 (role_attrs + 桩替换 + 登录统计钩子)；玩家等级未入 Attr，保持派生于建筑
7. **战斗模块完善 (08-05)** — 英雄属性接入战斗（属性加权公式）、战斗经验结算 + 逐场升级（下一场以新属性进入）、每场经验记录战报、MarchEvent 回传 game 落地 HeroAddExp；无战力聚合数字

### 👉 下一步建议: 2.8 货币兑换

| 内容 | 理由 |
|------|------|
| **2.8 货币兑换** | 一级→二级兑换未做；`ItemChange` 已支持货币类型，链路就绪 |
| 备选 | **配置表系统**（隐藏前置，技能数值/升级消耗依赖） |
