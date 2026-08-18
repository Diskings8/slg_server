package game_logics

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_union"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_unions"
	"server.slg.com/services/game/game_models"
)

// fakeUnionRepo 内存联盟持久化（无 DB，单测用）
type fakeUnionRepo struct {
	unions map[uint64]*game_models.Union
	nextID uint64
}

func (f *fakeUnionRepo) Get(unionID uint64) (*game_models.Union, error) { return f.unions[unionID], nil }
func (f *fakeUnionRepo) Create(u *game_models.Union) (uint64, error) {
	f.nextID++
	u.ID = f.nextID
	f.unions[u.ID] = u
	return u.ID, nil
}
func (f *fakeUnionRepo) Save(u *game_models.Union) error { f.unions[u.ID] = u; return nil }
func (f *fakeUnionRepo) Delete(unionID uint64) error     { delete(f.unions, unionID); return nil }
func (f *fakeUnionRepo) ExistsName(name string) (bool, error) {
	for _, u := range f.unions {
		if u.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// TestUnionCreateJoinLeave 建盟→加入→成员列表/信息→退出 全流程（自操作，不依赖 DB）
func TestUnionCreateJoinLeave(t *testing.T) {
	game_unions.SetRepo(&fakeUnionRepo{unions: map[uint64]*game_models.Union{}})

	leader := game_roles.NewTest(90001)
	member := game_roles.NewTest(90002)

	// 建盟
	ru, err := UnionCreate(leader, "测试联盟")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if leader.GetRoleUnion().UnionID() == 0 {
		t.Fatal("leader should be in union")
	}
	if leader.GetRoleUnion().Position() != int32(pb_union.UnionPosition_UNION_POSITION_LEADER) {
		t.Errorf("leader position = %d, want leader", leader.GetRoleUnion().Position())
	}
	unionID := ru.GetBrief().GetUnionId()
	if unionID == 0 {
		t.Fatal("union id should not be 0")
	}

	// 重复建盟 → 已加入
	if _, err := UnionCreate(leader, "另一联盟"); err == nil {
		t.Fatal("want error for already in union")
	}

	// 成员加入
	if _, err := UnionJoin(member, unionID); err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if member.GetRoleUnion().Position() != int32(pb_union.UnionPosition_UNION_POSITION_MEMBER) {
		t.Errorf("member position = %d, want member", member.GetRoleUnion().Position())
	}

	// 成员列表
	members, err := UnionMemberList(member)
	if err != nil {
		t.Fatalf("member list failed: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("member count = %d, want 2", len(members))
	}

	// 信息查询（member_count 动态）
	info, err := UnionInfo(unionID)
	if err != nil {
		t.Fatalf("info failed: %v", err)
	}
	if info.GetBrief().GetMemberCount() != 2 {
		t.Errorf("member count = %d, want 2", info.GetBrief().GetMemberCount())
	}

	// 成员退出
	if err := UnionLeave(member); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	if member.GetRoleUnion().IsInUnion() {
		t.Fatal("member should be out of union")
	}
	info, _ = UnionInfo(unionID)
	if info.GetBrief().GetMemberCount() != 1 {
		t.Errorf("after leave member count = %d, want 1", info.GetBrief().GetMemberCount())
	}
}

// TestUnionCreate_NameInvalid 非法名/重名
func TestUnionCreate_NameInvalid(t *testing.T) {
	game_unions.SetRepo(&fakeUnionRepo{unions: map[uint64]*game_models.Union{}})

	role := game_roles.NewTest(90003)

	if _, err := UnionCreate(role, "  "); err == nil {
		t.Fatal("want error for blank name")
	}
	if _, err := UnionCreate(role, "这个联盟名字实在是太长太长太长太长太长太长太长太长了"); err == nil {
		t.Fatal("want error for over-length name")
	}

	// 正常建盟后，重名被拒
	if _, err := UnionCreate(role, "联盟A"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	role2 := game_roles.NewTest(90004)
	if _, err := UnionCreate(role2, "联盟A"); err == nil {
		t.Fatal("want error for duplicate name")
	}
}
