package map_borns

import (
	"github.com/go4org/hashtriemap"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.BornBlockI = (*TempBornBlockManager)(nil)

type TempBornBlockManager struct {
	BronCount    int32
	emptyBornMap hashtriemap.HashTrieMap[cores_declarations.BornBlockID, map[int32]struct{}] // 空闲出生块集合
	useBornMap   hashtriemap.HashTrieMap[cores_declarations.BornBlockID, map[int32]struct{}] // 已使用出生块集合
}

func (t *TempBornBlockManager) Store(bornID cores_declarations.BornBlockID, data map[int32]struct{}) bool {
	t.useBornMap.Store(bornID, data)
	return true
}

func (t *TempBornBlockManager) Load(_ cores_declarations.BornBlockID) (map[int32]struct{}, bool) {
	return nil, false
}

func (t *TempBornBlockManager) Use(bornID cores_declarations.BornBlockID) bool {
	data, loadOk := t.emptyBornMap.LoadAndDelete(bornID)
	if !loadOk {
		return false
	}
	t.useBornMap.Store(bornID, data)
	return true
}

func (t *TempBornBlockManager) Free(bornID cores_declarations.BornBlockID) bool {
	data, loaded := t.useBornMap.LoadAndDelete(bornID)
	if !loaded {
		return false
	}
	t.emptyBornMap.Store(bornID, data)
	return true
}

func (t *TempBornBlockManager) Delete(bornID cores_declarations.BornBlockID) {
	t.emptyBornMap.Delete(bornID)
	t.useBornMap.Delete(bornID)
}

func (t *TempBornBlockManager) Range(f func(cores_declarations.BornBlockID, map[int32]struct{}) bool) {
	t.emptyBornMap.Range(func(key cores_declarations.BornBlockID, _ map[int32]struct{}) bool {
		mapID := key
		return f(mapID, map[int32]struct{}{int32(mapID): {}})
	})
}
