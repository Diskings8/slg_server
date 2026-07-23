package map_blocks

import (
	"sync"
	"sync/atomic"

	"server.slg.com/services/internal/cores/cores_declarations"
)

// MapBlock 地图块管理器，将大地图按区块划分，提供区块粒度的空闲地块查找和元素计数功能。
//
// 设计说明：
//   - 地图被划分为 ServerMapBlockCutNum（25）个区块，呈 5x5 网格分布
//   - 每个区块独立跟踪其内部空闲地块和已占用地块
//   - 主要用于后续的怪物刷新、资源分配等需要快速查找空闲地块的场景
//   - 如需更复杂的加权分配策略，可在上层按 FreeMapLen 做比例计算
type MapBlock struct {
	blocks      []blockPart                        // 区块数组，index 从 1 开始（0 位留空）
	blockLength int32                              // 每个区块的边长（格子数）
	config      cores_declarations.MapConfigI      // 地图配置，用于坐标转换
}

// blockPart 单个区块数据
//
// 每个区块是一个独立的管理单元，维护其覆盖范围内所有地块的状态：
//   - freeData:     当前空闲（无怪物/建筑等刷新元素）的地块集合
//   - notFreeData:  已被占用的地块集合
//   - Count:        按元素类型统计的数量，用于控制各类内容的分布
type blockPart struct {
	BlockID    cores_declarations.BornBlockID      // 区块 ID（1~25）
	FreeMapLen atomic.Int32                        // 空闲地块数量，原子操作无需加锁读取

	dataLocker  sync.RWMutex                       // 保护以下字段的读写锁
	Count       map[int32]*atomic.Int32            // map[元素类型ID]数量，用于各类内容的分布控制
	freeData    map[cores_declarations.MapID]struct{}    // 空闲地块集合
	notFreeData map[cores_declarations.MapID]struct{}    // 非空闲地块集合
}
