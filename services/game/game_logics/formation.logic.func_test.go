package game_logics

import (
	"testing"
)

// TestSetHeroSlotOneBased 槽位 1 基：1=大营可设置，越界（>3）返回 nil
func TestSetHeroSlotOneBased(t *testing.T) {
	// 大营（slot 1）
	slots := setHeroSlot(nil, 1, 9001, 100)
	if slots == nil || len(slots) != 1 || slots[0].GetHeroId() != 9001 || slots[0].GetSoldierNum() != 100 {
		t.Fatalf("slot 1（大营）应可设置，实际 %+v", slots)
	}
	// 1号位（slot 2）
	slots = setHeroSlot(slots, 2, 9002, 200)
	if slots == nil || len(slots) != 2 || slots[1].GetHeroId() != 9002 {
		t.Fatalf("slot 2 应可设置，实际 %+v", slots)
	}
	// 越界（slot 4 > maxFormationSlots）
	if slots := setHeroSlot(slots, 4, 9003, 300); slots != nil {
		t.Fatalf("slot 4 应越界返回 nil，实际 %+v", slots)
	}
}

// TestRemoveHeroSlotAtOneBased 清空 slot 1（大营），保持位置不变
func TestRemoveHeroSlotAtOneBased(t *testing.T) {
	slots := setHeroSlot(nil, 1, 9001, 100)
	slots = setHeroSlot(slots, 2, 9002, 200)
	slots = removeHeroSlotAt(slots, 1)
	if slots[0] != nil || slots[1] == nil || slots[1].GetHeroId() != 9002 {
		t.Fatalf("清空大营后位置应保持，实际 %+v", slots)
	}
}
