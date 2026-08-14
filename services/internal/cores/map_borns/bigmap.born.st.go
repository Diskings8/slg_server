package map_borns

import (
	"math/rand/v2"
	"sync"

	"github.com/go4org/hashtriemap"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.BornBlockManagerI = (*BigMapBornBlockManager)(nil)

// NewBigMapBornBlockManager 构造大地图出生块管理器
// bornChan 需有缓冲（容量=区块数），否则 Range→reload 对 nil/无缓冲 channel send 会永久阻塞
func NewBigMapBornBlockManager(count int32) *BigMapBornBlockManager {
	return &BigMapBornBlockManager{
		BornCount: count,
		bornChan:  make(chan cores_declarations.BornBlockID, count),
	}
}

// BigMapBornBlockManager 大地图出生块管理器
// 维护两个状态 map：emptyBornMap（空闲出生块）和 useBornMap（已使用出生块）
// 通过 Store/Load/Use/Free 操作管理出生块的生命周期
type BigMapBornBlockManager struct {
	BornCount    int32
	bornChan     chan cores_declarations.BornBlockID
	emptyBornMap hashtriemap.HashTrieMap[cores_declarations.BornBlockID, map[int32]struct{}] // 空闲出生块集合
	useBornMap   hashtriemap.HashTrieMap[cores_declarations.BornBlockID, map[int32]struct{}] // 已使用出生块集合
	reloadLocker sync.Mutex
}

// Store 存储一个出生块数据到空闲池中
func (b *BigMapBornBlockManager) Store(bornID cores_declarations.BornBlockID, data map[int32]struct{}) bool {
	b.emptyBornMap.Store(bornID, data)
	return true
}

// Load 从空闲池中加载指定 ID 的出生块数据
func (b *BigMapBornBlockManager) Load(bornID cores_declarations.BornBlockID) (map[int32]struct{}, bool) {
	return b.emptyBornMap.Load(bornID)
}

// Use 将指定的出生块从空闲池迁移到已使用池，标记为正在使用
// 返回 false 表示该出生块不在空闲池中
func (b *BigMapBornBlockManager) Use(bornID cores_declarations.BornBlockID) bool {
	data, loadOk := b.emptyBornMap.LoadAndDelete(bornID)
	if !loadOk {
		return false
	}
	b.useBornMap.Store(bornID, data)
	return true
}

// Free 将指定的出生块从已使用池释放回空闲池，标记为可用
// 返回 false 表示该出生块不在已使用池中
func (b *BigMapBornBlockManager) Free(bornID cores_declarations.BornBlockID) bool {
	data, loadOk := b.useBornMap.LoadAndDelete(bornID)
	if !loadOk {
		return false
	}
	b.emptyBornMap.Store(bornID, data)
	return true
}

// Delete 从空闲池和已使用池中同时删除指定出生块
func (b *BigMapBornBlockManager) Delete(bornID cores_declarations.BornBlockID) {
	b.emptyBornMap.Delete(bornID)
	b.useBornMap.Delete(bornID)
}

// Range 遍历所有出生块
func (b *BigMapBornBlockManager) Range(f func(cores_declarations.BornBlockID, map[int32]struct{}) bool) {
	retry := false
reTryLoop:
	if len(b.bornChan) == 0 {
		b.reload()
	}
	for {
		select {
		case bornID := <-b.bornChan:
			data, ok := b.Load(bornID)
			if ok {
				if !f(bornID, data) {
					return
				}
			}
		default:
			if retry {
				return
			}
			// 池里没有了，就重试一次。
			retry = true
			goto reTryLoop
		}
	}
}

func (b *BigMapBornBlockManager) reload() {
	b.reloadLocker.Lock()
	defer b.reloadLocker.Unlock()
	if len(b.bornChan) != 0 {
		return
	}

	// 收集全部空闲块并洗牌：保证每次分配块序随机（此前按 1..25 顺序，宏观上不随机）
	ids := make([]cores_declarations.BornBlockID, 0, b.BornCount)
	for bornID := int32(1); bornID <= b.BornCount; bornID++ {
		var useBornID = cores_declarations.BornBlockID(bornID)
		if _, ok := b.emptyBornMap.Load(useBornID); ok {
			ids = append(ids, useBornID)
		}
	}
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	for _, id := range ids {
		b.bornChan <- id
	}
}
