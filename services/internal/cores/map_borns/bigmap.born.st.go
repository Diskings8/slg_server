package map_borns

import (
	"sync"

	"github.com/go4org/hashtriemap"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.BornBlockI = (*BigMapBornBlockManager)(nil)

// BigMapBornBlockManager 大地图出生块管理器
// 维护两个状态 map：emptyBornMap（空闲出生块）和 useBornMap（已使用出生块）
// 通过 Store/Load/Use/Free 操作管理出生块的生命周期
type BigMapBornBlockManager struct {
	BronCount    int32
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

	for _, bornID := range blockSort {
		var useBornID = cores_declarations.BornBlockID(bornID)
		if _, ok := b.emptyBornMap.Load(useBornID); ok {
			b.bornChan <- useBornID
		}
	}
}
