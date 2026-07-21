# map_connects — 角色连接管理

> 路径: `services/internal/cores/map_connects/`  
> 文件: `connect.manager.st.go` · `map.connects.var.go` · `role.connect.st.go`

管理玩家与服务器的 gRPC 流连接，提供连接生命周期管理和基于 AOI 的消息推送。

---

## 全局连接管理器

```go
type allConnectManager struct {
    ctx        context.Context
    cancelFunc context.CancelFunc
    isStop     atomic.Bool
}
```

- 包初始化时自动创建单例（`init()` 函数）
- `ShutDown()` — 全局进程结束时关闭所有连接

---

## RoleConnect — 角色连接

```go
type RoleConnect struct {
    RwLock           sync.RWMutex
    stream           *rpc_streams.GrpcStreamServer
    roleID           uint64
    cityMapID        cores_declarations.MapID
    scaleLevel       cores_declarations.ScaleLevel
    curMapID         cores_declarations.MapID
    minMapScaleLevel cores_declarations.ScaleLevel
}
```

| 字段 | 说明 |
|---|---|
| `stream` | gRPC 流服务器 |
| `roleID` | 角色 ID |
| `cityMapID` | 主城地图 ID |
| `scaleLevel` | 当前屏幕缩放等级 |
| `curMapID` | 当前关注的地图 ID（AOI 位置） |
| `minMapScaleLevel` | 最小缩放等级（默认 ScaleLevel1） |

| 方法 | 说明 |
|---|---|
| `GetRoleID()` / `GetScreenMapID()` / `SetScreenMapID(mapID)` | 角色标识与视野位置 |
| `CheckInCity()` | 检查是否在城内（`curMapID == InvalidMapID`） |
| `Send(data)` | 通过 gRPC 流发送 `NodePacket` |
| `SetStream(stream)` / `GetStream()` | gRPC 流管理 |

---

## RoleConnectManager — 连接管理器

```go
type RoleConnectManager struct {
    connects map[uint64]*RoleConnect
    aoi      *map_aois.ScreenData
    sync.RWMutex
}
```

### 连接生命周期

| 方法 | 说明 |
|---|---|
| `NewRoleConnect(name, roleID, mapID, stream, receiveF)` | 创建连接，注册到 AOI，设置 gRPC 流 |
| `CloseRoleConnect(roleID)` | 关闭连接，从 AOI 退出 |
| `LoadRoleConnect(roleID)` | 查询连接 |
| `WaitDone(conn)` | 等待 gRPC 流结束并自动清理 |
| `SetRoleScreen(roleID, mapID)` | 更新角色 AOI 视野位置 |
| `Range(f)` | 遍历所有连接 |
| `CloseAll()` | 关闭所有连接 |
| `GetConnectRoleIDs()` | 获取所有在线角色 ID |

**创建流程**:
1. 校验 roleID 和全局停止状态
2. 检查是否已存在该角色连接（防止重复）
3. 创建 `RoleConnect`，设置 gRPC 流（含自定义 receive 回调）
4. 通过 `aoi.Move(conn, mapID)` 注册到 AOI 系统
5. 存入连接池

### 消息推送系统

| 方法 | 说明 |
|---|---|
| `PushToScreen(msgID, msg, mapIDList...)` | 按 AOI 九宫格推送消息（自动去重） |
| `PushToRoleID(msgID, msg, roleID)` | 推送给单个角色 |
| `PushToRoleIDs(msgID, msg, roleIDList...)` | 批量推送 |

**PushToScreen 流程**:
1. 遍历 `mapIDList`
2. 对每个 mapID 调用 `aoi.AroundConnects` 收集视野内角色（自动去重）
3. 序列化消息为 `NodePacket`
4. 逐角色调用 `Send`，断线时自动 `CloseRoleConnect`

---

## 设计要点

- **基于 AOI 的精准推送**: `PushToScreen` 自动按九宫格筛选可见玩家，实现"谁看到谁收到"
- **自动断线清理**: 推送时检测 gRPC 错误码（Canceled / Unavailable / Unknown）自动关闭连接
- **gRPC 流封装**: 通过 `rpc_streams.GrpcStreamServer` 管理流生命周期和并发
- **双参数复用 GC 优化**: 连接查询方法支持复用 map/切片参数
