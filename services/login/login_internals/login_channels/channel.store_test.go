//go:build integration

package login_channels_test

// 渠道声明存储单测 — 连真实 mysql（common_db_0）。
// 运行：go test -tags integration ./services/login/...

import (
	"testing"

	"server.slg.com/services/login/login_internals/login_channels"
	"server.slg.com/services/login/login_testutil"
)

func newStore(t *testing.T) *login_channels.ChannelStore {
	t.Helper()
	_, chStore, _ := login_testutil.SetupStores(t)
	return chStore
}

func TestSeedDefaultIdempotent(t *testing.T) {
	s := newStore(t)

	if err := s.SeedDefault(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s.SeedDefault(); err != nil {
		t.Fatalf("seed again: %v", err)
	}

	ch, err := s.GetChannel(0)
	if err != nil || ch == nil {
		t.Fatalf("get official channel: got=%v err=%v", ch, err)
	}
	if ch.ChannelName != "官方渠道" || ch.Status != 0 {
		t.Fatalf("unexpected official channel: %+v", ch)
	}

	// 未声明渠道返回 nil
	miss, err := s.GetChannel(999)
	if err != nil || miss != nil {
		t.Fatalf("undeclared channel should be nil: got=%v err=%v", miss, err)
	}
}
