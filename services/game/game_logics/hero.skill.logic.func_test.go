package game_logics

import (
	"os"
	"testing"
	"time"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/common/loggers"
	"server.slg.com/common/models"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
	"server.slg.com/services/game/game_models"
)

// TestMain 初始化雪花 ID 与 Go 内嵌配置（AddSkill/AddItem 生成主键；空配置建节点 0）
func TestMain(m *testing.M) {
	loggers.Init()
	snowflakes.Init()
	_ = game_conf.InitDefault()
	os.Exit(m.Run())
}

// newSkillRole 构造测试角色：造一个指定等级/觉醒状态的英雄，返回 role + 包装后的 hero
func newSkillRole(t *testing.T, heroLevel uint32, awakened bool) (*game_roles.Role, *role_heroes.RoleHero) {
	t.Helper()
	role := game_roles.NewTest(12345)
	heroModel := &game_models.RoleHero{
		ModelBase:  models.ModelBase{ID: 1001},
		RoleID:     role.ID,
		Level:      heroLevel,
		IsAwakened: awakened,
	}
	role.GetHeroes().List = append(role.GetHeroes().List, heroModel)
	return role, role_heroes.NewRoleHero(heroModel)
}

// addItems 给角色造道具
func addItems(role *game_roles.Role, itemID pb_confs.ItemID, count int64) {
	role.GetItems().AddItem(common_declarations.ItemUse{ItemID: itemID, Count: count}, time.Now().Unix())
}

// itemCount 读取道具数量
func itemCount(role *game_roles.Role, itemID pb_confs.ItemID) int64 {
	return role.GetItems().GetItemCount(int32(itemID))
}

// ────────────────────────── 槽位可用性 ──────────────────────────

func TestHeroSkillSlotUnlocked(t *testing.T) {
	cases := []struct {
		name     string
		level    uint32
		awakened bool
		slot     int32
		want     bool
	}{
		{name: "index0默认槽始终可用", level: 1, slot: 0, want: true},
		{name: "槽1等级不足", level: 9, slot: 1, want: false},
		{name: "槽1等级达标", level: 10, slot: 1, want: true},
		{name: "槽2等级不足", level: 19, awakened: true, slot: 2, want: false},
		{name: "槽2等级达标未觉醒", level: 20, awakened: false, slot: 2, want: false},
		{name: "槽2等级达标已觉醒", level: 20, awakened: true, slot: 2, want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, hero := newSkillRole(t, c.level, c.awakened)
			if got := heroSkillSlotUnlocked(hero, c.slot); got != c.want {
				t.Errorf("slotUnlocked(level=%d, awakened=%v, slot=%d) = %v, want %v", c.level, c.awakened, c.slot, got, c.want)
			}
		})
	}
}

// ────────────────────────── 装配技能 ──────────────────────────

func TestHeroEquipSkill_Success(t *testing.T) {
	role, hero := newSkillRole(t, 15, false) // 槽1解锁
	hs := role.GetSkills().AddSkill(101, 3)  // 技能101 装配次数上限3

	skills, err := HeroEquipSkill(role, hero, 1, 101)
	if err != nil {
		t.Fatalf("equip failed: %v", err)
	}
	if len(skills) != 1 || skills[0].GetSlotId() != 1 || skills[0].GetConfigId() != 101 {
		t.Fatalf("equipped skills = %+v, want slot1 skill101", skills)
	}
	if hs.GetEquipHeroID() != hero.GetID() {
		t.Errorf("equip_hero_id = %d, want %d", hs.GetEquipHeroID(), hero.GetID())
	}
	if hs.GetUsedCount() != 1 {
		t.Errorf("used_count = %d, want 1", hs.GetUsedCount())
	}
}

func TestHeroEquipSkill_SlotLocked(t *testing.T) {
	role, hero := newSkillRole(t, 5, false) // 槽1未解锁（需10级）
	role.GetSkills().AddSkill(101, 3)

	if _, err := HeroEquipSkill(role, hero, 1, 101); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillSlotLocked) {
		t.Fatalf("err = %v, want ErrSkillSlotLocked", err)
	}
}

func TestHeroEquipSkill_Slot2NeedsAwaken(t *testing.T) {
	// 槽2：等级达标但未觉醒 → 锁定
	role, hero := newSkillRole(t, 20, false)
	role.GetSkills().AddSkill(101, 3)
	if _, err := HeroEquipSkill(role, hero, 2, 101); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillSlotLocked) {
		t.Fatalf("err = %v, want ErrSkillSlotLocked (awaken required)", err)
	}

	// 觉醒后 → 成功
	role2, hero2 := newSkillRole(t, 20, true)
	role2.GetSkills().AddSkill(101, 3)
	if _, err := HeroEquipSkill(role2, hero2, 2, 101); err != nil {
		t.Fatalf("equip slot2 after awaken failed: %v", err)
	}
}

func TestHeroEquipSkill_SlotOccupied(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	role.GetSkills().AddSkill(101, 3)
	if _, err := HeroEquipSkill(role, hero, 1, 101); err != nil {
		t.Fatalf("first equip failed: %v", err)
	}
	if _, err := HeroEquipSkill(role, hero, 1, 102); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillSlotOccupied) {
		t.Fatalf("err = %v, want ErrSkillSlotOccupied", err)
	}
}

func TestHeroEquipSkill_NotOwned(t *testing.T) {
	role, hero := newSkillRole(t, 15, false) // 技能库无技能
	if _, err := HeroEquipSkill(role, hero, 1, 101); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillNotOwned) {
		t.Fatalf("err = %v, want ErrSkillNotOwned", err)
	}
}

func TestHeroEquipSkill_EquippedOther(t *testing.T) {
	role, heroA := newSkillRole(t, 15, false)
	role.GetSkills().AddSkill(101, 3)
	if _, err := HeroEquipSkill(role, heroA, 1, 101); err != nil {
		t.Fatalf("equip to heroA failed: %v", err)
	}

	// 第二个英雄尝试装配同一技能
	heroBModel := &game_models.RoleHero{ModelBase: models.ModelBase{ID: 1002}, RoleID: role.ID, Level: 15}
	role.GetHeroes().List = append(role.GetHeroes().List, heroBModel)
	heroB := role_heroes.NewRoleHero(heroBModel)

	if _, err := HeroEquipSkill(role, heroB, 1, 101); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillEquippedOther) {
		t.Fatalf("err = %v, want ErrSkillEquippedOther", err)
	}
}

func TestHeroEquipSkill_UseLimitExceed(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	hs := role.GetSkills().AddSkill(101, 1) // 装配次数上限1

	if _, err := HeroEquipSkill(role, hero, 1, 101); err != nil {
		t.Fatalf("first equip failed: %v", err)
	}
	if _, _, err := HeroUnequipSkill(role, hero, 1); err != nil {
		t.Fatalf("unequip failed: %v", err)
	}
	// 已装配1次（UsedCount=1=limit），拆卸后次数不还原 → 再装超限
	if _, err := HeroEquipSkill(role, hero, 1, 101); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillUseLimitExceed) {
		t.Fatalf("err = %v, want ErrSkillUseLimitExceed", err)
	}
	_ = hs
}

// ────────────────────────── 拆卸技能 ──────────────────────────

func TestHeroUnequipSkill_RefundByLevel(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	role.GetSkills().AddSkill(102, 2) // 102 升级消耗 2001x2/级
	addItems(role, 2001, 10)

	if _, err := HeroEquipSkill(role, hero, 1, 102); err != nil {
		t.Fatalf("equip failed: %v", err)
	}
	// 升级 1 级到 level2（投入 2 个道具）
	if _, err := HeroSkillUpgrade(role, hero, 1); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	before := itemCount(role, 2001) // 10-2=8

	skills, refund, err := HeroUnequipSkill(role, hero, 1)
	if err != nil {
		t.Fatalf("unequip failed: %v", err)
	}
	// 返还 = 升级次数(1) * 每级消耗(2) / 2 = 1
	if refund != 1 {
		t.Errorf("refund = %d, want 1", refund)
	}
	if len(skills) != 0 {
		t.Errorf("skills after unequip = %+v, want empty", skills)
	}
	if after := itemCount(role, 2001); after != before+refund {
		t.Errorf("item 2001 after = %d, want %d", after, before+refund)
	}
	// 技能库装配记录清空
	if hs := role.GetSkills().GetSkillByConfID(102); hs.GetEquipHeroID() != 0 {
		t.Errorf("equip_hero_id = %d, want 0 after unequip", hs.GetEquipHeroID())
	}
}

func TestHeroUnequipSkill_Empty(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	if _, _, err := HeroUnequipSkill(role, hero, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillSlotEmpty) {
		t.Fatalf("err = %v, want ErrSkillSlotEmpty", err)
	}
}

// ────────────────────────── 技能升级 ──────────────────────────

func TestHeroSkillUpgrade_Success(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	role.GetSkills().AddSkill(101, 3)
	addItems(role, 2001, 10)
	if _, err := HeroEquipSkill(role, hero, 1, 101); err != nil {
		t.Fatalf("equip failed: %v", err)
	}

	skill, err := HeroSkillUpgrade(role, hero, 1)
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	if skill.GetLevel() != 2 {
		t.Errorf("level = %d, want 2", skill.GetLevel())
	}
	if itemCount(role, 2001) != 9 {
		t.Errorf("item 2001 remain = %d, want 9", itemCount(role, 2001))
	}
	// 技能库等级不同步（保持初始 1）
	if hs := role.GetSkills().GetSkillByConfID(101); hs.GetLevel() != 1 {
		t.Errorf("skill lib level = %d, want 1 (not synced)", hs.GetLevel())
	}
}

func TestHeroSkillUpgrade_SlotEmpty(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	if _, err := HeroSkillUpgrade(role, hero, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillSlotEmpty) {
		t.Fatalf("err = %v, want ErrSkillSlotEmpty", err)
	}
}

func TestHeroSkillUpgrade_MaxLevel(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	role.GetSkills().AddSkill(101, 3)
	addItems(role, 2001, 10)
	if _, err := HeroEquipSkill(role, hero, 1, 101); err != nil {
		t.Fatalf("equip failed: %v", err)
	}
	hero.GetEquipSkillBySlot(1).Level = 10 // 直接设满级（MaxLevel=10）

	if _, err := HeroSkillUpgrade(role, hero, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroSkillMaxLevel) {
		t.Fatalf("err = %v, want ErrSkillMaxLevel", err)
	}
}

func TestHeroSkillUpgrade_CostNotEnough(t *testing.T) {
	role, hero := newSkillRole(t, 15, false)
	role.GetSkills().AddSkill(101, 3)
	if _, err := HeroEquipSkill(role, hero, 1, 101); err != nil {
		t.Fatalf("equip failed: %v", err)
	}
	// 不造道具

	_, err := HeroSkillUpgrade(role, hero, 1)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	r, ok := err.(rpc_results.ResultI)
	if !ok {
		t.Fatalf("err type = %T, want rpc_results.ResultI", err)
	}
	if r.Code() != pb_error_code.ErrorCode_ItemTypeNormalNotEnough {
		t.Errorf("code = %d, want ItemTypeNormalNotEnough", r.Code())
	}
}
