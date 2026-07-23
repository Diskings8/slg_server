# SLG Server 命名规范

> 适用于 `services/internal/cores` 及 `common/` 下所有 Go 源码。

---

## 目录命名

全小写 `snake_case`，Go 包名与目录名一致（去下划线）。

| 层级 | 规范 | 示例 |
|---|---|---|
| 顶层包 | `{领域}_{子领域}` | `map_aois/`, `marchdos/`, `roles/` |
| 子包 | `{功能}_{限定}` | `attack_march/`, `map_buildings/` |
| 文档 | `docs/` | `services/internal/cores/docs/` |

> **例外**：`marchdos/` 的子包 Go 包名为截短形式 —— `attack_march/ → package attack`，去掉 `_march` 后缀。

---

## 文件命名

```
{领域}.{概念}.{类型后缀}.go
```

使用**点号分隔的多段结构**，从广到窄分层表达文件身份：

- **领域**：所属模块（`map`, `role`, `march`, `core`, `manager`...）
- **概念**：业务实体，可继续细化（`data.screen`, `data.db`, `poller`...）
- **类型后缀**：文件职责分类（见下表）

### 类型后缀体系

| 后缀 | 含义 | 示例 |
|---|---|---|
| `.st.go` | **Struct/Type** 结构体定义 | `march.info.st.go`, `role.data.st.go` |
| `.func.go` | **Function** 方法/逻辑实现 | `map.block.func.go`, `manager.tick.func.go` |
| `.var.go` | **Variable** 包级变量/init | `map.manager.var.go`, `role.poller.var.go` |
| `.const.go` | **Constant** 常量枚举 | `core.const.go` |
| `.if.go` | **Interface** 接口定义 | `core.if.go` |
| `.db.func.go` | **Database Func** 数据库操作 | `march.infomanager.db.func.go` |
| `.copy.func.go` | **Copy Func** 深拷贝方法 | `role.data.copy.func.go` |

> 按**领域职责**（st/func/var/const/if）分类，而非按架构层级。**不**使用 handler/server/repo 等后缀。

### 命名链示例

```
map.block.st.go           — 地图 → 区块 → 结构体
manager.march.func.go     — 管理器 → 行军 → 函数
role.data.db.func.go      — 角色 → 数据 → 数据库 → 函数
aoi.screen.st.go          — AOI → 屏幕 → 结构体
bigmap.born.st.go         — 大地图 → 出生块 → 结构体
march.info.st.go          — 行军 → 信息 → 结构体
```

---

## 一览

| 元素 | 规范 |
|---|---|
| 目录名 | `snake_case` |
| Go 包名 | 目录名去下划线后全小写 |
| 文件名 | `{domain}.{concept}.{suffix}.go` |
| 类型后缀 | `.st` / `.func` / `.var` / `.const` / `.if` |
| 文件扩展名 | `.go` |
