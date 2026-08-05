package game_roles

import (
	"sync"
	"testing"

	"server.slg.com/common/utils/util_jsons"
)

func TestRoleNewAndReset(t *testing.T) {
	r := NewTest(1001)
	if r.ID != 1001 {
		t.Fatalf("expected ID 1001, got %d", r.ID)
	}

	// 验证 Init 不会 panic
	r.Init()

	// 验证 Reset 清理
	r.Reset()
	if r.ID != 0 {
		t.Fatal("Reset should clear ID")
	}
	if r.IsCopy() {
		t.Fatal("after Reset, IsCopy should be false")
	}
}

func TestRoleMarshalUnmarshal(t *testing.T) {
	r := NewTest(1002)
	r.IP = "127.0.0.1"
	r.Status = 1
	r.StatusEndTime = 0

	b, err := r.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	r2 := &Role{}
	if err := r2.Unmarshal(b); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if r2.ID != r.ID {
		t.Fatalf("expected ID %d, got %d", r.ID, r2.ID)
	}
}

func TestRoleMarshalJSON(t *testing.T) {
	r := NewTest(1003)
	r.Status = 1
	r.StatusEndTime = 0

	b, err := util_jsons.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var r2 Role
	if err := util_jsons.Unmarshal(b, &r2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if r2.ID != r.ID {
		t.Fatalf("expected ID %d, got %d", r.ID, r2.ID)
	}
	if r2.Status != r.Status {
		t.Fatalf("expected Status %d, got %d", r.Status, r2.Status)
	}
}

func TestRolePool(t *testing.T) {
	r := get()
	if r == nil {
		t.Fatal("Get() returned nil")
	}
	r.ID = 2001

	// 放回池中后 Reset 会被调用
	release(r)

	// 重新获取，应已 Reset
	r2 := get()
	if r2.ID != 0 {
		t.Fatal("pool Get should return Reset object")
	}
}

func TestRoleCopy(t *testing.T) {
	orig := NewTest(3001)
	orig.GateID = 5
	orig.IP = "10.0.0.1"
	orig.SetStatus(1, 0)

	// 模拟 copyLock
	// Copy 方法内部使用了 rolePool.Get 和 src 指针
	copyLock := &sync.RWMutex{}
	copied := orig.Copy(copyLock).(*Role)

	if copied.ID != orig.ID {
		t.Fatalf("expected ID %d, got %d", orig.ID, copied.ID)
	}
	if !copied.IsCopy() {
		t.Fatal("copy should report IsCopy() = true")
	}
	if copied.GateID != orig.GateID {
		t.Fatalf("expected GateID %d, got %d", orig.GateID, copied.GateID)
	}
	if copied.IP != orig.IP {
		t.Fatalf("expected IP %s, got %s", orig.IP, copied.IP)
	}
}
