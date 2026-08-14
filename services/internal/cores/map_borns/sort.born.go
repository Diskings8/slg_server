package map_borns

// 出生种子位置配置
//
// 大地图 1000×1000 切成 5×5 = 25 个出生块（每块 200×200），块内按固定间距布设候选种子，
// GetFreeBorn 在种子周围展开 3×3 校验完整空地后占为己用。
//
// 主城占地 3×3（中心格 ±1）。两主城中心间距 ≥ BornSeedSpacing 时，3×3 不重叠且边缘留出缓冲，
// 满足"主城间隔 >6 格"的要求（间距 9 → 中心距 9、城缘净距 7）。
const BornSeedSpacing int32 = 9

// BornSeedOffsets 生成块内候选种子的相对偏移（距块边缘留 margin，保证 3×3 邻域完整落在块内）。
func BornSeedOffsets(blockLength int32) []int32 {
	const margin int32 = 2 // 距块边缘最小偏移：种子 ±1 的 3×3 必须完整在块内
	var offs []int32
	for o := margin; o+margin < blockLength; o += BornSeedSpacing {
		offs = append(offs, o)
	}
	return offs
}
