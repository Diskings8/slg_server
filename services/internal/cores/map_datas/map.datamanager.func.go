package map_datas

import (
	"errors"

	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/internal/cores/cores_declarations"
)

func (mdm *MapDataManager) Clear(mapIDs []cores_declarations.MapID) {
	// todo
	panic("implement me")
}

func (mdm *MapDataManager) SetRoleMainCity(roleCityState cores_declarations.RoleMainCityState, dataSlice []*MapInfo, roleBrief *pb_role.RoleBrief) error {
	var coreIndex int
	switch roleCityState {
	case cores_declarations.RoleMainCityStateNormal:
		if len(dataSlice) != cores_declarations.RoleMainCityStateNormalCoverCount {
			return errors.New("地块数量不对")
		}
		coreIndex = cores_declarations.Land1CoverBaseKey
	default:
		if len(dataSlice) != cores_declarations.RoleMainCityStatePortableCoverCount {
			return errors.New("地块数量不对")
		}
		coreIndex = cores_declarations.Land3CoverBaseKey
	}
	// 检测位置可使用情况 todo

	//
	coreMapInfo := dataSlice[coreIndex]
	for _, mapInfo := range dataSlice {
		mapInfo.Free()
		mapInfo.serverID = roleBrief.GetRoleBaseInfo().GetSimpleInfo().GetServerId()
		mapInfo.ownerID = roleBrief.GetRoleBaseInfo().GetSimpleInfo().GetRoleId()
		mapInfo.coreMapID = coreMapInfo.mapID

		// aoi 更新
	}
	mdm.Save(dataSlice...)

	// 更新角色数据 todo

	// aoi更新

	return nil
}

// GetFreeBorn 可用出生点,失败要调用freeBornFunc放回到出生块里
// 注意：取出的mapSlice是带锁的
func (mdm *MapDataManager) GetFreeBorn() (mapIDs []int32, lockMapSlice LockMapSlice, bornID int32, baseMapID int32, freeBornFunc func(), err error) {
	mdm.BornAts.Range(func(bornIDTmp cores_declarations.BornBlockID, v map[int32]struct{}) bool {
		// 随机找一个四块地都是空地
		for mapID := range v {
			mapIDsTmp := mdm.GetConfig().CoverMapIDs(mapID, 1, cores_declarations.HallLandCover/2)
			if len(mapIDsTmp) != gamemap.HallCoverCount {
				continue
			}

			mapSliceTmp := l.GetSlice(mapIDsTmp)
			if !l.TryLock(mapSliceTmp) {
				continue
			}
			if len(mapSliceTmp) != len(mapIDsTmp) {
				l.UnLock(mapSliceTmp)
				return true
			}

			if areaID > 0 {
				for _, v := range mapSliceTmp {
					if v.AreaLevel != areaID {
						l.UnLock(mapSliceTmp)
						return true
					}
				}
			}

			if CheckCreateRoleSiteByMaps(false, mapSliceTmp...) {
				mapIDs = mapIDsTmp
				bornID = bornIDTmp
				lockMapSlice = LockMapSlice{
					data: mapSliceTmp,
					m:    l,
				}
				baseMapID = mapIDsTmp[gamemap.Land3CoverBaseKey]

				l.BornAts.Use(bornID)

				freeBornFunc = func() {
					l.BornAts.Free(bornID)
				}

				return false
			}
			l.UnLock(mapSliceTmp)
		}

		if config.Get().IsDevelop() {
			// 这个地块一个9块空地都找不到，直接将这块地置为不可创建，打印出错误提示
			logger.Get().Warn("在一个可放置玩家的位置，找不到一个1格相连的空地，打印出错误信息", zap.Int32("bornIDTmp", bornIDTmp), zap.Any("v", v))
		}

		l.BornAts.Delete(bornIDTmp)

		return true
	})

	if bornID < 1 {
		err = errors.New("没有空佘的位置可创号")
	}
	return mapIDs, lockMapSlice, bornID, baseMapID, freeBornFunc, err
}
