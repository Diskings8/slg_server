package game_logics

import (
	"testing"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/models"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
	"server.slg.com/services/game/game_models"
)

// newStarRole 构造测试角色：一个目标英雄 + 一张同配置英雄卡
func newStarRole(t *testing.T, targetStar, cardCount int32) (*game_roles.Role, *role_heroes.RoleHero) {
	t.Helper()
	role := game_roles.NewTest(30001)

	heroModel := &game_models.RoleHero{
		ModelBase: models.ModelBase{ID: 2001},
		RoleID:    role.ID,
		HeroConfID: 100,
		StarStage:  targetStar,
	}
	role.GetHeroes().List = append(role.GetHeroes().List, heroModel)

	// 同配置消耗卡
	for i := int32(0); i < cardCount; i++ {
		card := &game_models.RoleHero{
			ModelBase:  models.ModelBase{ID: uint64(3001 + i)},
			RoleID:     role.ID,
			HeroConfID: 100,
		}
		role.GetHeroes().List = append(role.GetHeroes().List, card)
	}

	return role, role_heroes.NewRoleHero(heroModel)
}

// TestHeroUpgradeStar_Success 升星成功：星级+1、消耗卡移除、养成消耗记录
func TestHeroUpgradeStar_Success(t *testing.T) {
	role, hero := newStarRole(t, 0, 1)

	if err := HeroUpgradeStar(role, hero); err != nil {
		t.Fatalf("upgrade star failed: %v", err)
	}
	if hero.GetStarStage() != 1 {
		t.Errorf("star_stage = %d, want 1", hero.GetStarStage())
	}
	// 升星发放自由属性点（星级不直接乘属性）
	if got := hero.GetAttrPoint(); got != game_conf.Load().Hero.StarPointPer {
		t.Errorf("attr_point = %d, want %d (StarPointPer)", got, game_conf.Load().Hero.StarPointPer)
	}
	// 消耗卡已从内存移除（只剩目标英雄）
	if len(role.GetHeroes().List) != 1 {
		t.Errorf("hero count = %d, want 1 (consume card removed)", len(role.GetHeroes().List))
	}
	// 养成消耗记录：CultivateStar，含被消耗英雄卡配置
	costs := role.GetCultivateCosts().List
	if len(costs) != 1 {
		t.Fatalf("cultivate cost count = %d, want 1", len(costs))
	}
	if costs[0].CultivateType != pb_cultivate.CultivateType_CultivateStar {
		t.Errorf("cultivate type = %v, want CultivateStar", costs[0].CultivateType)
	}
	if len(costs[0].Cost) != 1 || costs[0].Cost[0].GetKey() != 100 || costs[0].Cost[0].GetVal() != 1 {
		t.Errorf("cultivate cost = %+v, want [{key:100 val:1}]", costs[0].Cost)
	}
}

// TestHeroUpgradeStar_MaxStar 满星不可升
func TestHeroUpgradeStar_MaxStar(t *testing.T) {
	role, hero := newStarRole(t, 5, 1) // 已满星（上限 5）
	if err := HeroUpgradeStar(role, hero); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroStarFull) {
		t.Fatalf("err = %v, want ErrHeroStarFull", err)
	}
}

// TestHeroUpgradeStar_NoConsumeCard 无同配置卡不可升
func TestHeroUpgradeStar_NoConsumeCard(t *testing.T) {
	role, hero := newStarRole(t, 0, 0) // 无消耗卡
	if err := HeroUpgradeStar(role, hero); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroNoConsumeCard) {
		t.Fatalf("err = %v, want ErrHeroNoConsumeCard", err)
	}
}

// TestHeroUpgradeStar_ConsumeCardInFormation 消耗卡被编队引用 → 不可消耗
func TestHeroUpgradeStar_ConsumeCardInFormation(t *testing.T) {
	role, hero := newStarRole(t, 0, 1)
	role.GetFormations().List = append(role.GetFormations().List, &game_models.RoleFormation{
		ModelBase: models.ModelBase{ID: 9001},
		RoleID:    role.ID,
		HeroSlots: []*pb_maps_march.HeroSlot{{HeroId: 3001}},
	})
	if err := HeroUpgradeStar(role, hero); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroNoConsumeCard) {
		t.Fatalf("err = %v, want ErrHeroNoConsumeCard (card in formation)", err)
	}
	if hero.GetStarStage() != 0 {
		t.Errorf("star_stage = %d, want 0 (should not consume)", hero.GetStarStage())
	}
}

// TestHeroUpgradeStar_ConsumeCardHasSkill 消耗卡有技能装配 → 不可消耗
func TestHeroUpgradeStar_ConsumeCardHasSkill(t *testing.T) {
	role, hero := newStarRole(t, 0, 1)
	role.GetHeroes().List[1].EquipSkills = []*pb_skill.Skill{{ConfigId: 101, SlotId: 1}}
	if err := HeroUpgradeStar(role, hero); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroNoConsumeCard) {
		t.Fatalf("err = %v, want ErrHeroNoConsumeCard (card has skill)", err)
	}
}

// TestHeroUpgradeStar_ConsumeCardInvested 消耗卡有养成投入（升过星）→ 不可消耗
func TestHeroUpgradeStar_ConsumeCardInvested(t *testing.T) {
	role, hero := newStarRole(t, 0, 1)
	role.GetHeroes().List[1].StarStage = 1 // 消耗卡已升过星
	if err := HeroUpgradeStar(role, hero); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroNoConsumeCard) {
		t.Fatalf("err = %v, want ErrHeroNoConsumeCard (card invested)", err)
	}
}

// TestHeroUpgradeStar_SkipsUnavailableCard 多张卡时跳过不可消耗的，用可消耗的
func TestHeroUpgradeStar_SkipsUnavailableCard(t *testing.T) {
	role, hero := newStarRole(t, 0, 2) // 两张消耗卡 3001/3002
	// 3001 被编队引用（不可消耗），3002 可消耗
	role.GetFormations().List = append(role.GetFormations().List, &game_models.RoleFormation{
		ModelBase: models.ModelBase{ID: 9001},
		RoleID:    role.ID,
		HeroSlots: []*pb_maps_march.HeroSlot{{HeroId: 3001}},
	})
	if err := HeroUpgradeStar(role, hero); err != nil {
		t.Fatalf("upgrade star failed: %v", err)
	}
	if hero.GetStarStage() != 1 {
		t.Errorf("star_stage = %d, want 1", hero.GetStarStage())
	}
	// 3001（被引用）保留，3002（可消耗）被移除 → List = hero + 3001
	if len(role.GetHeroes().List) != 2 {
		t.Errorf("hero count = %d, want 2 (3002 consumed, 3001 kept)", len(role.GetHeroes().List))
	}
}
