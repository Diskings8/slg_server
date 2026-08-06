package map_managers

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

// TestMapManager_CreateRole 创建角色并落主城：
// CreateRole 分配出生点 + 落主城，返回主城核心 MapID，且核心格写入归属。
func TestMapManager_CreateRole(t *testing.T) {
	mdm := map_datas.NewMapDataManager(testMapConfig{}, "map_data")
	setBornSafe(mdm)

	tickerChan := make(chan *marchs.MarchInfo, 1000)
	marchMgr := marchs.New(tickerChan, "march_info", testMapConfig{}, cores_declarations.MarchTimeTypeStraight)
	mm := NewMapManager(1, cores_declarations.MapGroupBase, mdm, marchMgr, nil, nil)

	brief := &pb_role.RoleBrief{
		RoleBaseInfo: &pb_role.RoleBaseInfo{
			SimpleInfo: &pb_role.RoleSimpleInfo{
				RoleId:   10001,
				ServerId: 1,
				RoleName: "tester",
			},
		},
	}

	coreMapID, err := mm.CreateRole(brief)
	if err != nil {
		t.Fatalf("CreateRole err: %v", err)
	}
	if coreMapID <= 0 {
		t.Fatalf("coreMapID = %d, want > 0", coreMapID)
	}

	// 主城核心格应写入归属
	info, ok := mdm.GetMapInfo(coreMapID)
	if !ok {
		t.Fatalf("core map %d not found", coreMapID)
	}
	if info.GetOwnerID() != brief.GetRoleBaseInfo().GetSimpleInfo().GetRoleId() {
		t.Fatalf("ownerID = %d, want %d", info.GetOwnerID(), brief.GetRoleBaseInfo().GetSimpleInfo().GetRoleId())
	}
	if info.GetServerID() != brief.GetRoleBaseInfo().GetSimpleInfo().GetServerId() {
		t.Fatalf("serverID = %d, want %d", info.GetServerID(), brief.GetRoleBaseInfo().GetSimpleInfo().GetServerId())
	}
}
