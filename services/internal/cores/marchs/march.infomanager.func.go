package marchs

import "server.slg.com/common/conns/dbconn/dbconn_interface"

func (mm *MarchInfoManager) Init(dbc dbconn_interface.DbcI) ([]*MarchInfo, error) {
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

func (mm *MarchInfoManager) checkAutoMigrate(dbc dbconn_interface.DbcI) error {
	return dbc.Table(mm.GetTableName()).AutoMigrate(&MarchInfo{})
}

func (mm *MarchInfoManager) findMarchList(dbc dbconn_interface.DbcI, marchList []*MarchInfo) error {
	return dbc.Table(mm.GetTableName()).Find(&marchList)

}
