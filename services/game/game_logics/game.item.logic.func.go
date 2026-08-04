package game_logics

import (
	"time"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// ------------------------------- 道具统一变更 -------------------------------

// ItemChange 统一道具变更入口（所有道具增删统一走这里）
//
//	自动记录产销日志，外部调用方务必传入准确的原因
func ItemChange(role *game_roles.Role, addItems, useItems []common_declarations.ItemUse, reason common_declarations.ItemChangeReason) error {
	if len(addItems) == 0 && len(useItems) == 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "空行为")
	}
	optID := snowflakes.GenStringID()
	optTimeUx := time.Now().Unix()
	var vetLogs []*pb_item.ItemChangeLog
	if len(useItems) > 0 {
		// 检测是否道具够
		if errCode := role.CheckItemEnough(useItems); errCode != pb_error_code.ErrorCode_NoneErr {
			return rpc_results.Error(errCode, "扣除道具不足")
		}
		vetLogs = append(vetLogs, role.ReduceItem(useItems, optID, reason.ToString(), optTimeUx)...)
	}
	vetLogs = append(vetLogs, role.AddItem(addItems, optID, reason.ToString(), optTimeUx)...)
	return nil
}
