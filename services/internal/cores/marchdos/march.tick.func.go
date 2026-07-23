package marchdos

import (
	"time"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/api/protocol/pb/pb_maps_march"
)

// DefaultMarchTickHandler marchDoFunc 的默认实现
//
// tick 触发时：
//  1. 取行军信息，不存在则忽略
//  2. 检查行军是否真的到期，未到期则重新入队
//  3. 通过工厂创建行军执行器
//  4. 锁定行军和地块（召回中的行军不上目标锁）
//  5. 执行到达处理（按状态分流）
//  6. 出错时清理行军
//
// 调用方将它与 NewMarchDo 一起注入 NewMapManager：
//
//	NewMapManager(..., marchdos.DefaultMarchTickHandler, marchdos.NewMarchDo)
func DefaultMarchTickHandler(mm *map_managers.MapManager, marchID cores_declarations.MarchID) {
	marchInfo := mm.GetMarchManage().GetMarchInfo(marchID)
	if marchInfo == nil {
		return
	}

	// 检查行军是否真的到期，未到期则重新入队等待
	_, endTime := marchInfo.GetMarchStartAndEndTimeUx()
	if endTime > time.Now().Unix() {
		mm.GetMarchManage().TickerChan <- marchInfo
		return
	}

	// 锁定 marchLocker，防止并发处理
	if !marchInfo.LockMarchDo() {
		return
	}
	defer marchInfo.UnlockMarchDo()

	handle := NewMarchDo(mm, marchInfo)
	if handle == nil {
		return
	}

	// 召回中的行军不上目标锁（目标即是出发地，已无战斗逻辑）
	toMapLock := marchInfo.GetMarchState() != pb_maps_march.MarchState_Back

	handle.Lock(true, false, toMapLock)
	err := handle.Do()
	handle.Unlock()

	if err != nil {
		// 出错时清理：从 AOI、MapAttribute 中移除，推送召回，删除行军
		_ = handle.CallBack()
	}
}
