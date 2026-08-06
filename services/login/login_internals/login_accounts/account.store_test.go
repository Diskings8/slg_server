package login_accounts

// 账户/渠道绑定/角色映射存储单测 — 使用 sqlite 内存库。
// 覆盖：注册（账号+首渠道事务）、account_name 全局唯一、渠道账号唯一、多渠道绑同一账号、认证信息更新。

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/login/login_models"
)

func newTestAccountStore(t *testing.T) *AccountStore {
	t.Helper()
	// CreateAccountWithChannel 用雪花 ID 生成主键，需初始化（loggers 不依赖配置，snowflake 空配置建节点 0）
	loggers.Init()
	snowflakes.Init()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	s := NewAccountStore(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return s
}

func TestCreateAccountWithChannel(t *testing.T) {
	s := newTestAccountStore(t)

	acc := &login_models.Account{AccountName: "player1", PasswordHash: "abc"}
	binding := &login_models.ChannelAccount{ChannelType: 0, ChannelAccount: "player1"}
	if err := s.CreateAccountWithChannel(acc, binding); err != nil {
		t.Fatalf("create account with channel: %v", err)
	}
	if acc.ID == 0 {
		t.Fatal("account id should be auto-generated")
	}
	if binding.AccountID != acc.ID {
		t.Fatalf("binding.account_id should equal account id: acc=%d binding=%d", acc.ID, binding.AccountID)
	}

	byName, err := s.GetAccountByName("player1")
	if err != nil || byName == nil || byName.ID != acc.ID {
		t.Fatalf("get by name: got=%v err=%v", byName, err)
	}
	byID, err := s.GetAccountByID(acc.ID)
	if err != nil || byID == nil {
		t.Fatalf("get by id: got=%v err=%v", byID, err)
	}
	b, err := s.GetChannel(0, "player1")
	if err != nil || b == nil || b.AccountID != acc.ID {
		t.Fatalf("get channel binding: got=%v err=%v", b, err)
	}
}

func TestCreateAccountDuplicate(t *testing.T) {
	s := newTestAccountStore(t)

	// account_name 全局唯一（与渠道无关）
	if err := s.CreateAccountWithChannel(
		&login_models.Account{AccountName: "dup", PasswordHash: "a"},
		&login_models.ChannelAccount{ChannelType: 0, ChannelAccount: "dup"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	// 同名不同渠道 → 仍冲突（全局唯一）
	if err := s.CreateAccountWithChannel(
		&login_models.Account{AccountName: "dup", PasswordHash: "b"},
		&login_models.ChannelAccount{ChannelType: 1, ChannelAccount: "dup"}); err != ErrAccountExists {
		t.Fatalf("duplicate account_name should be ErrAccountExists, got: %v", err)
	}
	// 不同账号名 + 同一渠道账号 → 渠道已被占用
	if err := s.CreateAccountWithChannel(
		&login_models.Account{AccountName: "other", PasswordHash: "c"},
		&login_models.ChannelAccount{ChannelType: 0, ChannelAccount: "dup"}); err != ErrChannelExists {
		t.Fatalf("occupied channel should be ErrChannelExists, got: %v", err)
	}
}

func TestChannelBindingMultiChannel(t *testing.T) {
	s := newTestAccountStore(t)

	acc := &login_models.Account{AccountName: "player1", PasswordHash: "a"}
	if err := s.CreateAccountWithChannel(acc, &login_models.ChannelAccount{ChannelType: 0, ChannelAccount: "player1"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 同一账号可绑多渠道
	if err := s.CreateChannel(&login_models.ChannelAccount{AccountID: acc.ID, ChannelType: 1, ChannelAccount: "wx_123"}); err != nil {
		t.Fatalf("bind second channel: %v", err)
	}

	// 同一渠道账号只能映射到一个账号
	if err := s.CreateChannel(&login_models.ChannelAccount{AccountID: 999, ChannelType: 1, ChannelAccount: "wx_123"}); err != ErrChannelExists {
		t.Fatalf("channel already bound should be ErrChannelExists, got: %v", err)
	}

	// 认证信息留痕更新
	b, _ := s.GetChannel(0, "player1")
	if err := s.UpdateChannelAuthInfo(b.ID, "sign_abc"); err != nil {
		t.Fatalf("update auth_info: %v", err)
	}
	after, _ := s.GetChannel(0, "player1")
	if after.AuthInfo != "sign_abc" {
		t.Fatalf("auth_info not updated: %+v", after)
	}
}

func TestRoleMapping(t *testing.T) {
	s := newTestAccountStore(t)

	role := &login_models.Role{
		RoleID:    10001,
		AccountID: 1,
		ServerID:  1,
		RoleName:  "HeroA",
	}
	if err := s.CreateRole(role); err != nil {
		t.Fatalf("create role: %v", err)
	}

	got, err := s.GetRoleByAccountServer(1, 1)
	if err != nil || got == nil || got.RoleID != 10001 {
		t.Fatalf("get by account+server: got=%v err=%v", got, err)
	}

	byName, err := s.GetRoleByName(1, "HeroA")
	if err != nil || byName == nil {
		t.Fatalf("get by name: got=%v err=%v", byName, err)
	}

	list, err := s.GetRolesByAccount(1)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by account: len=%d err=%v", len(list), err)
	}

	// 撞唯一索引：服内角色名重复
	if err := s.CreateRole(&login_models.Role{
		RoleID:    10002,
		AccountID: 2,
		ServerID:  1,
		RoleName:  "HeroA",
	}); err != ErrRoleExists {
		t.Fatalf("duplicate role name should be ErrRoleExists, got: %v", err)
	}
	// 撞唯一索引：每账号每服一个角色
	if err := s.CreateRole(&login_models.Role{
		RoleID:    10003,
		AccountID: 1,
		ServerID:  1,
		RoleName:  "HeroB",
	}); err != ErrRoleExists {
		t.Fatalf("duplicate account+server should be ErrRoleExists, got: %v", err)
	}
}

func TestUpdateLastLogin(t *testing.T) {
	s := newTestAccountStore(t)

	acc := &login_models.Account{AccountName: "last", PasswordHash: "a"}
	if err := s.CreateAccountWithChannel(acc, &login_models.ChannelAccount{ChannelType: 0, ChannelAccount: "last"}); err != nil {
		t.Fatalf("create account: %v", err)
	}

	if err := s.UpdateLastLogin(acc.ID, 2, 20001); err != nil {
		t.Fatalf("update last login: %v", err)
	}
	got, _ := s.GetAccountByID(acc.ID)
	if got.LastLoginServerID != 2 || got.LastLoginRoleID != 20001 {
		t.Fatalf("last login not updated: %+v", got)
	}
}
