package map_datas

import (
	"math"
	"sort"
	"testing"

	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_borns"
)

// testMapConfig 测试用地图配置：1000×1000，坐标换算与 worldmap DefaultMapConfig 一致。
// 测试位于 cores 包内，不能引用 worldmap（反向依赖），故本地实现 MapConfigI。
type testMapConfig struct{}

func (testMapConfig) MapCount() int32 { return 1000 * 1000 }
func (testMapConfig) MapScope() int32 { return 1000 }
func (testMapConfig) MapID2XY(id cores_declarations.MapID) (x, y int32) {
	return id.Int32() % 1000, id.Int32() / 1000
}
func (testMapConfig) XY2MapID(x, y int32) cores_declarations.MapID {
	return cores_declarations.MapID(y*1000 + x)
}
func (testMapConfig) SortByDis(mapID cores_declarations.MapID, mapIDs []cores_declarations.MapID) {
	cx, cy := testMapConfig{}.MapID2XY(mapID)
	sort.Slice(mapIDs, func(i, j int) bool {
		ix, iy := testMapConfig{}.MapID2XY(mapIDs[i])
		jx, jy := testMapConfig{}.MapID2XY(mapIDs[j])
		di := (ix-cx)*(ix-cx) + (iy-cy)*(iy-cy)
		dj := (jx-cx)*(jx-cx) + (jy-cy)*(jy-cy)
		return di < dj
	})
}
func (testMapConfig) CoverMapIDs(id int32, _ int, i2 any) []cores_declarations.MapID {
	var r int32
	switch v := i2.(type) {
	case uint32:
		r = int32(v)
	case int:
		r = int32(v)
	default:
		r = 0
	}
	x := id % 1000
	y := id / 1000
	minX := int32(math.Max(float64(x-r), 0))
	maxX := int32(math.Min(float64(x+r), 999))
	minY := int32(math.Max(float64(y-r), 0))
	maxY := int32(math.Min(float64(y+r), 999))
	var ids []cores_declarations.MapID
	for ty := minY; ty <= maxY; ty++ {
		for tx := minX; tx <= maxX; tx++ {
			ids = append(ids, cores_declarations.MapID(ty*1000+tx))
		}
	}
	return ids
}

// addBornBlock 为指定出生块 Store 一批可出生种子，并把每个种子周围 3×3 设为可出生地形。
func addBornBlock(mdm *MapDataManager, bornMgr *map_borns.BigMapBornBlockManager, blockID cores_declarations.BornBlockID, seedXYs [][2]int32) {
	seeds := make(map[int32]struct{})
	for _, xy := range seedXYs {
		seed := mdm.GetConfig().XY2MapID(xy[0], xy[1])
		for _, mid := range mdm.GetConfig().CoverMapIDs(int32(seed), 1, 1) {
			if info, ok := mdm.GetMapInfo(mid); ok {
				info.SetElement(cores_declarations.ElementType_Terrain_1, 0, 0)
			}
		}
		seeds[int32(seed)] = struct{}{}
	}
	bornMgr.Store(blockID, seeds)
}

// newTestMDM 构建测试地图：区块 1/2 各挂可出生种子（后续出生点分配从区块 1 开始）。
func newTestMDM(t *testing.T) *MapDataManager {
	t.Helper()
	mdm := NewMapDataManager(testMapConfig{}, "map_data")
	bornMgr := map_borns.NewBigMapBornBlockManager(cores_declarations.ServerMapBlockCutNum)
	addBornBlock(mdm, bornMgr, 1, [][2]int32{{100, 100}, {150, 100}, {100, 150}, {150, 150}})
	addBornBlock(mdm, bornMgr, 2, [][2]int32{{250, 100}, {300, 100}, {250, 150}, {300, 150}})
	mdm.BornBlockManager = bornMgr
	return mdm
}

// TestGetFreeBorn_AllocatesNineCells 出生点分配：拿到完整 9 格、核心格在 3×3 中心、全可出生，
// 且 Use 生效后二次调用回落下一区块。
func TestGetFreeBorn_AllocatesNineCells(t *testing.T) {
	mdm := newTestMDM(t)

	mapIDs, ls, bornID, coreMapID, _, err := mdm.GetFreeBorn()
	if err != nil {
		t.Fatalf("GetFreeBorn err: %v", err)
	}
	if len(mapIDs) != cores_declarations.HallCoverCount {
		t.Fatalf("got %d cells, want %d", len(mapIDs), cores_declarations.HallCoverCount)
	}
	if bornID != 1 {
		t.Fatalf("bornID = %d, want 1", bornID)
	}
	if coreMapID != mapIDs[cores_declarations.Land3CoverBaseKey] {
		t.Fatalf("coreMapID = %d, want mapIDs[%d]=%d",
			coreMapID, cores_declarations.Land3CoverBaseKey, mapIDs[cores_declarations.Land3CoverBaseKey])
	}
	// 先释放 GetFreeBorn 分配的写锁，再读取元素（已持写锁时 RLock 会重入死锁）
	ls.Unlock()
	for _, id := range mapIDs {
		info, ok := mdm.GetMapInfo(id)
		if !ok {
			t.Fatalf("cell %d missing", id)
		}
		if info.GetElementType().IsCantBornUse() {
			t.Fatalf("cell %d not born-safe, element=%d", id, info.GetElementType())
		}
	}

	// 区块 1 已 Use，二次调用应回落区块 2
	_, ls2, bornID2, _, _, err := mdm.GetFreeBorn()
	if err != nil {
		t.Fatalf("second GetFreeBorn err: %v", err)
	}
	if bornID2 != 2 {
		t.Fatalf("second bornID = %d, want 2", bornID2)
	}
	ls2.Unlock()
}

// TestSetRoleMainCity_WritesCellAttrs 放置主城：9 格 ownerID/serverID/baseMapID 全部正确写入。
func TestSetRoleMainCity_WritesCellAttrs(t *testing.T) {
	mdm := newTestMDM(t)

	mapIDs, lockMapSlice, _, coreMapID, _, err := mdm.GetFreeBorn()
	if err != nil {
		t.Fatalf("GetFreeBorn err: %v", err)
	}

	brief := &pb_role.RoleBrief{
		RoleBaseInfo: &pb_role.RoleBaseInfo{
			SimpleInfo: &pb_role.RoleSimpleInfo{ServerId: 1, RoleId: 1001},
		},
	}

	// 注入 no-op，避免依赖 Redis/DB 的角色 poller
	orig := roleBriefSaveFunc
	roleBriefSaveFunc = func(*pb_role.RoleBrief) {}
	defer func() { roleBriefSaveFunc = orig }()

	err = mdm.SetRoleMainCity(cores_declarations.RoleMainCityStateNormal, lockMapSlice.Data(), brief)
	// 无论成功失败都先释放写锁，再读取验证（已持写锁时 RLock 会重入死锁）
	lockMapSlice.Unlock()
	if err != nil {
		t.Fatalf("SetRoleMainCity err: %v", err)
	}

	for _, id := range mapIDs {
		info, ok := mdm.GetMapInfo(id)
		if !ok {
			t.Fatalf("cell %d missing", id)
		}
		if got := info.GetOwnerID(); got != 1001 {
			t.Fatalf("cell %d owner = %d, want 1001", id, got)
		}
		if got := info.GetServerID(); got != 1 {
			t.Fatalf("cell %d server = %d, want 1", id, got)
		}
		if got := info.GetBaseMapID(); got != coreMapID {
			t.Fatalf("cell %d base = %d, want core %d", id, got, coreMapID)
		}
	}
}

// TestSetRoleMainCity_RejectsWrongCount 格子数不对应返回 error（校验分支）。
func TestSetRoleMainCity_RejectsWrongCount(t *testing.T) {
	mdm := newTestMDM(t)

	info, _ := mdm.GetMapInfo(testMapConfig{}.XY2MapID(100, 100))
	brief := &pb_role.RoleBrief{
		RoleBaseInfo: &pb_role.RoleBaseInfo{
			SimpleInfo: &pb_role.RoleSimpleInfo{ServerId: 1, RoleId: 1001},
		},
	}
	if err := mdm.SetRoleMainCity(cores_declarations.RoleMainCityStateNormal, []*MapInfo{info}, brief); err == nil {
		t.Fatal("want error for wrong cell count")
	}
}
