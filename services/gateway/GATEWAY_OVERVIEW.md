# Gateway — 网关接入服务

> 路径: `services/gateway`  
> 模块: `server.slg.com/services/gateway`

---

## 概述

`gateway` 是 SLG 服务端的**客户端统一接入网关**：对外开 **TCP** 长连接承接客户端，对内通过 **gRPC + etcd 发现** 转发到各业务节点。不持有任何业务逻辑，只做协议分类、转发与回包。

当前实现的转发协议段：

- **login 协议段**（注册/登录/区服列表/进入区服）→ 同步调 login 节点 `Do`
- **game 协议段**（业务 1000001~ + 相机 42200~）→ 进服后经**双向流**到 `game[server_id]`

`game` 协议段采用**懒连接双向流**：首次 game 请求时才打开到对应区服 game 节点的流，`EnterServer` 成功时捕获 `serverID/roleID` 作为后续路由依据。

---

## 目录结构

```
gateway/
├── main.go                                    # 入口：gRPC + TCP 双服务 + etcd 注册 + 生命周期
│
├── session_gateways/                           # 客户端会话与转发（核心）
│   ├── session.st.go                         # Session 结构体 + 读循环/连接管理
│   ├── session.forward.func.go               # switchForward 协议分类 + login 段转发
│   ├── session.forward.game.func.go          # game 段双向流转发（懒连接 + 下行回包）
│   └── session.forward_test.go               # switchForward 单测（fake conn + mock login）
│
├── mix_server_gateways/                        # gRPC 混合服务（节点间通信入口）
│   └── mix.server.st.go                      # MixServer：GatewayService（当前仅 NotifyInfo）
│
└── gateway_internals/                          # 内部基础设施
    └── gateway_rpc_clients/                   # 出向 RPC 客户端门面
        └── gateway.rpc.client.st.go          # login（instance=0）+ game（按 server_id）懒连接 hub
```

---

## 核心设计

### 1. 双端口服务（main.go）

| 服务 | 承载 | 职责 |
|---|---|---|
| **TCP**（`servers.NewTcpBuilder`） | 客户端长连接 | 每连接一个 `Session`，`RunToReceiveFromConn` 读包 → 转发 |
| **gRPC**（`servers.NewGrpcBuilder` + `MixServer`） | 节点间通信 | 注册 `GatewayService`（当前仅 `NotifyInfo` 占位） |

两个服务由 `servers.NewLifecycle` 统一管理启动/关闭；系统信号 → `ctx.cancel()` → 优雅退出。

### 2. 包协议约定（客户端 ↔ gateway）

```
上行：TCP header(msgID) + body = 序列化请求 proto（如 LoginAccountReq）
下行：TCP header(msgID) + body = 序列化 common.MessagePacket（err_code + body）
```

- **`MessagePacket`**：`body + dev_msg + err_code`，客户端回包的**统一信封**
- **`NodePacket`**：`role_id + msgId + MessagePacket`，节点间传输的统一信封
- gateway 向后端转发时解包 `MessagePacket.Body`（业务 proto 字节流）装入 `NodePacket`，后端响应再取回 `NodePacket.Message` 还原信封

### 3. 协议分段路由（switchForward）

`Session.switchForward` 按 `MsgID` 分类目标节点：

| 分段 | MsgID 范围 | 目标 |
|---|---|---|
| login | `LoginCreateAccount/LoginAccount/LoginServerList/LoginEnterServer`（2000001~） | login 节点 `Do` |
| game 业务 | `1000001 ~ 1009999` | game 双向流 |
| game 相机 | `42200 ~ 42299` | game 双向流 |
| 其他（含下推段 10000001+） | — | 客户端不应上行 → `ProtocolNotFound` 错误包 |

### 4. login 段：同步请求-响应（`forwardToLogin`）

```
forwardToLogin(packet)
  ├─ 取 login 客户端（gateway_rpc_clients.Client().Login()，instance=0 懒连接）
  ├─ EnterServer 转发前 captureEnterServerID（从请求体捕获 server_id）
  ├─ cli.Do(ctx, NodePacket{MsgId, Message{Body}})   ← 同步 RPC
  ├─ 成功后 captureEnterServerRoleID（从响应体捕获 role_id）
  └─ writeNodePacket：回包客户端（MessagePacket 信封）
```

**进服状态捕获**是 gateway 的核心状态机：`EnterServer` 请求/响应双向捕获后，`Session` 才具备 `serverID + roleID`，后续 game 协议才能路由。

### 5. game 段：懒连接双向流（`forwardToGame`）

```
forwardToGame(packet)
  ├─ serverID/roleID 为空（未进服）→ 拒绝（Failed 错误包）
  ├─ gameStream 为空 → connectGameStream(serverID, roleID)
  │     └─ Client().Game(serverID).GetGameServiceClient().Stream(ctx)
  │           └─ 首包握手 GameServiceNodePacketReq{RoleId: roleID}
  ├─ stream.Send(GameServiceNodePacketReq{RoleId, Packet: NodePacket{MsgId, Message{Body}}})
  └─ 记录 lastSeq = 请求 seq（供下行响应匹配）

recvFromGame（后台 goroutine）
  ├─ rsp.GetPacket() → writeNodePacket → 客户端 TCP
  ├─ 请求-响应：seq 回最近一次上行请求的 seq（lastSeq）
  └─ 推送（isPushMsgID：MsgID ≥ 10000000）→ seq=0
```

| 事件 | 处理 |
|---|---|
| TCP 断开 | `RunToReceiveFromConn` 返回 → `cleanupGameStream()` |
| game 流断开 | `recvFromGame` 收 `err` → `cleanupGameStream()` |
| 流清理 | 幂等：`cancel()` + `CloseSend()`，game 侧 `gateConnectDo` 收 Done → `Offline` 下线 |

### 6. 出向 RPC 客户端（gateway_rpc_clients）

包级单例 `GatewayRpcClient`，懒连接：

| 访问器 | 目标 | 说明 |
|---|---|---|
| `Login()` | login 节点 | instance=0，懒建 hub + 缓存 client |
| `Game(serverID)` | game 节点 | `serverID → hub` 映射，instance = serverID（一进程 = 一区服） |
| `SetLoginClient()` | — | 测试注入 / 节点切换 |

---

## 消息流全景

```
客户端 ──TCP──▶ gateway
                 │
        switchForward（按 MsgID 分类）
        ├── login 段 ──gRPC Unary──▶ login.Do ──▶ 回包客户端
        └── game 段 ──gRPC 双向流──▶ game.Stream（懒连接）
                上行  GameServiceNodePacketReq{RoleId, NodePacket}
                下行  GameServiceNodePacketRsp{Packet} ──▶ 回包客户端
                      └─ game 主动下推（Push 段 MsgID≥10000000）→ seq=0
```

---

## 通信

```
客户端 ──TCP──▶ gateway（本服务）
gateway ──gRPC──▶ login  （AccountService.Do，instance=0）
gateway ──gRPC──▶ game   （GameService.Stream，instance=server_id）

ETCD 注册：/node:service:gateway/{instance}/
发现：gateway → login/game 经 rpc_handlers hub → etcd GetNodeTypeServerAddrByInstance
```

- **被调方**：login（login 段）、game（game 段双向流）
- **调用方**：客户端（TCP）
- **下推方**：game 经双向流主动推送（地图/行军信息、维护/版本通知）

---

## 外部依赖

- `api/protocol/pb/pb_protocol` — MsgID 枚举（协议分段依据）
- `api/protocol/pb/pb_common` — NodePacket / MessagePacket 信封
- `api/protocol/pb/pb_account` — login 协议体（EnterServer 请求/响应捕获）
- `api/protocol/pb/pb_game` — GameService 双向流
- `api/protocol/pb/pb_gateway` — GatewayService（节点间占位）
- `common/conns/netconn` — TCP 连接抽象（`tcp_conn` + `packets`）
- `common/conns/rpcconn/rpc_handlers` — 出向 client hub（login/game 发现）
- `common/conns/etcdconn` — ETCD 服务注册（NodeGatewayService）
- `common/servers` — 生命周期框架（gRPC + TCP 统一管理）

---

## 当前状态

| 子系统 | 状态 | 说明 |
|--------|------|------|
| TCP 会话 + 读循环 | ✅ | `RunToReceiveFromConn` + 断开清理 |
| login 段转发 | ✅ | 同步 Do + EnterServer 双向捕获 |
| game 段双向流 | ✅ | 懒连接 + 握手 + 上下行 + 推送 seq 处理 |
| 出向 RPC 客户端 | ✅ | login/game 懒连接 hub |
| gRPC MixServer | ⚠️ | `GatewayService` 仅 `NotifyInfo` 占位，其余待实现 |
| 后端下推扩展 | ⬜ | `RunToSendToConn` 为 TODO（未来承载 game 侧更多下推通道） |

---

## 测试

`session.forward_test.go` — 不依赖 etcd/redis，用 fake conn 捕获回包 + mock login 客户端注入，覆盖：

- 协议分类（login / game / 下推非 game）
- 未进服发 game 协议 → 拒绝（Failed）
- EnterServer 成功后 serverID/roleID 捕获
- login 转发回包信封（seq 回显 + MessagePacket 结构）
- login 节点不可达 → SystemBusy 错误包
- 不支持协议 → ProtocolNotFound 错误包
