# map_borns — 出生块管理

> 路径: `services/internal/cores/map_borns/`  
> 文件: `bigmap.born.st.go` · `sort.born.go` · `temp.born.st.go`

实现了 `cores_declarations.BornBlockI` 接口，管理玩家出生点的分配与回收。

---

## BigMapBornBlockManager — 大地图出生块管理器

```go
type BigMapBornBlockManager struct {
    BronCount    int32
    bornChan     chan cores_declarations.BornBlockID
    emptyBornMap hashtriemap.HashTrieMap[...]  // 空闲出生块
    useBornMap   hashtriemap.HashTrieMap[...]  // 已使用出生块
    reloadLocker sync.Mutex
}
```

### 双池状态机

```
Store ──→ emptyBornMap (空闲池)
              │
              ▼  Use()
         useBornMap  (使用池)
              │
              ▼  Free()
         emptyBornMap (空闲池)
```

| 方法 | 说明 |
|---|---|
| `Store(bornID, data)` | 存入空闲池 |
| `Load(bornID)` | 从空闲池加载 |
| `Use(bornID)` | 空闲 → 使用（`LoadAndDelete` + `Store`） |
| `Free(bornID)` | 使用 → 空闲（`LoadAndDelete` + `Store`） |
| `Delete(bornID)` | 从两个池中同时删除 |
| `Range(f)` | 遍历空闲块（带 channel + reload 机制） |

### Range 遍历优化

- 使用 `bornChan` 通道缓冲空闲块 ID
- 按 `blockSort` 优先级轮询：`[, 1, 2, 3]`
- 通道为空时触发 `reload()` 重新填充
- 两次重试仍为空则结束遍历

### blockSort — 优先级排序

```go
var blockSort = []int32{1, 2, 3}
```

---

## TempBornBlockManager — 临时出生块管理器

```go
type TempBornBlockManager struct {
    emptyBornMap hashtriemap.HashTrieMap[...]
    useBornMap   hashtriemap.HashTrieMap[...]
}
```

简化版实现，适用于临时地图（非持久化出生点）：

| 方法 | 与 BigMap 的区别 |
|---|---|
| `Store(bornID, data)` | 直接存入使用池 |
| `Load(bornID)` | 返回 nil, false |
| `Use(bornID)` | 同上（空 → 使用） |
| `Free(bornID)` | 同上（使用 → 空） |
| `Range(f)` | 遍历空闲池，每个 key 展开为单格 `map[int32]struct{}` |

---

## 设计要点

- **双池状态管理**: 空闲 ↔ 使用 的完整生命周期，确保同一出生块不被重复分配
- **接口抽象**: `BornBlockI` 支持两种不同场景（大地图 vs 活动地图）
- **并发安全**: 使用 `hashtriemap` 并发哈希 Trie 图，`reloadLocker` 防止重入
