# Game 服务对标分析: ldl vs ai4slg

> 生成日期: 2026-07-28  
> ldl 路径: `C:\workspace\ldl\server\services\game` (1251 文件, 成熟产品)  
> ai4slg 路径: `C:\workspace\a4s\slg_server\services\game` (40 文件, 早期建设)

---

## 一、整体规模对比

| 维度 | ldl | ai4slg | 差距 |
|------|-----|--------|------|
| Go 源文件数 | 1251 | 40 | 31× |
| 实体子模块 | 24 个 | 4 个 (heroes, skills, skillcollections, cultivate_costs) | 6× |
| 内部业务逻辑模块 | 50+ | 0 (逻辑在 cores 层) | ∞ |
| Stream 协议处理器 | ~44 个 | 0 (空骨架) | ∞ |
| RPC Unary 处理器 | 16 个 | 2 个 (CreateRole/LoginOnce 仅日志) | 8× |
| 架构层次 | 6 层完整 | 3 层雏形 | 2× |
| 测试文件 | 大量 | 1 个 (game.role.st_test.go) | ∞ |
| 代码生成 | gorm gen + proto gen | gorm gen 入口 (无生成) | ⬜ |

---

## 二、架构层次对比

### 2.1 整体架构

```
ldl (6 层)                    ai4slg (3 层)
─────────                     ─────────
handler/ (stream+game)  ← →  game_handlers/ + game_servers/  ⚠️ 空骨架
entity/                 ← →  game_entitys/                   🟡 局部(仅hero系)
model/                  ← →  game_models/                    🟡 局部(仅4个model)
query/ (gorm gen)       ← →  ❌ 缺失
internal/               ← →  services/internal/cores/         🟡 独立位置,不同风格
generate/               ← →  game_generates/                  ⬜ 入口存在,无产出
rpcclient/              ← →  ❌ 缺失
```

### 2.2 角色实体 (Role) 子模块对比

#### ldl Role 挂载了 24 个子模块:

| 模块 | 说明 |
|------|------|
| `Activity` | 活动数据(17+活动类型) |
| `ActivityOther` | 活动其他数据 |
| `Attr` | 角色属性(serverID, 资源, VIP等) |
| **`Builds`** | **建筑(城建系统)** |
| `Arena` | 竞技场 |
| `Dungeon` | 关卡 |
| **`Equips`** | **装备** |
| **`Heroes`** | **英雄** |
| `Gift` | 礼包 |
| **`Items`** | **物品(背包)** |
| `Privileges` | 特权(月卡/周卡/VIP) |
| `Race` | 军备竞赛 |
| `Recruit` | 招募 |
| `RoleUnion` | 联盟信息 |
| `ShopDailyDeal` | 商城:每日特惠 |
| `ShopDailyMustHave` | 商城:每日必买 |
| `ShopDawnFund` | 商城:黎明基金 |
| `ShopWeekDeal` | 商城:每周特惠 |
| `SimpleModule` | 简单模块 |
| `Stores` | 商店 |
| `SystemSet` | 系统设置 |
| **`Teams`** | **布阵队伍** |
| **`Tasks`** | **任务** |
| `Towers` | 荣耀远征 |
| `BuildData` | 建筑相关数据 |
| `WorldBoss` | 世界Boss |
| `Privacy` | 隐蔽指挥所 |

#### ai4slg Role 当前挂载了 0 个子模块:

```go
func (r *Role) New() {
    roleID := r.ID
    _ = roleID
    // 全部注释掉了
    // r.Heroes = heroes.New(roleID)
    // r.Skills = skills.New(roleID)
    // ...
}
```

**子模块差距: ldl 24 个 vs ai4slg 0 个** (虽有4个实体包但未挂载到Role上)

---

## 三、英雄养成子系统深度对比

### 3.1 数据模型 (model)

| 维度 | ldl | ai4slg | 差距 |
|------|-----|--------|------|
| RoleHero 字段数 | 9 (ID, Level, StarStage, HonorLevel, TeamID, ShowState, Skills, Power, RoleID) | 8 (ID, Level, Exp, Cultivates, EquipSkills, EquipWeapons, Troops, IsLocked) | 设计理念不同 |
| HeroSkills 类型 | 自定义类型(实现 driver.Valuer/Scan) | pb_skill.Skill 直接存储 | 相当 |
| GORM 生成 | 自动生成 (gorm gen) | 手写 | 有差距 |
| DAO 层 | query.RoleHero (类型安全) | 手写 CRUD | 有差距 |

### 3.2 实体层 (entity)

| 维度 | ldl | ai4slg | 差距 |
|------|-----|--------|------|
| Heroes | 完整(Copy/Format/Init/Contains/Create/Gets) | 完整(Copy/Format2Pb/Init) | 接近,少业务方法 |
| Hero | 属性缓存(AttrsCache) + 战力缓存(PowersCache) + 技能管理(SkillLevel/SetSkillLevel/AddSkill) | 无缓存,无技能管理 | 缺少属性系统和战力 |
| Hero 额外字段 | Attrs, AttrsCache, PowersCache | 无 | 缺少属性计算 |
| 测试 | 完整测试(TestCopy/TestInit/TestHeroesFormat/TestHeroEntity) | 无 | ❌ |

### 3.3 业务逻辑层 (internal)

| 功能 | ldl | ai4slg | 差距 |
|------|-----|--------|------|
| 英雄升级 | `heroes.UpgradeLevel()` + handler `StreamHeroUpgradeLevel` | ❌ | 缺失 |
| 英雄升星 | `StreamHeroUpgradeStar` (含碎片消耗) | ❌ | 缺失 |
| 技能升级 | `StreamHeroUpgradeSkill` (含条件检查) | ❌ | 缺失 |
| 荣誉升级 | `StreamHeroUpgradeHonorLevel` (含碎片组合) | ❌ | 缺失 |
| 英雄合成 | `StreamHeroSynthetic` | ❌ | 缺失 |
| 英雄空投 | `StreamHeroMapShow` | ❌ | 缺失 |
| 属性查询 | `StreamHeroQueryAttrBuild` | ❌ | 缺失 |
| 技能解锁 | `UnlockSkill()` (按等级/星级解锁) | ❌ | 缺失 |
| 行军回城 | `MarchBackArrive()` (英雄状态恢复/士兵归还/体力恢复) | ❌ | 缺失 |
| 属性刷新 | `refreshattr.Hero()` (多模块属性聚合) | ❌ | 缺失 |
| 推送 | `articlepush.PushHeroAsync()` | ❌ | 缺失 |

### 3.4 养成体系概念对比

| 养成维度 | ldl | ai4slg |
|----------|-----|--------|
| 等级 (Level) | ✅ 升级+Exp配置 | ✅ 有Level+Exp字段 |
| 星阶 (StarStage) | ✅ 升星+碎片分解 | ❌ 无 |
| 荣誉等级 (HonorLevel) | ✅ 荣誉升级+碎片组合 | ❌ 无 |
| 技能 (Skills) | ✅ 技能组(按level/star解锁+升级) | ⚠️ 有EquipSkills但无升级逻辑 |
| 属性 (Attrs) | ✅ 完整属性系统+模块化缓存 | ⚠️ 有Cultivates(5维培养)但无计算 |
| 战力 (Power) | ✅ 自动计算+缓存 | ❌ 无 |
| 阵位 (TeamID) | ✅ 队伍编成关联 | ❌ 无 |
| 展示状态 (ShowState) | ✅ 空投/回收/投放 | ❌ 无 |
| 培养(Cultivate) | ❌ 无专门Cultivate | ✅ 有5维培养字段 |
| 士兵 (Troops) | ✅ 独立士兵系统 | ⚠️ 有Troops字段 |
| 装备 (EquipWeapons) | ✅ 独立装备系统 | ⚠️ 有EquipWeapons字段 |

---

## 四、协议层对比

### 4.1 Stream 消息处理器

| 维度 | ldl | ai4slg |
|------|-----|--------|
| 协议注册 | `protocol_gen.go` 自动生成 + `protocol.go` 路由分发 | 空文件 |
| 英雄协议 | 7 个 (1100-1105, 1151) | ❌ |
| 建筑协议 | 多个 (升级/拆除/队列等) | ❌ |
| 物品协议 | 多个 (使用/合成等) | ❌ |
| 任务协议 | 多个 | ❌ |
| 队伍协议 | 多个 | ❌ |
| 装备协议 | 多个 | ❌ |
| …其他 | ~40 个领域处理器 | ❌ |
| Stream 入口 | `handler/role_stream.go` 完整实现 | `game.protocol.st.go` 空返回 |

### 4.2 RPC Unary 处理器

| ldl | ai4slg |
|-----|--------|
| 16 个独立处理器 (role/trade/union/article/battle/cross/sdk/radar/privacy/trial/zombie/mail/citycompetition/platform/rank) | `CreateRole` + `LoginOnce` 仅打日志 |

---

## 五、缺失的核心子系统清单

以下为 ldl 中有而 ai4slg 中 **完全缺失** 的子系统:

| 子系统 | ldl 复杂度 | 优先级 | 说明 |
|--------|-----------|--------|------|
| **物品系统 (Items)** | entity+model+internal | P0 | 背包是养成消耗的基础,没有物品系统所有养成无法运作 |
| **城建系统 (Builds)** | entity+model+internal | P0 | 核心玩法之一,角色等级依赖总部等级 |
| **队伍编成 (Teams)** | entity+model+internal | P0 | 英雄上阵/布阵,直接关联行军战斗 |
| **属性系统 (Attr)** | entity+model+internal | P0 | 角色资源/属性/VIP等基础数据 |
| **招募系统 (Recruit)** | entity+model+internal | P1 | 英雄获取途径 |
| **装备系统 (Equips)** | entity+model+internal | P1 | 英雄装备,关联养成 |
| **任务系统 (Task)** | entity+model+internal | P1 | 引导/日常/成就 |
| **商店系统 (Shop/Store)** | entity+model+internal | P1 | 资源流转 |
| **科技系统 (Tech)** | internal-only | P1 | 城建后期 |
| **联盟系统 (Union)** | 12+ entity + 10+ internal | P2 | 社交玩法 |
| **活动系统 (Activity)** | 17+活动类型 | P2 | 运营活动 |
| **邮件系统 (Mail)** | internal | P2 | 系统通知 |
| **特权系统 (Privilege)** | entity+model+internal | P2 | 月卡/VIP |
| **竞技场 (Arena)** | entity+model+internal | P2 | PVP玩法 |
| **雷达系统 (Radar)** | entity+internal | P2 | 探索玩法 |

---

## 六、优势分析: ai4slg 相较于 ldl 的差异化

尽管 ai4slg game 层大幅落后,但以下方面是 ldl 不具备的:

| 优势 | 说明 |
|------|------|
| **独立的地图核心引擎** | `services/internal/cores/` 是独立于 game 的地图引擎包,含 AOI/行军/MarchDo/战斗结算,ldl 无此模块 |
| **AOI 九宫格视野** | 基于 Screen 的视野管理,ldl 无此系统 |
| **行军执行器模式** | BaseMarch/SingleMarch/MultiMarch 模板方法模式,ldl 无此抽象 |
| **NPC/建筑覆盖层** | map_buildings 独立覆盖层设计 |
| **行军战斗流水线** | attack_march/assist_march/develop_march 按类型分离,ldl 战斗在 internal/battle |
| **泛型和函数式选项** | 使用 Go 泛型 + Options 模式,设计更现代化 |

**关键结论**: ldl 强在 **game 业务层的完整性和成熟度**, ai4slg 强在 **地图/行军/战斗引擎层的架构设计**。两者是互补关系 — ai4slg 需要补 game 层, ldl 没有 cores 引擎层。

---

## 七、文件级差异详情

### game 入口 (main.go)

| 注册服务 | ldl (17+) | ai4slg (2) |
|----------|----------|------------|
| GameStream | ✅ | ✅ (空实现) |
| GameService/Handler | ✅ | ✅ (空实现) |
| Trade | ✅ | ❌ |
| Battle | ✅ | ❌ |
| Union | ✅ | ❌ |
| Article | ✅ | ❌ |
| Cross | ✅ | ❌ |
| SDK | ✅ | ❌ |
| Radar/Privacy/Trial/Platform/Zombie/Mail/CityCompetition/Rank | ✅ | ❌ |
| TCC/UnionRank/UnionBoss | ✅ | ❌ |
| GameConfig | ✅ | ❌ |

### entity 层初始化

```go
// ldl: role.Init() 对每个实体模块调用 Init(db) → AutoMigrate
func Init(db *gorm.DB) {
    heroes.Init(db)   // → db.AutoMigrate(&model.RoleHero{})
    items.Init(db)    // → db.AutoMigrate(&model.RoleItem{})
    teams.Init(db)    // → db.AutoMigrate(&model.RoleTeam{})
    builds.Init(db)   // → db.AutoMigrate(&model.RoleBuild{})
    // ... 22 个模块
}

// ai4slg: 只初始化了 2 个模块
func Init(writeDB) {
    hero_skills.Init(writeDB)           // ✅
    hero_skillcollections.Init(writeDB) // ✅
    // ❌ 缺失: role_heroes, cultivate_costs, 以及其他所有模块
}
```

---

## 八、未来规划路线图

### Phase 1 — 补齐养成基础 (2-3周)

**目标**: 让已写的 hero/skill/cost 实体真正运作起来,打通从 DB → Entity → Role → 协议的基础链路

| 步骤 | 内容 | 参考 ldl |
|------|------|----------|
| 1.1 | **game_declarations 填充** — 定义游戏常量、接口、公共结构体 | `game.ty.go` 定义 CultivateType 等 |
| 1.2 | **Role 挂载子模块** — 在 `Role.New()` 中实例化 Heroes/Skills/SkillCollections/Costs | `role.New()` 实例化 24 个子模块 |
| 1.3 | **补齐缺失的 DB 操作** — cultivate_costs 的 DBCreate/DBGet/DBSave/DBDelete | 参考 heroes/db.go |
| 1.4 | **补充 AutoMigrate** — game.role.poller.go Init() 中添加 RoleHero + CultivateCost 迁移 | `role.Init()` 统一管理所有 migrate |
| 1.5 | **搭建协议路由** — 实现 `gateConnectDo()` 和 `protocol_gen.go` 的消息分发框架 | `handler/protocol.go` + `protocol_gen.go` |
| 1.6 | **实现英雄协议** — StreamHeroList (获取英雄列表) | hero/hero.go |
| 1.7 | **补齐 CultivateCost DB** — 完成 cost.db.go 的 CRUD | 参考 heroes/db.go |

**产出**: 英雄数据可从客户端拉取,Role 实体完整加载子模块

---

### Phase 2 — 核心养成玩法 (3-4周)

**目标**: 实现英雄养成的核心玩法逻辑,让玩家可以升级/升星/培养英雄

| 步骤 | 内容 | 参考 ldl |
|------|------|----------|
| 2.1 | **英雄升级** — LevelUp (消耗+等级提升+属性刷新) | `heroes.UpgradeLevel()` + handler `1100` |
| 2.2 | **英雄培养(Cultivate)** — 5 维属性培养(攻击/防御/智力/速度/迁城) | ai4slg 特有,Cultivates 字段已有 |
| 2.3 | **技能升级** — 消耗材料升级技能等级 | `StreamHeroUpgradeSkill` (1102) |
| 2.4 | **技能解锁** — 按等级/条件自动解锁新技能 | `UnlockSkill()` |
| 2.5 | **英雄合成** — 碎片合成新英雄 | `StreamHeroSynthetic` (1105) |
| 2.6 | **英雄锁/治疗** — 受伤英雄锁定恢复 | IsLocked 字段 + 治疗逻辑 |
| 2.7 | **技能收藏** — 收藏激活加成 | SkillCollection 已有数据结构 |
| 2.8 | **养成消耗系统** — CultivateCost 完整实现(消耗跟踪/重置) | 已有数据结构 |

**产出**: 英雄养成核心闭环: 升级→培养→技能→战斗→经验/受伤

---

### Phase 3 — 基础子系统 (4-6周)

**目标**: 搭建游戏运行必需的底层子系统

| 步骤 | 内容 | 优先级 |
|------|------|--------|
| 3.1 | **物品系统(Items)** — 背包/道具/消耗品 | P0 — 养成消耗基础 |
| 3.2 | **属性系统(Attr)** — 角色资源/属性/VIP | P0 — 角色基础数据 |
| 3.3 | **队伍编成(Teams)** — 英雄上阵/布阵 | P0 — 关联行军战斗 |
| 3.4 | **城建系统(Builds)** — 建筑升级/队列 | P0 — 核心玩法 |
| 3.5 | **协议注册表** — 自动生成协议号↔处理器映射 | P0 — 消息分发基础 |

---

### Phase 4 — 战斗与行军打通 (2-3周)

**目标**: 将 game 层的英雄养成数据与 cores 层的战斗/行军引擎连接

| 步骤 | 内容 |
|------|------|
| 4.1 | **英雄属性→战斗公式** — 养成属性注入 cores 战斗结算 |
| 4.2 | **行军队伍数据** — MarchInfo.Team 与 game Teams 同步 |
| 4.3 | **战斗结果回调** — 战后英雄经验/受伤/士兵损失处理 |
| 4.4 | **行军回城处理** — 英雄状态恢复/士兵归还/体力恢复 |

---

### Phase 5 — 子系统扩展 (6-8周)

**目标**: 按业务需求逐个补齐游戏子系统

| 步骤 | 内容 | 参考 ldl |
|------|------|----------|
| 5.1 | 招募系统 (英雄获取) | `entity/role/recruit/` |
| 5.2 | 装备系统 (英雄装备) | `entity/role/equips/` |
| 5.3 | 任务系统 (引导/日常) | `entity/role/task/` |
| 5.4 | 商店系统 (资源流转) | `entity/role/shops/` + `stores/` |
| 5.5 | 科技系统 (城建后期) | `internal/tech/` |
| 5.6 | 联盟系统 (社交玩法) | `entity/role/union/` + `internal/union/` |

---

### Phase 6 — 运营与高阶玩法 (长期)

| 步骤 | 内容 | 参考 ldl |
|------|------|----------|
| 6.1 | 活动系统(17+类型) | `entity/role/activities/` + `internal/activity/` |
| 6.2 | 特权系统(月卡/VIP) | `entity/role/privileges/` |
| 6.3 | 邮件系统 | `internal/mail/` |
| 6.4 | 竞技场 | `entity/role/arena/` + `internal/arena/` |
| 6.5 | 世界Boss | `entity/role/worldboss/` |
| 6.6 | RedisStream 事件总线 | `entity/init.go` 的 20+ handler |

---

## 九、建议优先级总结

```
现在 → Phase 1 (打通基础链路,2-3周)
         ↓
Phase 2 (核心养成玩法,3-4周) ← 👈 你问的"下一步"
         ↓
Phase 3 (基础子系统,4-6周)
         ↓
Phase 4 (战斗打通,2-3周)
         ↓
Phase 5 (子系统扩展,6-8周)
         ↓
Phase 6 (运营活动,长期)
```

### 👉 建议立刻开始的下一步

结合你已完成的内容(hero 实体/技能/收藏/消耗),最合理的下一步是 **Phase 1**:

1. **挂载 Role 子模块** — `Role.New()` 中实例化 Heroes/Skills/SkillCollections/Costs (半天)
2. **补齐 DB 和 migrate** — cost.db.go + 补充 AutoMigrate (半天)
3. **搭建协议路由** — 让客户端能调用 game (1-2天)
4. **实现 HeroList 协议** — 验证全链路通 (1天)

**总计约 1 周**就能让英雄数据从客户端可见,打通 DB→Entity→Role→gRPC 的完整链路,为后续的升级/培养等玩法开发奠定基础。
