package map_blocks

import (
	"iter"
	"maps"
	"sync/atomic"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
)

// NewMapBlock 创建地图块管理器
//
// 初始化流程：
//  1. 根据地图配置计算每个区块的边长
//  2. 创建 25 个区块（ServerMapBlockCutNum），计算每个区块的区域信息
//  3. 遍历所有地图格子，将其归入对应区块的空闲集合
//
// 注意：有刷新元素（如怪物）的地块通过后续的 MapDataAdd 调用移入 notFreeData
func NewMapBlock(mapData *map_datas.MapDataManager) *MapBlock {
	config := mapData.GetConfig()
	// 每个区块的边长 = 地图每行格子数 / 行切块数
	blockLength := config.MapScope() / cores_declarations.ServerMapBlockRowCutNum

	mb := &MapBlock{
		blocks:      make([]blockPart, cores_declarations.ServerMapBlockCutNum+1), // 索引 0 留空
		blockLength: blockLength,
		config:      config,
	}

	// 初始化所有区块
	for i := int32(1); i <= cores_declarations.ServerMapBlockCutNum; i++ {
		mb.blocks[i] = blockPart{
			BlockID:     cores_declarations.BornBlockID(i),
			Count:       make(map[int32]*atomic.Int32),
			freeData:    make(map[cores_declarations.MapID]struct{}),
			notFreeData: make(map[cores_declarations.MapID]struct{}),
		}
	}

	// 遍历地图数据，将每个格子归入对应区块
	mapData.Range(func(info *map_datas.MapInfo) bool {
		blockID := mb.CalcBlock(int32(info.GetPointX()), int32(info.GetPointY()))
		block := &mb.blocks[blockID]

		block.FreeMapLen.Add(1)
		block.dataLocker.Lock()
		block.freeData[info.GetMapID()] = struct{}{}
		block.dataLocker.Unlock()
		return true
	})

	return mb
}

// FirstXY 计算指定区块的起始坐标（左上角）
// blockID 范围：1 ~ ServerMapBlockCutNum
func (mb *MapBlock) FirstXY(blockID cores_declarations.BornBlockID) (x, y int32) {
	id := int32(blockID)
	x = ((id - 1) % cores_declarations.ServerMapBlockRowCutNum) * mb.blockLength
	y = ((id - 1) / cores_declarations.ServerMapBlockRowCutNum) * mb.blockLength
	return
}

// MapIDFirst 取得区块内的第一个地图格子 ID
func (mb *MapBlock) MapIDFirst(blockID cores_declarations.BornBlockID) cores_declarations.MapID {
	x, y := mb.FirstXY(blockID)
	return mb.config.XY2MapID(x, y)
}

// CalcBlock 根据坐标计算所属区块 ID
func (mb *MapBlock) CalcBlock(x, y int32) cores_declarations.BornBlockID {
	return cores_declarations.BornBlockID(
		x/mb.blockLength + y/mb.blockLength*cores_declarations.ServerMapBlockRowCutNum + 1,
	)
}

// CalcBlockMapID 根据格子 ID 计算所属区块 ID
func (mb *MapBlock) CalcBlockMapID(mapID cores_declarations.MapID) cores_declarations.BornBlockID {
	x, y := mb.config.MapID2XY(mapID)
	return mb.CalcBlock(x, y)
}

// Get 根据区块 ID 获取区块数据
func (mb *MapBlock) Get(blockID cores_declarations.BornBlockID) *blockPart {
	id := int32(blockID)
	if id < 1 || id > cores_declarations.ServerMapBlockCutNum {
		return nil
	}
	return &mb.blocks[id]
}

// GetBlockByMapID 根据格子 ID 获取所属区块
func (mb *MapBlock) GetBlockByMapID(mapID cores_declarations.MapID) *blockPart {
	return mb.Get(mb.CalcBlockMapID(mapID))
}

// Range 遍历所有区块
func (mb *MapBlock) Range(f func(b *blockPart) bool) {
	for i := int32(1); i <= cores_declarations.ServerMapBlockCutNum; i++ {
		if !f(&mb.blocks[i]) {
			return
		}
	}
}

// MapDataAdd 标记地块被刷新元素占用
//
// 当某个地块上生成了怪物、资源点等刷新类元素时调用，
// 将该地块从空闲集合移至占用集合，并更新元素计数。
// elementID < 1 时忽略（无刷新元素的地块不处理）
func (mb *MapBlock) MapDataAdd(mapID cores_declarations.MapID, elementID int32) {
	if elementID < 1 {
		return
	}
	block := mb.GetBlockByMapID(mapID)
	if block == nil {
		return
	}

	block.dataLocker.Lock()
	defer block.dataLocker.Unlock()

	if _, ok := block.freeData[mapID]; ok {
		block.notFreeData[mapID] = struct{}{}
		delete(block.freeData, mapID)
		block.FreeMapLen.Add(-1)

		if block.Count[elementID] == nil {
			block.Count[elementID] = &atomic.Int32{}
		}
		block.Count[elementID].Add(1)
	}
}

// MapDataDel 标记地块的刷新元素被移除
//
// 当地块上的怪物/资源被采集或清除时调用，
// 将该地块从占用集合移回空闲集合。
func (mb *MapBlock) MapDataDel(mapID cores_declarations.MapID, elementID int32) {
	block := mb.GetBlockByMapID(mapID)
	if block == nil {
		return
	}

	block.dataLocker.Lock()
	defer block.dataLocker.Unlock()

	if _, ok := block.notFreeData[mapID]; ok {
		block.freeData[mapID] = struct{}{}
		delete(block.notFreeData, mapID)
		block.FreeMapLen.Add(1)

		if block.Count[elementID] == nil {
			return
		}
		block.Count[elementID].Add(-1)
	}
}

// TypeCount 获取指定元素类型在当前区块的数量
func (b *blockPart) TypeCount(elementID int32) int32 {
	b.dataLocker.RLock()
	defer b.dataLocker.RUnlock()
	if b.Count[elementID] == nil {
		return 0
	}
	return max(0, b.Count[elementID].Load())
}

// FreeRandOne 从区块中随机获取一个空闲地块 ID
//
// 注意：返回的地块 ID 可能在调用方使用时已被占用，
// 调用方自行通过 map_datas.MapDataManager.TryLock 加锁确认。
// 返回 -1 表示该区块无空闲地块。
func (b *blockPart) FreeRandOne() cores_declarations.MapID {
	b.dataLocker.RLock()
	defer b.dataLocker.RUnlock()
	for mapID := range b.freeData {
		return mapID
	}
	return -1
}

// FreeData 返回区块内所有空闲地块的迭代器
//
// 使用 Go 1.23 的 iter.Seq 迭代器模式：
//
//	for mapID := range block.FreeData() {
//	    // 处理空闲地块
//	}
func (b *blockPart) FreeData() iter.Seq[cores_declarations.MapID] {
	return func(yield func(cores_declarations.MapID) bool) {
		b.dataLocker.RLock()
		tmp := maps.Clone(b.freeData)
		b.dataLocker.RUnlock()
		for k := range tmp {
			if !yield(k) {
				return
			}
		}
	}
}

// NotFreeData 返回区块内所有已占用地块的迭代器
func (b *blockPart) NotFreeData() iter.Seq[cores_declarations.MapID] {
	return func(yield func(cores_declarations.MapID) bool) {
		b.dataLocker.RLock()
		tmp := maps.Clone(b.notFreeData)
		b.dataLocker.RUnlock()
		for k := range tmp {
			if !yield(k) {
				return
			}
		}
	}
}

// FreeCount 获取区块的空闲地块数量（原子读取，无需加锁）
func (b *blockPart) FreeCount() int32 {
	return b.FreeMapLen.Load()
}
