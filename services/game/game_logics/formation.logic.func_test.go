package game_logics

import (
	"testing"
)

// TestSetHeroSlotZeroBased 槽位 0 基：0=大营可设置，越界（>=3）返回 nil
func TestSetHeroSlotZeroBased(t *testing.T) {
	// 大营（slot 0）
	slots := setHeroSlot(nil, 0, 9001, 100)
	if slots == nil || len(slots) != 1 || slots[0].GetHeroId() != 9001 || slots[0].GetSoldierNum() != 100 {
		t.Fatalf("slot 0（大营）应可设置，实际 %+v", slots)
	}
	// 1号位
	slots = setHeroSlot(slots, 1, 9002, 200)
	if slots == nil || len(slots) != 2 || slots[1].GetHeroId() != 9002 {
		t.Fatalf("slot 1 应可设置，实际 %+v", slots)
	}
	// 越界（slot 3 >= maxFormationSlots）
	if slots := setHeroSlot(slots, 3, 9003, 300); slots != nil {
		t.Fatalf("slot 3 应越界返回 nil，实际 %+v", slots)
	}
}

// TestRemoveHeroSlotAtZeroBased 清空 slot 0（大营），保持位置不变
func TestRemoveHeroSlotAtZeroBased(t *testing.T) {
	slots := setHeroSlot(nil, 0, 9001, 100)
	slots = setHeroSlot(slots, 1, 9002, 200)
	slots = removeHeroSlotAt(slots, 0)
	if slots[0] != nil || slots[1] == nil || slots[1].GetHeroId() != 9002 {
		t.Fatalf("清空大营后位置应保持，实际 %+v", slots)
	}
}
