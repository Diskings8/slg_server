package game_logics

import (
	"server.slg.com/api/protocol/pb/pb_attr"
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// AttrGetPb 获取角色属性
func AttrGetPb(role *game_roles.Role) *pb_attr.RoleAttr {
	return role.GetAttr().Format2Pb()
}

// FillRoleSimpleInfo 填充角色简略信息（server_id / vip_level），供 account 登录流调用
func FillRoleSimpleInfo(role *game_roles.Role, simple *pb_role.RoleSimpleInfo) {
	role.GetAttr().FillRoleSimpleInfo(simple)
}
