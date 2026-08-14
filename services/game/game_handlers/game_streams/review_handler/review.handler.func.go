package review_handler

import (
	"context"
	"errors"
	"math/rand/v2"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_review"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
	"server.slg.com/services/game/game_logics"
	"server.slg.com/services/internal/cores/map_datas/map_events"
)

// HandlerReviewStart 触发审查：结算每日次数 → 消耗 1 次 → 生成任务 + 经验
func HandlerReviewStart(ctx context.Context, roleID uint64, req *pb_review.ReviewStartReq, resp *pb_review.ReviewStartResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	conf := game_conf.Load().Review
	if conf == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "review config not loaded")
	}

	tasks, expGained, leveledUp, startErr := role.GetReviews().StartReview(conf)
	if startErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, startErr.Error())
	}
	poller.Save() // 打脏，异步持久化

	reviews := role.GetReviews().Review
	resp.Chances = reviews.Chances
	resp.Exp = reviews.Exp
	resp.Level = reviews.Level
	resp.ExpGained = expGained
	resp.LeveledUp = leveledUp
	for _, t := range tasks {
		ti := &pb_review.ReviewTaskInfo{TaskId: t.TaskID, EventType: t.Type}
		for _, rw := range t.Rewards {
			ti.Rewards = append(ti.Rewards, &pb_review.ItemReward{ItemId: rw.ItemID, Count: rw.Count})
		}
		resp.Tasks = append(resp.Tasks, ti)
	}
	return nil
}

// HandlerReviewTaskSelect 选择任务执行：调 worldmap 在主城 5×5 外圈刷出事件
func HandlerReviewTaskSelect(ctx context.Context, roleID uint64, req *pb_review.ReviewTaskSelectReq, resp *pb_review.ReviewTaskSelectResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	task, selErr := role.GetReviews().SelectTask(req.GetTaskId())
	if selErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, selErr.Error())
	}

	// 主城核心格（从建筑取）
	mainCity := role.GetBuildings().GetMainCity()
	if mainCity == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "主城不存在")
	}

	rsp, callErr := game_rpc_clients.WorldMap().SpawnReviewEvent(ctx, &pb_worldmap.SpawnReviewEventReq{
		RoleId:    roleID,
		CoreMapId: mainCity.MapID,
		EventType: task.Type,
	})
	if callErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, callErr.Error())
	}
	resp.MapId = rsp.GetMapId()
	return nil
}

// HandlerEventClick 气泡点击事件（采集/寻宝）：worldmap +进度；完成后按事件类型发奖
func HandlerEventClick(ctx context.Context, roleID uint64, req *pb_worldmap.EventClickReq, resp *pb_worldmap.EventClickRsp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	rsp, callErr := game_rpc_clients.WorldMap().EventClick(ctx, &pb_worldmap.EventClickReq{
		RoleId: roleID,
		MapId:  req.GetMapId(),
	})
	if callErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, callErr.Error())
	}
	resp.Progress = rsp.GetProgress()
	resp.Completed = rsp.GetCompleted()
	resp.EventType = rsp.GetEventType()

	if rsp.GetCompleted() {
		// 完成发奖：宝箱→随机道具；采集→资源道具
		if grantErr := grantReviewEventReward(role, rsp.GetEventType()); grantErr != nil {
			return rpc_results.Error(pb_error_code.ErrorCode_Failed, grantErr.Error())
		}
		poller.Save() // 打脏持久化
	}
	return nil
}

// grantReviewEventReward 按事件类型发放完成奖励（宝箱→随机资源道具；采集→固定资源道具）
func grantReviewEventReward(role *game_roles.Role, eventType int32) error {
	var items []common_declarations.ItemUse
	switch map_events.EventType(eventType) {
	case map_events.EventTypeTreasure:
		// 宝箱：随机一种资源道具
		items = []common_declarations.ItemUse{{ItemID: pb_confs.ItemID(1001 + rand.IntN(4)), Count: 10}}
	case map_events.EventTypeResource:
		// 采集：固定资源道具
		items = []common_declarations.ItemUse{{ItemID: 1001, Count: 10}}
	default:
		return errors.New("未知事件类型")
	}
	return game_logics.ItemChange(role, items, nil, common_declarations.ReasonReward)
}
