package util_roles

import (
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_role"
)

// CopyRoleBrief 复制一份角色简略信息
func CopyRoleBrief(brief *pb_role.RoleBrief) *pb_role.RoleBrief {
	return &pb_role.RoleBrief{
		RoleBaseInfo:  CopyFromRoleBaseInfo(brief.GetRoleBaseInfo()),
		UnionBaseInfo: CopyFromUnionBaseInfo(brief.GetUnionBaseInfo()),
		LogoutAt:      brief.GetLogoutAt(),
		LoginAt:       brief.GetLoginAt(),
	}
}

// CopyFromRoleBaseInfo 复制一份角色基础信息
func CopyFromRoleBaseInfo(info *pb_role.RoleBaseInfo) *pb_role.RoleBaseInfo {
	return proto.Clone(info).(*pb_role.RoleBaseInfo)
}

// CopyFromUnionBaseInfo 复制一份角色联盟基础信息
func CopyFromUnionBaseInfo(info *pb_role.RoleUnionBaseInfo) *pb_role.RoleUnionBaseInfo {
	return proto.Clone(info).(*pb_role.RoleUnionBaseInfo)
}
