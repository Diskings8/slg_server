package login_servers

// 区服存储单测 — 使用 sqlite 内存库，覆盖：种子幂等 / 列表 / 查询。

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestServerStore(t *testing.T) *ServerStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	s := NewServerStore(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return s
}

func TestSeedIfEmptyIdempotent(t *testing.T) {
	s := newTestServerStore(t)

	if err := s.SeedIfEmpty(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s.SeedIfEmpty(); err != nil {
		t.Fatalf("seed again: %v", err)
	}

	list, err := s.ListServers()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("seed should insert exactly one server, got %d", len(list))
	}
	if list[0].ID != 1 || list[0].ServerName != "S1" {
		t.Fatalf("unexpected seed server: %+v", list[0])
	}

	sv, err := s.GetServer(1)
	if err != nil || sv == nil {
		t.Fatalf("get server 1: got=%v err=%v", sv, err)
	}
	miss, err := s.GetServer(99)
	if err != nil || miss != nil {
		t.Fatalf("missing server should be nil: got=%v err=%v", miss, err)
	}
}
