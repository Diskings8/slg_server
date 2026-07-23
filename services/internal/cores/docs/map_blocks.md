# map_blocks — 地图区块

> 路径: `services/internal/cores/map_blocks/`  
> 文件: `map.block.st.go` · `map.block.func.go`

地图区块模块，将大地图按区块进行分区管理，提供区块粒度的空闲地块查找和元素计数功能。

---

## MapBlock

```go
type MapBlock struct {
    blocks      []blockPart     // 区块数组，index 从 1 开始（0 位留空）
    blockLength int32           // 每个区块的边长（格子数）
    config      MapConfigI      // 地图配置
}
```

### blockPart

```go
type blockPart struct {
    BlockID    BornBlockID      // 区块 ID（1~25）
    FreeMapLen atomic.Int32     // 空闲地块数量（原子操作，无需加锁读取）
    Count       map[int32]*atomic.Int32 // map[元素类型ID]数量
    freeData    map[MapID]struct{}      // 空闲地块集合
    notFreeData map[MapID]struct{}      // 非空闲地块集合
}
```

### 初始化

`NewMapBlock(mapData)` — 遍历地图数据，将每个格子归入对应区块的空闲集合。
地图被划分为 `ServerMapBlockCutNum`（25）个区块，呈 5×5 网格分布。

### 核心方法

| 方法 | 说明 |
|------|------|
| `CalcBlock(x, y)` | 根据坐标计算所属区块 ID，返回 `BornBlockID` |
| `CalcBlockMapID(mapID)` | 根据格子 ID 计算所属区块 ID |
| `Get(blockID)` | 根据区块 ID 获取区块数据 |
| `GetBlockByMapID(mapID)` | 根据格子 ID 获取所属区块 |
| `MapDataAdd(mapID, elementID)` | 标记地块被刷新元素占用（空闲→占用） |
| `MapDataDel(mapID, elementID)` | 标记地块刷新元素移除（占用→空闲） |
| `Range(f)` | 遍历所有区块 |
| `FirstXY(blockID)` | 获取区块左上角坐标 |

### blockPart 方法

| 方法 | 说明 |
|------|------|
| `TypeCount(elementID)` | 获取指定元素类型在当前区块的数量 |
| `FreeRandOne()` | 随机获取一个空闲地块 ID（返回 -1 表示无空闲） |
| `FreeCount()` | 原子读取空闲地块数量 |
| `FreeData()` | 返回空闲地块迭代器（Go 1.23 iter.Seq） |
| `NotFreeData()` | 返回已占用地块迭代器 |

### 设计说明

- 主要服务于后续的怪物刷新、资源分配等需要快速查找空闲地块的场景
- 空闲计数使用 `atomic.Int32`，无锁读取，高并发安全
- `MapDataAdd/Del` 自动维护空闲/占用集合的双向迁移
- 跨服区块同步等更复杂场景可在上层叠加

---

## 参考实现

参考自 `server/services/internal/gamemap/block/entity.go`，适配 cores 的类型系统：
- 使用 `cores_declarations.BornBlockID` 替代 `int32`
- 使用 `cores_declarations.MapConfigI` 替代 `gamemap.MapConfig`
- 使用 `map_datas.MapDataManager` 替代 `mapdata.MyMaps`
