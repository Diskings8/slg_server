package marchs

import (
	"sync/atomic"

	"go.uber.org/zap"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/asyncsave_entity"
)

func (mm *MarchInfoManager) IsDelete() bool {
	return false
}

func (mm *MarchInfoManager) Tag() string {
	return "MarchInfoManager"
}

func (mm *MarchInfoManager) Saving() *atomic.Bool {
	return &mm.saving
}

func (mm *MarchInfoManager) SaveDo() {
	mm.allMarchLock.Lock()
	tmp := make([]*MarchInfo, 0, len(mm.allMarch))
	for _, v := range mm.allMarch {
		tmp = append(tmp, v)
	}
	mm.allMarchLock.Unlock()

	num := 0
	waitSlice := make([]*MarchInfo, 0)
	for _, v := range tmp {
		if v.isNeedSave.Load() && !v.isNeedDelete.Load() {
			if v.TryLock() {
				v.isNeedSave.Store(false)

				waitSlice = append(waitSlice, v)
				num++
				if num >= dbconn.MaxSaveLen {
					mm.save(waitSlice)
					num = 0
					waitSlice = waitSlice[:0]
				}
			}
		}
	}
	if len(waitSlice) > 0 {
		mm.save(waitSlice)
	}

	// 处理异步删除队列
	mm.processDeleteQueue()
}

func (mm *MarchInfoManager) save(waitSlice []*MarchInfo) {
	err := dbconn.GetWriteDbConn().Table(mm.tableName).Save(waitSlice).Error()
	if err != nil {
		loggers.Logger.Error(err.Error())
	}
	for _, v := range waitSlice {
		if err != nil {
			v.isNeedSave.Store(true)
		}
		v.Unlock()
	}
}

// processDeleteQueue 处理异步删除队列
//
// 从 deleteQueue 中取出所有待删除行军 ID，批量执行 db.Delete。
func (mm *MarchInfoManager) processDeleteQueue() {
	mm.deleteQueueLock.Lock()
	toDelete := mm.deleteQueue
	mm.deleteQueue = nil
	mm.deleteQueueLock.Unlock()

	if len(toDelete) == 0 {
		return
	}

	for _, marchID := range toDelete {
		err := dbconn.GetWriteDbConn().Table(mm.tableName).
			Delete(&MarchInfo{MarchID: marchID}).Error()
		if err != nil {
			loggers.Logger.Error("processDeleteQueue delete march failed",
				zap.Uint64("marchID", marchID.Uint64()),
				zap.Error(err),
			)
		}
	}
}

// Save 保存数据
func (mm *MarchInfoManager) Save(i *MarchInfo) {
	if i.IsMock() {
		return
	}
	i.isNeedSave.Store(true)
	asyncsave_entity.EntitySaveFunc(mm)
}
