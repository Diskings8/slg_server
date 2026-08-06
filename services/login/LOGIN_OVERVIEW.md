# Login — 账号登录服务

> 路径: `services/login`  
> 模块: `server.slg.com/services/login`

---

## 概述

`login` 是 SLG 服务端的**账号登录节点**（跨服全局单例，`instance=0`），提供注册、登录、区服列表、进入区服（建角）RPC。账号/渠道/角色映射/区服数据落 **`common_db_0`**（首个使用该库的服务）。

登录链路（客户端 → gateway → login）中，gateway 的 `switchForward` 已实现 **login 协议段转发**（按 MsgID 分类 → 调 login 节点的 `Do` → 回包客户端）；服务间经 gRPC + etcd 发现。game 协议段（推流下行）仍为后续工作。

核心设计取向：**账号身份（`account_name`）由游戏自己维护、全局唯一、与渠道无关**；渠道侧原生账号（SDK UID / openid 等）在独立的渠道绑定表中声明关联，多渠道可绑定同一账号 → 角色挂账号下，**跨渠道不丢角色**。

---

## 目录结构

```
login/
├── main.go                                    # 入口：DB(common_db_0) + 三表 Migrate + 渠道/区服种子 + ETCD + reflection
├── LOGIN_OVERVIEW.md
│
├── login_models/                              # GORM 模型（全部落 common_db_0）
│   ├── account.st.go                          # login_account 账户表（account_name 全局唯一）
│   ├── channel.st.go                          # login_channel 渠道声明表
│   ├── channel_account.st.go                  # login_channel_account 渠道账号绑定表（auth_info）
│   ├── role.st.go                             # login_role 账号×区服×角色映射
│   └── server.st.go                           # login_server 区服表（ID=server_id=game instance）
│
├── login_internals/
│   ├── login_accounts/                        # 账户 / 渠道绑定 / 角色映射存储
│   │   ├── account.store.func.go             # 事务建账号+首渠道绑定 / 查询 / last_login / 认证信息更新
│   │   ├── role.store.func.go                # 角色映射 CRUD
│   │   └── account.store_test.go
│   ├── login_channels/                        # 渠道声明存储 + 官方渠道种子
│   │   ├── channel.store.func.go
│   │   └── channel.store_test.go
│   ├── login_servers/                         # 区服列表存储 + 默认区服种子
│   │   ├── server.store.func.go
│   │   └── server.store_test.go
│   ├── login_tokens/                          # 进程内登录票据（TTL 24h）
│   │   └── token.manager.func.go
│   └── login_game_clients/                    # 按 server_id 发现对应 game 节点的门面
│       ├── game.client.st.go                 #   + RoleCreator 接口（测试可 mock）
│       └── game.client.func.go
│
└── login_handlers/
    └── login_servers/                         # AccountService（CreateAccount/LoginAccount/ServerList/EnterServer）
        ├── login.server.st.go                # LoginServer + SetStore 依赖注入
        ├── login.server.func.go              # 4 个 RPC 实现
        └── login.server_test.go              # bufconn RPC 冒烟（sqlite + mock game）
```

---

## 核心设计

### 1. 账号与渠道分离（三张表）

**`login_account` 账户表** — 账号身份本体：
`account_name`（**游戏侧账号，全局唯一，与渠道无关**）+ `password_hash`（md5(salt + name + ":" + pwd)）+ `status` + `last_login_server_id / last_login_role_id`。

**`login_channel` 渠道声明表** — 不同渠道在此声明：
`channel_type`（= 协议 `ChannelType` 枚举）+ `channel_name` + `secret`（第三方渠道 MD5 签名校验密钥）。**官方渠道（`Mine=0`）也只是渠道的一种**，种子启动幂等插入；第三方渠道后续在表中声明。

**`login_channel_account` 渠道账号绑定表** — 渠道原生账号 ↔ 游戏账户：
`account_id`（索引，**一账号可多渠道**）+ `channel_type/channel_account`（**联合唯一**，一渠道账号只映射一账户）+ `auth_info`（**记录该渠道登录时提供的认证信息**，如第三方 MD5 签名，留痕/后续可校验）。

> 设计意图：`account_name` 和 `channel_account` 两个命名空间彻底分开，互不干扰；绑定表把二者关联到同一 `account_id`，角色挂账号下 → 跨渠道共享、不丢角色。

### 2. 注册 / 登录流程

```
CreateAccount(channelType, account_name, password)
  ├─ 渠道已声明？
  ├─ account_name 全局查重（重复 → AlreadyExists）
  └─ 事务：建账号 + 首渠道绑定（binding.channel_account = account_name，auth_info 留空）

LoginAccount(channelType, account_name, password)
  ├─ 渠道已声明？
  ├─ ① 渠道绑定 (channelType, account_name) 命中 → 校验账号密码 + 刷新 auth_info
  ├─ ② 未命中 → 按全局 account_name + 密码匹配已有账号 → 命中则【自动绑定该渠道】（多渠道入口）
  └─ ③ 都未命中 → Unauthenticated
```

多渠道绑定**不需要额外协议**：同一 `account_name` + 密码从任意已声明渠道登录，都会解析到同一账号、同一角色列表。

### 3. 进入区服（EnterServer）

```
EnterServer(account_id, server_id, role_id, role_name, token)
  ├─ token / 账号 / 区服 校验
  ├─ 新建角色（role_id=0）：role_name 查重 → roleID = snowflakes.GenUUID()
  │     → 先调 game[server_id].CreateRole（出生点/主城/游戏数据由 game 落库）
  │     → 成功后写 login_role 映射行（失败则无脏写）
  ├─ 已有角色：校验该账号在该服拥有该角色
  ├─ 更新 account.last_login_server_id / last_login_role_id
  └─ 返回角色信息 → 客户端据此连对应服 gateway 进游戏
```

`roleID` 由 login 分配（全局唯一），game 直接使用——对应 `game.server.create_role.go` 注释"roleID 由 login/account 节点分配"。

### 4. 登录票据

进程内 `TokenManager`（`sync.RWMutex` + map，TTL 24h，`crypto/rand` 32B hex）。每次登录重新签发。单节点内存态足够当前需求；多实例共享时换 redis/etcd。**不使用 cacheconn**（其 redis client 未真正初始化）。

### 5. RPC 服务（AccountService）

| RPC | 说明 |
|-----|------|
| `CreateAccount` | 注册账号 + 首个渠道绑定（事务） |
| `LoginAccount` | 登录，返回账号 + 角色列表 + 登录票据 |
| `ServerList` | 区服列表（客户端选服） |
| `EnterServer` | 进入区服：新建角（调 game.CreateRole）或已有角 |
| `Do` | 统一协议入口（gateway 转发用），按 MsgID 路由 → 登录错误码映射 |

---

## 调用链

```
客户端（TCP: msgID + 请求 proto body）
  → gateway.switchForward（按 MsgID 分类）
      └─ login 协议段 → AccountService.Do → 回包（MessagePacket 信封：err_code + body）
      └─ game 协议段 → 后续接入

Do 内按 MsgID 路由：
  LoginCreateAccount/LoginAccount/LoginServerList/LoginEnterServer → 对应 handler

EnterServer 新建角：
  login → game[server_id].CreateRole（instance 即区服号）
           └─ game → worldmap.CreateRole（分配出生点/落主城）→ 建主城 → DBCreate 落库

ETCD 注册：/node:service:login/{instance}/
发现：gateway → login 通过 rpc_handlers hub（GetAccountServiceClient）；login → game 通过 NewClientHandler(server_id) → GetGameHandlerClient
```

- **调用方**：gateway 转发（未实现） / 后续其它节点
- **被调方**：`game`（建角）、自身 DB（`common_db_0`）

---

## 外部依赖

- `api/protocol/pb/pb_account` — 协议定义（AccountService + ChannelType 枚举）
- `api/protocol/pb/pb_role` — RoleSimpleInfo（角色列表 / 建角返回）
- `api/protocol/pb/pb_common` — CreateRoleReq（建角调 game）
- `common/conns/dbconn` — MySQL（`DB.Common` 库，common_db_0）
- `common/conns/rpcconn/rpc_handlers` — game 客户端发现（按 server_id）
- `common/conns/etcdconn` — ETCD 服务注册（NodeLoginService=60）
- `common/utils/snowflakes` — 账号/角色/绑定雪花 ID
- `common/servers` — 生命周期框架
