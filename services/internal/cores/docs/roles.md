# roles — 角色数据管理

> 路径: `services/internal/cores/roles/`  
> 文件: `role.data.st.go` · `role.data.copy.func.go` · `role.data.db.func.go` · `role.poller.func.go` · `role.poller.var.go` · `role.brief.st.go` · `role.queue.st.go`

---

## 1. Data — 角色数据

```go
type Data struct {
    RoleID          uint64
    Queue           map[int32][]*GenerateQueue
    Brief           *Brief
    LastConnectTime int64
    copyLock        *sync.RWMutex
    src             *Data
}
```

实现了 `common_declarations.DataI` 接口。

### 生命周期

| 方法 | 说明 |
|---|---|
| `NewRoleDataInfo(id)` | 构造函数 |
| `UniqueID()` / `CacheKey()` / `Tag()` | 标识接口 |
| `Init()` | 初始化 |
| `Reset()` | 重置所有字段 |
| `Marshal()` / `Unmarshal(b)` | JSON 序列化/反序列化 |
| `Save(isDelete)` | 持久化（panic: implement me） |

### 写时复制 (Copy-on-Write)

| 方法 | 说明 |
|---|---|
| `Copy(rw)` | 创建副本，使用 `copyLock` 延迟拷贝 |
| `IsCopy()` | 检查是否为副本 |
| `GetBrief()` | 获取 Brief（副本模式触发延迟深拷贝 `Clone()`） |
| `GetQueue()` | 获取 Queue（副本模式触发延迟深拷贝 `slices.Clone`） |

### 生成队列

| 方法 | 说明 |
|---|---|
| `AddQueue(queueKey, mapID)` | 添加地图到指定队列 |
| `ReleaseRoleQueue(queueKey, baseMapInfo)` | 从队列中释放指定地图 |

### 数据库 CRUD

| 方法 | 说明 |
|---|---|
| `DBCreate()` | 创建记录 |
| `DBDelete()` | 删除记录 |
| `DBSave()` | 保存记录 |
| `DBGet()` | 查询记录 |
| `Value()` / `Scan(input)` | gorm 序列化接口 |

---

## 2. Brief — 角色简略信息

```go
type Brief struct {
    RoleBrief *pb_role.RoleBrief
}

func (b *Brief) Clone() *Brief {
    return &Brief{RoleBrief: util_roles.CopyRoleBrief(b.RoleBrief)}
}
```

- `Clone()` — 深拷贝 `pb_role.RoleBrief`

---

## 3. GenerateQueue — 生成队列

```go
type GenerateQueue struct {
    MapID cores_declarations.MapID
}
```

---

## 4. 轮询器系统

**文件**: `role.poller.var.go` · `role.poller.func.go`

```go
var pollerManager *pollers.PollerManager[*Data]
var jsonCache = cache.New(10*time.Minute, 5*time.Minute)
```

### 初始化

`Init(ctx)` — 配置三级轮询周期：

| 周期 | 值 |
|---|---|
| 短轮询 | `crontabs.Pre30Seconds` |
| 中轮询 | `crontabs.Pre1Minutes` |
| 长轮询 | `crontabs.AHalfDay` |

`loader(id)` — 数据加载器：从数据库读取，未找到则返回空创建。

### API

| 方法 | 说明 |
|---|---|
| `GetPollerMgr()` | 获取轮询管理器 |
| `GetPoller(id)` | 获取角色轮询器 |
| `Get(id)` → `(data, freeFunc, releaseFunc, err)` | **标准获取模式**：获取数据 + 释放函数 + 保存函数 |
| `GetCopy(id)` | 获取副本（COW 模式） |
| `Close()` | 关闭轮询器 |

**jsonCache**: 缓存 `Marshal` 后的 JSON 字节，10 分钟过期，减少序列化开销。

---

## 设计要点

- **写时复制 (Copy-on-Write)**: `Data.Copy()` 创建轻量副本，`GetBrief()` / `GetQueue()` 在首次访问时通过 `copyLock` 触发源数据的深拷贝，读多写少场景大幅减少锁竞争
- **轮询器模式**: `PollerManager` 三级轮询周期（30s / 1m / 半天），平衡实时性与性能
- **JSON 缓存**: `jsonCache` 缓存序列化字节，避免重复 Marshal
- **双路径获取**: `Get()` 返回读写接口（data + save + release），`GetCopy()` 返回只读副本
- **GORM 集成**: `Value()` / `Scan()` 支持直接作为 gorm 模型字段序列化
