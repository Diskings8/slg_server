package game_role_handler

import (
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// Poller 角色 poller 接口 — 测试时可注入 mock
type Poller interface {
	Release()
	GetCopy() *game_roles.Role
	Get() (*game_roles.Role, error)
	Save() // 打脏标记，异步保存器持久化
}

// GetRole 获取角色 poller + 数据
//
//	用法:
//	  poller, role, err := game_role_handler.GetRole(roleID)
//	  if err != nil { return err }
//	  defer poller.Release()
func GetRole(roleID uint64) (Poller, *game_roles.Role, rpc_results.ResultI) {
	rolePoller, err := game_roles.GetPoller(roleID)
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

// Do 自动管理 poller 生命周期，执行操作后自动释放
//
//	用法:
//	  err := game_role_handler.Do(roleID, true, func(role *game_roles.Role) error {
//	      return doSomething(role)
//	  })
func Do(roleID uint64, needSave bool, fn func(role *game_roles.Role) rpc_results.ResultI) rpc_results.ResultI {
	poller, role, err := GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	return fn(role)
}

// GetCopy 获取角色数据副本（只读，无需 Release）
func GetCopy(roleID uint64) (*game_roles.Role, rpc_results.ResultI) {
	rolePoller, err := game_roles.GetPoller(roleID)
	if err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_SystemBusy, err.Error())
	}
	return rolePoller.GetCopy(), nil
}
