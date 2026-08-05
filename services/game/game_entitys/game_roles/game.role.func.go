package game_roles

import (
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_declarations"
)

// GetRole 获取角色 poller + 数据（写路径：持锁，改完需 poller.Save 打脏 + defer Release）。
//
//	用法:
//	  poller, role, err := game_roles.GetRole(roleID)
//	  if err != nil { return err }
//	  defer poller.Release()
func GetRole(roleID uint64) (game_declarations.PollerI[*Role], *Role, rpc_results.ResultI) {
	rolePoller, err := getPoller(roleID)
	if err != nil {
		return nil, nil, rpc_results.Error(pb_error_code.ErrorCode_SystemBusy, err.Error())
	}
	role, err := rolePoller.Get()
	if err != nil {
		rolePoller.Release()
		return nil, nil, rpc_results.Error(pb_error_code.ErrorCode_SystemBusy, err.Error())
	}
	return rolePoller, role, nil
}

// GetCopy 获取角色数据副本（只读路径：免锁 COW 快照，无需 Release）。
func GetCopy(roleID uint64) (*Role, rpc_results.ResultI) {
	rolePoller, err := getPoller(roleID)
	if err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_SystemBusy, err.Error())
	}
	return rolePoller.GetCopy(), nil
}

// Do 自动管理 poller 生命周期（GetRole → fn → defer Release）。
func Do(roleID uint64, needSave bool, fn func(role *Role) rpc_results.ResultI) rpc_results.ResultI {
	poller, role, err := GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	return fn(role)
}
