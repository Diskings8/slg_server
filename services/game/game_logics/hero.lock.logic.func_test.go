package game_logics

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_error_code"
)

// TestHeroLock_Unlock 锁定/解锁设置 IsLocked
func TestHeroLock_Unlock(t *testing.T) {
	_, hero := newStarRole(t, 0, 1)

	HeroLock(hero)
	if !hero.GetIsLocked() {
		t.Fatal("hero should be locked after HeroLock")
	}

	HeroUnlock(hero)
	if hero.GetIsLocked() {
		t.Fatal("hero should be unlocked after HeroUnlock")
	}
}

// TestHeroUpgradeStar_LockedCardNotConsumable 锁定的英雄卡不可作为升星消耗卡
func TestHeroUpgradeStar_LockedCardNotConsumable(t *testing.T) {
	role, hero := newStarRole(t, 0, 1) // 消耗卡 3001
	role.GetHeroes().List[1].IsLocked = true

	if err := HeroUpgradeStar(role, hero); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroNoConsumeCard) {
		t.Fatalf("err = %v, want HeroNoConsumeCard (locked card)", err)
	}
	if hero.GetStarStage() != 0 {
		t.Errorf("star_stage = %d, want 0 (locked card not consumed)", hero.GetStarStage())
	}
}

// TestHeroUpgradeStar_SkipsLockedCard 多张卡时跳过锁定的，用未锁定的
func TestHeroUpgradeStar_SkipsLockedCard(t *testing.T) {
	role, hero := newStarRole(t, 0, 2) // 3001/3002
	role.GetHeroes().List[1].IsLocked = true // 3001 锁定

	if err := HeroUpgradeStar(role, hero); err != nil {
		t.Fatalf("upgrade star failed: %v", err)
	}
	if hero.GetStarStage() != 1 {
		t.Errorf("star_stage = %d, want 1", hero.GetStarStage())
	}
	// 3001（锁定）保留，3002 被消耗 → List = hero + 3001
	if len(role.GetHeroes().List) != 2 {
		t.Errorf("hero count = %d, want 2", len(role.GetHeroes().List))
	}
}
