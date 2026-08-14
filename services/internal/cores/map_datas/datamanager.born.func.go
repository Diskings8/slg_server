package map_datas

import "server.slg.com/services/internal/cores/cores_declarations"

// CanBeOccupiedByMainCity 该格是否可被主城占用：
//   - 可诞生地形（Terrain_1/2/3）→ 允许
//   - 资源格且初始等级 ≤ MaxBornResourceLevel 且未开发（isDeveloped=false）→ 允许（主城占用低等级资源）
//   - 其他 → 不允许
//
// needLock=false 时直接读字段（调用方已持该格写锁，如 GetFreeBorn 的 TryLock）；
// needLock=true 时走 getter（RLock）。
func CanBeOccupiedByMainCity(info *MapInfo, needLock bool) bool {
	if info == nil {
		return false
	}
	var et cores_declarations.ElementType
	if needLock {
		et = info.GetElementType()
	} else {
		et = info.ElementType
	}
	if !et.IsCantBornUse() {
		return true
	}
	if !et.IsResource() {
		return false
	}
	var lv cores_declarations.MapLevel
	var developed bool
	if needLock {
		lv = info.GetLevel()
		developed = info.GetIsDeveloped()
	} else {
		lv = info.Level
		developed = info.isDeveloped
	}
	return lv <= cores_declarations.MaxBornResourceLevel && !developed
}

func CheckRoleBornSiteSafeByMapInfos(needLock bool, mapInfos ...*MapInfo) bool {
	return checkRoleBornSiteSafeByMapInfos(needLock, nil, mapInfos...)
}

func checkRoleBornSiteSafeByMapInfos(needLock bool, checkFunc func(info *MapInfo) bool, mapList ...*MapInfo) bool {
	for _, mapInfo := range mapList {
		// 先检测函数
		if checkFunc != nil {
			if !checkFunc(mapInfo) {
				return false
			}
		}

		// 检测该格是否可被主城占用（可诞生地形，或初始等级≤5且未开发的资源格）
		if !CanBeOccupiedByMainCity(mapInfo, needLock) {
			return false
		}
	}
	return true
}
