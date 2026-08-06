package login_channels

// 渠道声明存储单测 — 使用 sqlite 内存库，覆盖：默认种子幂等 / 声明查询 / 未声明返回 nil。

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"server.slg.com/services/login/login_models"
)

func newTestChannelStore(t *testing.T) *ChannelStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	s := NewChannelStore(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return s
}

func TestSeedDefaultIdempotent(t *testing.T) {
	s := newTestChannelStore(t)

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
	miss, err := s.GetChannel(99)
	if err != nil || miss != nil {
		t.Fatalf("undeclared channel should be nil: got=%v err=%v", miss, err)
	}
}

func TestGetChannel(t *testing.T) {
	s := newTestChannelStore(t)

	if err := s.db.Create(&login_models.Channel{ChannelType: 1, ChannelName: "测试渠道", Status: 0}).Error; err != nil {
		t.Fatalf("declare channel: %v", err)
	}
	ch, err := s.GetChannel(1)
	if err != nil || ch == nil || ch.ChannelName != "测试渠道" {
		t.Fatalf("get channel 1: got=%v err=%v", ch, err)
	}
}
