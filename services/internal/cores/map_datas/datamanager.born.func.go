package map_datas

import "server.slg.com/services/internal/cores/cores_declarations"

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

		var mapElementType cores_declarations.ElementType
		if needLock {
			mapElementType = mapInfo.GetElementType()
		} else {
			mapElementType = mapInfo.ElementType
		}
		// 检测地形是否不可诞生
		if mapElementType.IsCantBornUse() {
			return false
		}
	}
	return true
}
