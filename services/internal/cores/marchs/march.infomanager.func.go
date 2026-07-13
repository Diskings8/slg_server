package marchs

import (
	"server.slg.com/common/common_declarations"
)

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
		// 处理驻守等状态 todo

		mm.allMarch[marchInfo.MarchID] = marchInfo
		mm.TickerChan <- marchInfo
	}
	return marchList, nil
}

func (mm *MarchInfoManager) checkAutoMigrate(dbc common_declarations.DbcI) error {
	return dbc.Table(mm.GetTableName()).AutoMigrate(&MarchInfo{})
}

func (mm *MarchInfoManager) findMarchList(dbc common_declarations.DbcI, marchList []*MarchInfo) error {
	return dbc.Table(mm.GetTableName()).Find(&marchList)

}
