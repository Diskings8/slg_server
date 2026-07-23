package marchs

import (
	"errors"
	"time"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// New 初始化
func New(tickerChan chan *MarchInfo, tableName string, mapConfig cores_declarations.MapConfigI, marchTimeType cores_declarations.MarchTimeType) *MarchInfoManager {
	m := &MarchInfoManager{
		TickerChan:       tickerChan,
		mapConfig:        mapConfig,
		MarchTimeType:    marchTimeType,
		allMarch:         make(map[cores_declarations.MarchID]*MarchInfo),
		allAssembleMarch: make(map[cores_declarations.MarchID][]*MarchInfo),
		mapMarch:         make([]MapAttribute, mapConfig.MapCount()),
		tableName:        tableName,
	}
	return m
}

func (mm *MarchInfoManager) Init(dbc common_declarations.DbcI) ([]*MarchInfo, error) {
	if err := mm.checkAutoMigrate(dbc); err != nil {
		return nil, err
	}
	var marchList []*MarchInfo
	if err := mm.findMarchList(dbc, marchList); err != nil {
		return nil, err
	}

	mm.allMarchLock.Lock()
	defer mm.allMarchLock.Unlock()
	for _, marchInfo := range marchList {
		// 行军挂载到地图
		mm.MapAttributeMarchCreate(marchInfo)
		// 处理驻守等状态
		if marchInfo.IsMarchTypeAssist() {
			_, toMapID, _ := marchInfo.GetMapIDs()
			if marchInfo.GetMarchState() == pb_maps_march.MarchState_Station {
				mm.MapAttributeGet(toMapID).AssistArrive(marchInfo)
			}
		}

		mm.allMarch[marchInfo.MarchID] = marchInfo
		mm.TickerChan <- marchInfo
	}
	return marchList, nil
}

func (mm *MarchInfoManager) checkAutoMigrate(dbc common_declarations.DbcI) error {
	return dbc.Table(mm.GetTableName()).AutoMigrate(&MarchInfo{})
}

func (mm *MarchInfoManager) findMarchList(dbc common_declarations.DbcI, marchList []*MarchInfo) error {
	return dbc.Table(mm.GetTableName()).Find(&marchList).Error()

}

//----------------MapAttribute 相关----------------------------//

// MapAttributeGet 取得数据
func (mm *MarchInfoManager) MapAttributeGet(mapID cores_declarations.MapID) *MapAttribute {
	if mapID < 0 || int32(mapID) >= mm.mapConfig.MapCount() {
		return nil
	}
	return &mm.mapMarch[mapID]
}

// MapAttributeMarchCreate 创建行军
func (mm *MarchInfoManager) MapAttributeMarchCreate(marchInfo *MarchInfo) {
	formMapID, toMapID, firstFormMapID := marchInfo.GetMapIDs()
	if formMapID >= 0 {
		mm.MapAttributeGet(formMapID).marchAdd(marchInfo)
	}
	if toMapID >= 0 {
		mm.MapAttributeGet(toMapID).marchAdd(marchInfo)
	}
	if firstFormMapID != formMapID && firstFormMapID >= 0 {
		mm.MapAttributeGet(firstFormMapID).marchAdd(marchInfo)
	}
}

// MapAttributeMarchDelete 删除行军
func (mm *MarchInfoManager) MapAttributeMarchDelete(marchInfo *MarchInfo) {
	if marchInfo.GetFromMapID() >= 0 {
		mm.MapAttributeGet(marchInfo.GetFromMapID()).marchDel(marchInfo.GetMarchID())
	}
	if marchInfo.GetToMapID() >= 0 {
		mm.MapAttributeGet(marchInfo.GetToMapID()).marchDel(marchInfo.GetMarchID())
	}
	if marchInfo.GetSrcFromMapID() >= 0 && marchInfo.GetSrcFromMapID() != marchInfo.GetFromMapID() {
		mm.MapAttributeGet(marchInfo.GetSrcFromMapID()).marchDel(marchInfo.GetMarchID())
	}
}

// MapAttributeMarchChange 行军新的目标位置
func (mm *MarchInfoManager) MapAttributeMarchChange(marchInfo *MarchInfo, newMapID cores_declarations.MapID) {
	// 删除旧的坐标关联的行军
	mm.MapAttributeGet(marchInfo.FromMapID).marchDel(marchInfo.MarchID)

	// 记录开始的坐标
	if marchInfo.GetSrcFromMapID() != marchInfo.FromMapID {
		marchInfo.SrcFromMapID = marchInfo.FromMapID
	}
	marchInfo.FromMapID = marchInfo.ToMapID // 开始转为当前的结束目标地址
	marchInfo.ToMapID = newMapID            // 目标地址改为新的坐标
	// 更新实际出发地（召回目标），FromMapID 此时已是旧的 ToMapID（停留点）
	marchInfo.TransitMapID = marchInfo.FromMapID
	mm.MapAttributeGet(marchInfo.ToMapID).marchAdd(marchInfo) // 重新绑定新的目标地址
}

// MapAttributeMarchModToMapID 行军修改新的目标位置
func (mm *MarchInfoManager) MapAttributeMarchModToMapID(marchInfo *MarchInfo, newToMapID cores_declarations.MapID) {
	// 删除旧的坐标关联的行军
	mm.MapAttributeGet(marchInfo.ToMapID).marchDel(marchInfo.MarchID)

	// 记录开始的坐标
	marchInfo.ToMapID = newToMapID                            // 目标地址改为新的坐标
	mm.MapAttributeGet(marchInfo.ToMapID).marchAdd(marchInfo) // 重新绑定新的目标地址
}

// MapAttributeMarchModFormMapID 行军修改新的起始目标位置，注意: marchInfo 需上层加锁
func (mm *MarchInfoManager) MapAttributeMarchModFormMapID(marchInfo *MarchInfo, newMapID cores_declarations.MapID, isAllForm bool) {
	// 删除旧的坐标关联的行军
	if marchInfo.GetSrcFromMapID() != marchInfo.FromMapID && marchInfo.GetSrcFromMapID() >= 0 {
		mm.MapAttributeGet(marchInfo.GetSrcFromMapID()).marchDel(marchInfo.MarchID)
	}
	// 开始的坐标更换
	marchInfo.SrcFromMapID = newMapID
	if isAllForm {
		marchInfo.FromMapID = newMapID
	}

	if marchInfo.GetSrcFromMapID() != marchInfo.FromMapID && marchInfo.GetSrcFromMapID() >= 0 {
		mm.MapAttributeGet(marchInfo.GetSrcFromMapID()).marchAdd(marchInfo) // 重新绑定新的目标地址
	}
}

// MapAttributeMarchCallBack 行军返回处理
//
// 从返回出发地（FromMapID）移除行军，添加到返回目标（ToMapID）。
// 注意：调用方需保证 marchInfo 数据在调用期间稳定（已加锁或仅单线程访问）。
// 内部直接访问字段而非通过 Getter（避免已持有写锁时 RLock 死锁）。
func (mm *MarchInfoManager) MapAttributeMarchCallBack(marchInfo *MarchInfo) {
	mm.MapAttributeGet(marchInfo.FromMapID).marchDel(marchInfo.MarchID)
	mm.MapAttributeGet(marchInfo.ToMapID).marchAdd(marchInfo)
}

// ---------------------------March 相关---------------------------//

func preCheckCreateMarch(marchInfo *MarchInfo) {
	if marchInfo.StartTimeUx == 0 {
		marchInfo.StartTimeUx = time.Now().Unix()
	}
	if marchInfo.EndTimeUx == 0 {
		marchInfo.EndTimeUx = time.Now().Add(time.Second * 60).Unix()
	}
	// 记录本次行军的实际出发地（用于召回时确定返回目标）
	marchInfo.TransitMapID = marchInfo.FromMapID
}

// CreateMarch 创建行军
func (mm *MarchInfoManager) CreateMarch(marchInfo *MarchInfo) error {
	if marchInfo == nil || marchInfo.GetMarchID() > 0 {
		return errors.New("参数错误")
	}

	preCheckCreateMarch(marchInfo)

	err := dbconn.GetWriteDbConn().Table(mm.tableName).Create(marchInfo).Error()
	if err != nil {
		return err
	}

	mm.addMarchInfo(marchInfo)

	mm.MapAttributeMarchCreate(marchInfo)

	return nil
}

// CreateMarchInBatches 创建行军列表
func (mm *MarchInfoManager) CreateMarchInBatches(marchInfoList ...*MarchInfo) ([]*MarchInfo, error) {
	createMarch := make([]*MarchInfo, 0, len(marchInfoList))
	for _, marchInfo := range marchInfoList {
		if marchInfo == nil || marchInfo.GetMarchID() > 0 {
			loggers.Logger.Error("CreateMarchInBatches march is empty")
			continue
		}
		preCheckCreateMarch(marchInfo)

		createMarch = append(createMarch, marchInfo)
	}

	if len(createMarch) <= 0 {
		return nil, errors.New("CreateMarchInBatches createMarch is empty")
	}

	err := dbconn.GetWriteDbConn().Table(mm.tableName).CreateInBatches(createMarch, 20).Error()
	if err != nil {
		return nil, err
	}

	for _, marchInfo := range createMarch {
		mm.addMarchInfo(marchInfo)
		mm.MapAttributeMarchCreate(marchInfo)
	}
	return createMarch, nil
}

// CreateMockMarch 创建假行军注意不会入库
func (mm *MarchInfoManager) CreateMockMarch(marchInfo *MarchInfo) error {
	if marchInfo == nil || marchInfo.MarchID < 1 {
		return errors.New("参数错误")
	}

	preCheckCreateMarch(marchInfo)

	marchInfo.isMock.Store(true)

	mm.addMarchInfo(marchInfo)

	mm.MapAttributeMarchCreate(marchInfo)

	return nil
}

// DeleteMarch 删除行军
func (mm *MarchInfoManager) DeleteMarch(marchInfo *MarchInfo) error {
	if marchInfo == nil || marchInfo.MarchID < 1 {
		return errors.New("参数错误")
	}

	// 非测试行军
	if !marchInfo.IsMock() {
		err := dbconn.GetWriteDbConn().Table(mm.tableName).Delete(marchInfo).Error()
		if err != nil {
			return err
		}
	}
	marchInfo.isNeedDelete.Store(true)

	mm.allMarchLock.Lock()
	for _, v := range marchInfo.AoiBlock {
		v.MarchDelete(marchInfo)
	}
	for _, v := range marchInfo.PassingAoiBlock {
		v.PassingMarchDelete(marchInfo)
	}
	delete(mm.allMarch, marchInfo.MarchID)
	mm.allMarchLock.Unlock()

	mm.MapAttributeMarchDelete(marchInfo)
	return nil
}

// AllMarch 全部行军
func (mm *MarchInfoManager) AllMarch() []*MarchInfo {
	var marchInfos []*MarchInfo
	mm.allMarchLock.RLock()
	defer mm.allMarchLock.RUnlock()
	for _, marchInfo := range mm.allMarch {
		marchInfos = append(marchInfos, marchInfo)
	}
	return marchInfos
}

// GetMarchInfo 单条行军查询
func (mm *MarchInfoManager) GetMarchInfo(marchID cores_declarations.MarchID) *MarchInfo {
	mm.allMarchLock.RLock()
	defer mm.allMarchLock.RUnlock()
	for _, marchInfo := range mm.allMarch {
		if marchInfo.GetMarchID() == marchID {
			return marchInfo
		}
	}
	return nil
}

func (mm *MarchInfoManager) GetMarchInfoByType(marchTypes ...cores_declarations.MarchType) []*MarchInfo {
	var result = make([]*MarchInfo, 0)
	mm.allMarchLock.RLock()
	defer mm.allMarchLock.RUnlock()
	for _, marchInfo := range mm.allMarch {
		for _, marchType := range marchTypes {
			if marchInfo.MarchType == marchType {
				result = append(result, marchInfo)
			}
		}
	}
	return result
}
