# SLG Server

> 项目路径: `E:\gamewrokspace\ai4slg\slg_server`  
> 模块: `server.slg.com`  
> 语言: Go

---

## 游戏设计目标

1. **视野系统** — 前端拖动地图，服务端基于 AOI 九宫格推送视野内格子 & 行军更新
2. **主城** — 玩家主城占地 3×3（9格），支持 Normal（固定）和 Portable（便携）两种状态
3. **出征** — 点击地块选择出征队伍，行军到达即触发战斗
4. **战斗** — 立即结算的行军结果（非回合制），到达出结果
5. **英雄养成** — 核心养成线，hero 培养是玩法驱动力
6. **格子争夺** — 占领空地 / 放弃已占领 / 攻占敌对目标
7. **城池攻占** — NPC 城市和玩家城市的攻防，涉及联盟占领、驻军等

---

## 项目结构

```
├── api/          # protobuf 协议定义与生成代码
├── common/       # 公共工具库（hashmaps, pollers, conns, loggers...）
├── envs/         # 环境配置
├── rpc_tools/    # RPC 工具
├── scripts/      # 构建/同步脚本
└── services/     # 业务服务
    └── internal/cores/    # 核心游戏逻辑 ← 主要工作区
```

---

## 项目规范

| 规范 | 文档 |
|---|---|
| 文件 & 目录命名规范 | [docs/naming-convention.md](docs/naming-convention.md) |

---

## 文档地图

所有模块文档集中在源码同级的 `docs/` 子目录中，以 `MODULE_OVERVIEW.md` 为入口索引。

### cores 模块

入口: `services/internal/cores/CORES_OVERVIEW.md`

| 子包 | 详细文档 |
|---|---|
| 核心声明 | [docs/cores_declarations.md](services/internal/cores/docs/cores_declarations.md) |
| AOI 视野管理 | [docs/map_aois.md](services/internal/cores/docs/map_aois.md) |
| 地图区块 | [docs/map_blocks.md](services/internal/cores/docs/map_blocks.md) |
| 出生块管理 | [docs/map_borns.md](services/internal/cores/docs/map_borns.md) |
| 角色连接 | [docs/map_connects.md](services/internal/cores/docs/map_connects.md) |
| 地图数据 & 建筑 | [docs/map_datas.md](services/internal/cores/docs/map_datas.md) |
| 地图管理器 | [docs/map_managers.md](services/internal/cores/docs/map_managers.md) |
| 行军信息 & 队伍 | [docs/marchs.md](services/internal/cores/docs/marchs.md) |
| 行军执行器 | [docs/marchdos.md](services/internal/cores/docs/marchdos.md) |
| 角色数据管理 | [docs/roles.md](services/internal/cores/docs/roles.md) |

### 新增模块文档的约定

- 模块入口文档统一命名 `{MODULE}_OVERVIEW.md`，放在模块根目录
- 子包详细文档放在模块根目录下的 `docs/` 子目录
- 在本文件的"文档地图"中补充入口链接
- 保持文档精炼：概述 + 结构 + 关键设计，细节放子文档

---

## 常用命令

```bash
# 运行服务
go run services/...

# 构建
go build ./...

# proto 同步到客户端
pwsh -File scripts/sync_proto_client.ps1
```
