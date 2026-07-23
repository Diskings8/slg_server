# 地图模块优化追踪

> 对比参考仓库后梳理出的待办改进项。
> 按优先级分 P0（紧急）/ P1（重要）/ P2（可优化），建议按优先级 + 模块维度推进。

---

## 总览

| 优先级 | 数量 | 关键领域 |
|--------|------|----------|
| 🔴 P0 | 0 | 三项全部完成 ✅ |
| 🟠 P1 | 11 | 架构分层、AOI、数据模型、持久化、推送 |
| 🟡 P2 | 11 | 性能优化、锁模式、接口完备性、测试覆盖 |

---

## 🔴 P0 — 紧急缺失

### P0-1: `BaseMarch.CallBack`/`CallBackNow` 完整状态机

- **状态：** ✅ 已完成
- **涉及文件：**
  - `marchdos/base.march.st.go` — 新增回调 opt 链字段 + 注册方法 + 模板方法 + BackArrive opt 链 + ReTry
  - `marchdos/single.march.st.go` — SingleMarch 召回逻辑 + BackArrive（锁定/推送/删除）+ ReTry + AOI 路径重算
  - `marchdos/multi.march.st.go` — MultiMarch 召回逻辑 + BackArrive（锁定/推送/删除）+ ReTry + AOI 路径重算
  - `marchs/march.infomanager.func.go` — `MapAttributeMarchCallBack` 改为直接访问字段避免死锁
- **参考：** 参考仓库 `marchdo/base.go:356-455`
- **设计说明：**
  - 沿用现有 opt 链模式，新增 `prepareCallBackOpts → callBackOpts → finishCallBackOpts` 三阶段
  - 同理 CallBackNow，子类型通过 `Add*CallBackOpt()` 插入自定义逻辑
  - 状态守卫：`Back`/`Error`/`Battle` 跳过
  - 时间重算：等比例对称（走了多久就需要多久返回）
  - `MapAttributeMarchCallBack` 更新地图行军索引
  - `TickerAddMarch` 重新注册到达 ticker
  - `UpdateMarchPush` 推送客户端
  - BackArrive 沿用 `prepareBackArriveOpts → backArriveOpts → finishBackArriveOpts` 模板
  - ReTry 实现 3 次重试（间隔 100ms），中途检查状态变更提前退出
  - AOI 重算：清空 `AoiBlock`/`PassingAoiBlock` → `MarchAOISetupSingle` 重新计算
  - `Do()` 分流：检测 `MarchState_Back` 时路由到 `BackArrive()` 而非正常到达流程

---

### P0-2: 战斗到达流水线

- **状态：** ✅ 已完成
- **涉及文件：**
  - `marchdos/march.factory.go` — 注册式工厂 + `DefaultMarchTickHandler` tick 分发
  - `marchdos/attack_march/march.func.go` — 串联各阶段（Prepare → Do → Finish）
  - `marchdos/attack_march/march.result.st.go` — 战斗结果数据结构
  - `marchdos/attack_march/march.battle.func.go` — 战斗结算引擎（逐层对战 + PvE/攻城）
  - `marchdos/attack_march/march.settle.func.go` — 战损结算 + 占领判定 + 溃败处理
  - `marchdos/attack_march/march.push.func.go` — 战报推送（复用现有推送通道）
  - `marchdos/attack_march/march.event.func.go` — 事件触发（预留空桩）
  - `marchdos/single.march.st.go` — Do() 状态分流、召回持久化（Save）
  - `marchdos/march.tick.func.go` — 默认 tick 处理（到期检查 + marchLocker + 状态分流）
  - `map_datas/map.info.st.go` — 新增 `Occupy()`、`GetOwnerID()`、`GetOverlayBuilding()`
- **参考：** 参考仓库 `marchdo/monster.go` + `marchdo/hall.go` + `handler/mapsearch.go`
- **流水线：** `checkTargetLegality → settleBattle → processBattleResult → pushBattleResult`
- **子任务：**
  - [x] 战斗结算（多层对战：assist → stay → idle → PvE/攻城）
  - [x] 战损结算（按比例分配存活数）
  - [x] 占领判定（`occupyTile` / `tryOccupy`）
  - [x] `IsDefeated` 溃败标记 + 状态转换
  - [x] `MarchTickHandler` tick 分发 + 并发控制（marchLocker）
  - [x] Do() 状态分流（Back 走 BackArrive，其他走正常到达）
  - [x] CallBack / CallBackNow 异步持久化（Save）
  - [ ] 战报推送（`PushMarchBattleResult`，区分集结/单人）— 第二阶段 TODO：需补充 PB 协议
  - [ ] 事件触发 — 预留空桩

---

### P0-3: 每格行军聚合管理（`MapAttribute`）

- **状态：** ✅ 已完成
- **涉及文件：**
  - `marchs/map.attribute.st.go` — 新增 `GetMarchIDList()`、`GetMapMarchLen()`、`CleanAllMarch()`
  - `marchs/march.infomanager.func.go` — 修复 `findMarchList` 值传递 bug（改为返回切片）
- **参考：** 参考仓库 `march/map.go` + `march/map_assist.go`
- **说明：** MapAttribute 此前已具备基础功能（marchMap + assistSlice），补全缺失的查询和清理方法
- **子任务：**
  - [x] `GetMapMarch(mapID)` — 通过 `MapAttributeGet` 获取
  - [x] `AssistArrive` / `AssistCallBack` — 驻守到达/返回（已有）
  - [x] `MarchChangeToMapID` — 通过 `MapAttributeMarchChange` 实现（已有）
  - [x] `GetMarchIDList()` — 新增，获取地块上所有行军 ID
  - [x] `GetMapMarchLen()` — 新增，基于 `hashmaps.Map.Len()` 的 O(1) 查询
  - [x] `CleanAllMarch()` — 新增，清空地块行军+驻军（测试用）
  - [x] `findMarchList` 传参修复 — 原值传递导致 Init 加载不到数据

---

## 🟠 P1 — 重要优化

### P1-1: 抽取独立 handler 层

- **文件：** 新增 `map_handler/` 或类似目录
- **参考：** 参考仓库 `handler/`（handler 只做请求校验和响应组装）
- **说明：** 当前 `MapManager` 承担了 RPC 校验 + 编排双重职责。抽取 handler 层将 RPC 入口与编排逻辑分离。
- **状态：** [ ] 待开始

### P1-2: `marchdos/` 工厂模式重构

- **文件：** `marchdos/march.factory.go`
- **参考：** 参考仓库 `marchdo/interface.go:50-120` 显式 `NewMarchDo` 工厂函数
- **说明：** 当前使用 `init()` 自注册模式，可扩展性好但链路追踪难。改为显式工厂函数，提升可读性和可测试性。
- **状态：** [ ] 待开始

### P1-3: AOI 行军分离 — `crossMarch` / `passingMarch`

- **文件：** `map_aois/aoi.screen.st.go` / `data.screen.func.go`
- **参考：** 参考仓库 AOI Screen 内维护 `march`（起点/终点在 Screen 内）和 `crossMarch`（仅经过）双集合
- **说明：** 减少不必要的 AOI 事件推送，经过行军不触发视野更新
- **状态：** [ ] 待开始

### P1-4: 实现 `MovePath` 路径采样方法

- **文件：** `map_aois/data.screen.func.go`
- **参考：** 参考仓库 `aoi/aoi.go:373-416` 按 `step=ScreenWeight*2` 跳跃采样
- **说明：** 行军路径 AOI 推送时，避免逐格遍历，按步长采样
- **状态：** [ ] 待开始

### P1-5: 双存储策略（`bigMapData` + `smallMapData`）

- **文件：** `map_datas/map.info.st.go` / `map.datamanager.func.go`
- **参考：** 参考仓库 `mapdata/maps.go:86-87`
  - `bigMapData []MapInfo` — 值类型 slice，紧凑内存，减少 GC 扫描
  - `smallMapData map[int32]*MapInfo` — 稀疏地图，用 map 节省内存
- **说明：** 当前只有单一 map 存储。区分稠密大地图（1000x1000）和稀疏战场地图，选择合适的存储结构。
- **状态：** [ ] 待开始

### P1-6: 批量异步持久化

- **文件：** `marchs/march.infomanager.db.func.go` / `map_datas/map.datamanager.func.go`
- **参考：** 参考仓库 `asyncsave2` — `Save()` 仅设 `isNeedSave` 标记，定时批量写入 DB，内部去重合并
- **说明：** 避免每次操作直接写 DB，改为标记 → 批量刷盘模式
- **状态：** [ ] 待开始

### P1-7: 抽取独立 validator 层

- **文件：** 新增 `march_validator.go` 或类似
- **参考：** 参考仓库 `handler/march_validator.go` 使用 `createMarchCtx` / `changeMarchCtx` 传递校验结果
- **说明：** 将创建行军、变更行军等操作的校验逻辑从 handler/manager 中抽离
- **状态：** [ ] 待开始

### P1-8: 战争情报推送（`PushWarInformation`）

- **文件：** `map_managers/manager.push.func.go`
- **参考：** 参考仓库 `manage/interface.go:413-489` 区分攻击方/被攻击方，分别按角色/联盟推送，远程服通过 gRPC 跨服推送
- **说明：** 攻防双方收到不同维度的战报
- **状态：** [ ] 待开始

### P1-9: `RoleMapManager` 角色→地块路由表

- **文件：** 新增 `map_datas/role.map.manager.func.go` 或类似
- **参考：** 参考仓库独立于 `MapInfo` 的 `RoleMapManager`，支持按 `UnionID` 查询联盟成员位置、按 `RoleID` 查询位置
- **说明：** 避免遍历全 map 查找角色位置，建立倒排索引。已在 `map.union.func.go` 中实现 `UnionMemberMapIDs`
- **状态：** [ ] 待开始

### P1-10: `Init()` 集结重建和异常恢复

- **文件：** `marchs/march.assemble.func.go` / `march.infomanager.func.go`
- **参考：** 参考仓库 `march/interface.go:18-99` 的 `Init()` 从 DB 加载后重建 `assembleMarchMap`，处理主行军缺失的异常
- **说明：** 服务器重启时重建集结关系，处理孤儿集结行军
- **状态：** [ ] 待开始

### P1-11: AOI 推送去重

- **文件：** `map_managers/manager.push.func.go`
- **参考：** 参考仓库 `upMapSync()` 先按 mapID 收集需推送的角色，再统一推送，避免 AOI 重叠区域的重复推送
- **说明：** AOI 九宫格边界重叠区域可能重复推送同一条消息给同一角色
- **状态：** [ ] 待开始

---

## 🟡 P2 — 可优化

### P2-1: RAII 锁包装器

- **文件：** `map_datas/` 或新增 `common/`
- **参考：** 参考仓库 `LockMapSlice`（`mapdata/maps.go:48-59`）自动解锁
- **说明：** 用 `defer` + 包装器模式避免手动 lock/unlock 遗漏

### P2-2: 完整 `MarchDo` 接口定义

- **文件：** `marchdos/march.interface.go`
- **说明：** 当前 `MarchDoFuncHandleI` 缺少 `CallBackNow`, `Accel`, `BackArrive`, `ReTry` 方法定义

### P2-3: 九宫格边界检查 switch-case 归整

- **文件：** `map_aois/data.screen.func.go`
- **参考：** 参考仓库清晰的边界处理（左边界 case 1、右边界 case 0）

### P2-4: `AroundConnects` 复用参数，减少 GC 分配

- **文件：** `map_aois/data.screen.func.go`
- **参考：** 参考仓库传入 `map[uint64]MapRoleConnect` 避免每次分配新 map

### P2-5: JSON 扩展字段模式

- **文件：** `map_datas/map.info.st.go`
- **参考：** 参考仓库 `MapInfo` 内嵌 `*RadarExcavation`, `*PrivacyTask` 等 `*Struct` 类型
- **说明：** 零值为 nil 时不占空间，配合 GORM json 序列化

### P2-6: 加速记录链 `AccelTime`

- **文件：** `marchs/march.info.st.go`
- **参考：** 参考仓库 `AccelTime []AccelRecord` + `GetEffectiveUseTime()` 按比例计算等效已用时间
- **说明：** 用于行军位置插值计算

### P2-7: `CreateMarchInBatches` 批量创建行军

- **文件：** `marchs/march.infomanager.func.go`
- **参考：** 参考仓库 `CreateMarchInBatches()`（`march/interface.go:149-179`）支持一次创建多个行军并批量入 DB
- **说明：** 当前已有该功能

### P2-8: 通过消息队列推送战报

- **文件：** 与 `marchdos/` + 消息队列集成
- **参考：** 参考仓库 `BackArrive()` 通过 `redisstream.PubProtoMessage` 将 `MarchBackArrive` 消息发送到角色服务
- **说明：** 实现跨服务解耦

### P2-9: `mapViewAssembler` 组装模式

- **文件：** `map_managers/manager.push.func.go`
- **参考：** 参考仓库先序列化基础信息，再通过 `mapViewAssembler` 按类型补全（角色头像、同盟信息等）
- **说明：** 避免一次性全量组装，按需补全

### P2-10: `sync.Pool` 复用 PB 对象

- **文件：** 全局或在推送相关文件
- **参考：** 参考仓库 `MapPBGet()` / `MapPBPut()`（`manage/interface.go:37-57`）

### P2-11: 单元测试覆盖

- **文件：** `marchdos/*_test.go`
- **参考：** 参考仓库有 62 个 `*_test.go` 文件覆盖各个包
- **说明：** 参考使用真实 DB（`base_test.go`）的测试基座模式

---

## 按模块维度聚合

| 模块 | P0 | P1 | P2 |
|------|----|----|----|
| `marchdos/`（行军执行） | P0-1 ✅, P0-2 ✅ | P1-2 | P2-2, P2-8 |
| `marchs/`（行军数据） | P0-3 ✅ | P1-10 | P2-6, P2-7 |
| `map_aois/`（AOI 视野） | — | P1-3, P1-4 | P2-3, P2-4 |
| `map_datas/`（地图数据） | — | P1-5, P1-9 | P2-1, P2-5 |
| `map_managers/`（地图管理） | — | P1-1, P1-8, P1-11 | P2-9, P2-10 |
| 持久化层 | — | P1-6 | — |
| 新增 validator/ | — | P1-7 | — |
| 测试 | — | — | P2-11 |

---

## 建议推进顺序

```
阶段 1（P0 补齐）✅ 全部完成
├── P0-2 战斗流水线 ← 最紧急，核心玩法 ✅
├── P0-1 CallBack 状态机 ✅
└── P0-3 MapAttribute 聚合 ✅

阶段 2（P1 基础架构）
├── P1-1 handler 层抽取 ← 架构清晰化前提
├── P1-7 Validator 分离 ← 代码质量前提
├── P1-2 工厂模式重构 ← 可测试性
├── P1-5 双存储策略 ← 性能基础
└── P1-6 批量异步持久化 ← 性能基础

阶段 3（P1 AOI 与推送）
├── P1-3 crossMarch 分离
├── P1-4 MovePath 采样
├── P1-11 AOI 推送去重
├── P1-8 战争情报推送
└── P1-9 RoleMapManager

阶段 4（P2 优化）
├── P2-1 ~ P2-11 按需推进
└── P2-11 测试覆盖穿插各阶段
```
