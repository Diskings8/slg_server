//go:build integration

package login_servers_store_test

// 区服存储单测 — 连真实 mysql（common_db_0）。
// 运行：go test -tags integration ./services/login/...

import (
	"testing"

	"server.slg.com/services/login/login_internals/login_servers_store"
	"server.slg.com/services/login/login_testutil"
)

func newStore(t *testing.T) *login_servers_store.ServerStore {
	t.Helper()
	_, _, svStore := login_testutil.SetupStores(t)
	return svStore
}

func TestSeedIfEmptyIdempotent(t *testing.T) {
	s := newStore(t)

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
	found := false
	for _, sv := range list {
		if sv.ID == 1 && sv.ServerName == "S1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("seed server 1 (S1) missing: %+v", list)
	}

	sv, err := s.GetServer(1)
	if err != nil || sv == nil {
		t.Fatalf("get server 1: got=%v err=%v", sv, err)
	}
	miss, err := s.GetServer(99999)
	if err != nil || miss != nil {
		t.Fatalf("missing server should be nil: got=%v err=%v", miss, err)
	}
}
