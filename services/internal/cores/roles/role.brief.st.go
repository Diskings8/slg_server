package roles

import (
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/common/utils/util_roles"
)

type Brief struct {
	RoleBrief *pb_role.RoleBrief
}

func (b *Brief) Clone() *Brief {
	return &Brief{RoleBrief: util_roles.CopyRoleBrief(b.RoleBrief)}
}
