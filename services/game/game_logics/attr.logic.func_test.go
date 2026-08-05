package game_logics

import (
	"testing"
	"time"

	"server.slg.com/services/game/game_entitys/game_roles"
)

// TestAttrDefaults 新建角色的 Attr 默认值：ServerID=0，VIPLevel=1（未过期）
func TestAttrDefaults(t *testing.T) {
	role := game_roles.NewTest(50001)
	if got := role.ServerID(); got != 0 {
		t.Errorf("ServerID() = %d, want 0 (default)", got)
	}
	if got := role.VIPLevel(); got != 1 {
		t.Errorf("VIPLevel() = %d, want 1 (default)", got)
	}
}

// TestAttrUpdateLogin 首次登录记录 CreateAt/LoginAt，LoginDays=1；同日再登录不增加
func TestAttrUpdateLogin(t *testing.T) {
	role := game_roles.NewTest(50001)
	attr := role.GetAttr()

	attr.UpdateLogin()
	model := attr.Ensure()
	if model.CreateAt == 0 {
		t.Error("CreateAt should be set on first login")
	}
	if model.LoginAt == 0 {
		t.Error("LoginAt should be set on login")
	}
	if model.LoginDays != 1 {
		t.Errorf("LoginDays = %d, want 1 (first login)", model.LoginDays)
	}

	// 同日再登录：跨日判断不成立，LoginDays 不变
	attr.UpdateLogin()
	if model.LoginDays != 1 {
		t.Errorf("LoginDays = %d, want 1 (same day)", model.LoginDays)
	}
}

// TestAttrUpdateLogout 登出累计在线时长（自 LoginAt 起）并记录 LogoutAt
func TestAttrUpdateLogout(t *testing.T) {
	role := game_roles.NewTest(50001)
	attr := role.GetAttr()
	attr.Ensure().LoginAt = time.Now().Unix() - 100 // 模拟 100 秒前登录

	attr.UpdateLogout()
	model := attr.Ensure()
	if model.LogoutAt == 0 {
		t.Error("LogoutAt should be set on logout")
	}
	if model.OnlineTime < 100 {
		t.Errorf("OnlineTime = %d, want >= 100", model.OnlineTime)
	}
}

// TestAttrVipLevelEffective 未过期返回 VipLevel，已过期（VipEndTime 早于当前）返回 0
func TestAttrVipLevelEffective(t *testing.T) {
	role := game_roles.NewTest(50001)
	attr := role.GetAttr()
	attr.Ensure().VipLevel = 3
	if got := attr.VipLevelEffective(); got != 3 {
		t.Errorf("VipLevelEffective() = %d, want 3 (active)", got)
	}

	attr.Ensure().VipEndTime = time.Now().Unix() - 1
	if got := attr.VipLevelEffective(); got != 0 {
		t.Errorf("VipLevelEffective() = %d, want 0 (expired)", got)
	}
}

// TestAttrFormat2Pb Format2Pb 字段映射
func TestAttrFormat2Pb(t *testing.T) {
	role := game_roles.NewTest(50001)
	attr := role.GetAttr()
	attr.Ensure().ServerID = 1
	attr.Ensure().VipLevel = 2

	pb := attr.Format2Pb()
	if pb.GetServerId() != 1 || pb.GetVipLevel() != 2 {
		t.Errorf("Format2Pb = %+v, want server_id=1 vip_level=2", pb)
	}
	if pb.GetLoginDays() != 0 || pb.GetOnlineTime() != 0 {
		t.Errorf("Format2Pb login stats = %+v, want zero", pb)
	}
}
