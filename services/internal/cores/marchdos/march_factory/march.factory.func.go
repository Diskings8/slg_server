package march_factory

import (
	"time"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos/assist_march"
	"server.slg.com/services/internal/cores/marchdos/attack_march"
	"server.slg.com/services/internal/cores/marchdos/develop_march"
	"server.slg.com/services/internal/cores/marchdos/strategy_march"
	"server.slg.com/services/internal/cores/marchdos/sweep_march"
	"server.slg.com/services/internal/cores/marchs"
)

// NewMarchDo 根据 MarchType 创建对应的行军执行器
func NewMarchDo(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	if marchInfo == nil || mm == nil {
		return nil
	}

	switch marchInfo.MarchType {
	case cores_declarations.MarchTypeAttack:
		return attack_march.New(mm, marchInfo)
	case cores_declarations.MarchTypeAssist:
		return assist_march.New(mm, marchInfo)
	case cores_declarations.MarchTypeSweep:
		return sweep_march.New(mm, marchInfo)
	case cores_declarations.MarchTypeStrategy:
		return strategy_march.New(mm, marchInfo)
	case cores_declarations.MarchTypeDevelop:
		return develop_march.New(mm, marchInfo)
	default:
		return nil
	}
}

// MarchTickHandler marchDoFunc 的默认实现
func MarchTickHandler(mm *map_managers.MapManager, marchID cores_declarations.MarchID) {
	marchInfo := mm.GetMarchManage().GetMarchInfo(marchID)
	if marchInfo == nil {
		return
	}

	_, endTime := marchInfo.GetMarchStartAndEndTimeUx()
	if endTime > time.Now().Unix() {
		mm.GetMarchManage().TickerChan <- marchInfo
		return
	}

	if !marchInfo.LockMarchDo() {
		return
	}
	defer marchInfo.UnlockMarchDo()

	handle := NewMarchDo(mm, marchInfo)
	if handle == nil {
		return
	}

	toMapLock := marchInfo.GetMarchState() != pb_maps_march.MarchState_Back
	handle.Lock(true, false, toMapLock)
	err := handle.Do()
	handle.Unlock()

	if err != nil {
		_ = handle.CallBack()
	}
}
