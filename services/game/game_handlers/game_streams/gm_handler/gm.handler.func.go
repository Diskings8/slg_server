package gm_handler

import (
	"context"
	"fmt"
	"strconv"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_gm"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerGm 通用 GM 指令入口 (1000052)
//
// 节点收到 GM 请求后，按 req.cmd 分发到对应功能的 GM 方法：
//   - cmd 未注册 → ParamError
//   - 目标角色默认当前操作角色（req.role_id=0）；指定 role_id 可操作其他角色（GM 调试用）
func HandlerGm(ctx context.Context, roleID uint64, req *pb_gm.GmReq, resp *pb_gm.GmResp) rpc_results.ResultI {
	info, ok := GetGmCmd(req.GetCmd())
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("gm cmd not registered: %s", req.GetCmd()))
	}

	// 目标角色：GmReq.role_id=0 默认当前操作角色
	targetRoleID := req.GetRoleId()
	if targetRoleID == 0 {
		targetRoleID = roleID
	}

	msg, result := info.F(ctx, targetRoleID, req.GetArgs())
	if result != nil {
		return result
	}

	resp.Cmd = req.GetCmd()
	resp.Msg = msg
	return nil
}

// gmHeroSetLevel GM：设置英雄等级（args: hero_id level），经验清零并重算战斗属性。
// 供测试/运营调试用，非玩家路径。
func gmHeroSetLevel(ctx context.Context, roleID uint64, args []string) (string, rpc_results.ResultI) {
	if len(args) < 2 {
		return "", rpc_results.Error(pb_error_code.ErrorCode_ParamError, "usage: gm hero.set_level <hero_id> <level>")
	}

	heroID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return "", rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("invalid hero_id: %s", args[0]))
	}
	level, err := strconv.ParseUint(args[1], 10, 32)
	if err != nil {
		return "", rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("invalid level: %s", args[1]))
	}

	poller, role, gErr := game_roles.GetRole(roleID)
	if gErr != nil {
		return "", gErr
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(heroID)
	if hero == nil {
		return "", rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	if logicErr := game_logics.GmSetHeroLevel(hero, uint32(level)); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return "", r
		}
		return "", rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("gm set hero level failed: %s", logicErr.Error()))
	}

	poller.Save()
	return fmt.Sprintf("hero %d level set to %d", heroID, hero.GetLevel()), nil
}

// init 注册 GM 命令
func init() {
	RegisterGmCmd("hero.set_level", gmHeroSetLevel)
}
