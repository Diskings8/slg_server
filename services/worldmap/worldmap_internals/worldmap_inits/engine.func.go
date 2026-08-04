package worldmap_inits

import (
	"context"
	"fmt"
	"time"

	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/redisstream"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos/march_factory"
	"server.slg.com/services/internal/cores/marchs"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// Engine cores 引擎聚合
type Engine struct {
	ctx              context.Context
	Config           *DefaultMapConfig
	MapDataManager   *map_datas.MapDataManager
	MarchInfoManager *marchs.MarchInfoManager
	MapManager       *map_managers.MapManager

	battleHub *rpc_handlers.ClientHandler // 战斗节点客户端（battle 结算回调）
}

// NewEngine 初始化 cores 引擎
func NewEngine(ctx context.Context) *Engine {
	mapConfig := NewDefaultMapConfig()

	mapData := map_datas.NewMapDataManager(mapConfig, "map_data")

	// 初始化地图元素（限定元素集合 + 种子确定性生成），保证视野查询有数据、出生点可诞生
	InitMapElements(mapData, defaultMapSeed)

	tickerChan := make(chan *marchs.MarchInfo, 1000)
	marchMgr := marchs.New(tickerChan, "march_info", mapConfig, cores_declarations.MarchTimeTypeStraight)

	e := &Engine{
		ctx:              ctx,
		Config:           mapConfig,
		MapDataManager:   mapData,
		MarchInfoManager: marchMgr,
	}

	// 行军执行回调 — 到达时创建执行器并结算，结算完成后通知 game
	marchDoFunc := func(mm *map_managers.MapManager, marchID cores_declarations.MarchID) {
		e.MarchTickHandler(mm, marchID)
	}
	// 行军执行器工厂 — 按 MarchType 分派 attack/develop/assist 等
	marchDoHandleFunc := func(mm *map_managers.MapManager, info *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
		return march_factory.NewMarchDo(mm, info)
	}

	manager := map_managers.NewMapManager(
		1,
		cores_declarations.MapGroupBase,
		mapData,
		marchMgr,
		marchDoFunc,
		marchDoHandleFunc,
	)
	manager.Start()

	e.MapManager = manager

	// 注入战斗结算回调（内部调用 battle 节点 RPC）
	e.initBattleSettle(manager)

	return e
}

// initBattleSettle 初始化战斗结算客户端并注入回调
func (e *Engine) initBattleSettle(mm *map_managers.MapManager) {
	e.battleHub = rpc_handlers.NewClientHandler(*vgc.CommonGlobalVarInstance)
	mm.SetBattleSettleFunc(e.settleBattle)
}

// settleBattle 注入回调实现：调用 battle 节点 BattleSettle RPC
func (e *Engine) settleBattle(req *pb_battle.BattleSettleReq) (*pb_battle.BattleSettleRsp, error) {
	if e.battleHub == nil {
		return nil, fmt.Errorf("battle hub not initialized")
	}
	cli := e.battleHub.GetBattleHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("battle node not connected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rsp, err := cli.BattleSettle(ctx, req)
	if err == nil && rsp != nil {
		// 异步保存战报（fire-and-forget，不阻塞战斗 tick）
		e.saveBattleRecord(req, rsp)
	}
	return rsp, err
}

// saveBattleRecord 异步保存战报到 battle_record 节点（角色/联盟/地块三维索引）
func (e *Engine) saveBattleRecord(req *pb_battle.BattleSettleReq, rsp *pb_battle.BattleSettleRsp) {
	cli := e.battleHub.GetBattleRecordHandlerClient()
	if cli == nil {
		loggers.Logger.Warn("battle_record node not connected, skip save")
		return
	}

	saveReq := &pb_battle_record.SaveBattleRecordReq{
		MarchId:         req.GetMarchId(),
		AttackerRoleId:  req.GetRoleId(),
		AttackerUnionId: req.GetUnionId(),
		MapId:           req.GetMapId(),
		MarchType:       req.GetMarchType(),
		AttackerWin:     rsp.GetAttackerWin(),
		IsOccupied:      rsp.GetOccupied(),
		BuildingDamage:  rsp.GetBuildingDamage(),
		Results:         rsp.GetResults(),
		BattleTime:      time.Now().Unix(),
	}

	// 防守方角色/联盟去重
	roleSeen := make(map[uint64]struct{})
	unionSeen := make(map[uint64]struct{})
	for _, g := range req.GetDefenderGroups() {
		for _, d := range g.GetMarches() {
			if d == nil {
				continue
			}
			if d.GetRoleId() > 0 {
				if _, ok := roleSeen[d.GetRoleId()]; !ok {
					roleSeen[d.GetRoleId()] = struct{}{}
					saveReq.DefenderRoleIds = append(saveReq.DefenderRoleIds, d.GetRoleId())
				}
			}
			if d.GetUnionId() > 0 {
				if _, ok := unionSeen[d.GetUnionId()]; !ok {
					unionSeen[d.GetUnionId()] = struct{}{}
					saveReq.DefenderUnionIds = append(saveReq.DefenderUnionIds, d.GetUnionId())
				}
			}
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := cli.SaveBattleRecord(ctx, saveReq); err != nil {
			loggers.Logger.Warn("save battle record failed",
				zap.Uint64("march_id", req.GetMarchId()),
				zap.Error(err))
		}
	}()
}

// Stop 停止引擎
func (e *Engine) Stop() {
	e.MapManager.Stop()
}

// MarchTickHandler 行军到达处理器
//
// 参考 march_factory.MarchTickHandler 的标准流程：
//  1. 取得行军，未到时间则重新入队等待
//  2. 锁定行军，按 MarchType 创建执行器（attack/develop/assist...）
//  3. Do() 执行到达业务（战斗结算 / 采集 / 驻守）
//  4. 执行失败则 CallBack 召回
//  5. 结算完成后通过 Redis Stream 通知 game
func (e *Engine) MarchTickHandler(mm *map_managers.MapManager, marchID cores_declarations.MarchID) {
	marchInfo := mm.GetMarchManage().GetMarchInfo(marchID)
	if marchInfo == nil {
		return
	}

	// 未到到达时间，重新排队等待 tick
	_, endTime := marchInfo.GetMarchStartAndEndTimeUx()
	if endTime > time.Now().Unix() {
		mm.GetMarchManage().TickerChan <- marchInfo
		return
	}

	if !marchInfo.LockMarchDo() {
		return
	}
	defer marchInfo.UnlockMarchDo()

	handle := march_factory.NewMarchDo(mm, marchInfo)
	if handle == nil {
		return
	}

	toMapLock := marchInfo.GetMarchState() != pb_maps_march.MarchState_Back
	handle.Lock(true, false, toMapLock)
	err := handle.Do()
	handle.Unlock()

	if err != nil {
		// 结算失败，召回行军
		_ = handle.CallBack()
		return
	}

	// 结算成功，通知 game（回城到站也会走到这里，由 state 区分）
	e.OnMarchArrived(marchInfo)
}

// OnMarchArrived 行军到达/结算完成后的回调处理
func (e *Engine) OnMarchArrived(marchInfo *marchs.MarchInfo) {
	if marchInfo == nil {
		return
	}

	_, toMapID, _ := marchInfo.GetMapIDs()
	loggers.Logger.Info("march arrived",
		zap.Uint64("march_id", marchInfo.GetMarchID().Uint64()),
		zap.Uint64("role_id", marchInfo.GetFromRoleID()),
		zap.Int32("to", toMapID.Int32()),
		zap.Uint32("march_type", uint32(marchInfo.MarchType)),
		zap.Int32("state", int32(marchInfo.MarchState)))

	// 回城到站（MarchState_Back）用独立事件类型，便于 game 区分回城结算与目标点结算
	eventType := pb_redis_stream.MarchEventType_MARCH_EVENT_ARRIVED
	if marchInfo.MarchState == pb_maps_march.MarchState_Back {
		eventType = pb_redis_stream.MarchEventType_MARCH_EVENT_BACKARRIVED

		// 战败召回：行军在返回途中（尚未回城到站）不发事件；
		// 真正回城到站（BackArrive 已 DeleteMarch，行军不在管理器）才发 BACKARRIVED。
		if e.MapManager.GetMarchManage().GetMarchInfo(marchInfo.GetMarchID()) != nil {
			loggers.Logger.Info("march defeated, recalling to transit, BACKARRIVED deferred",
				zap.Uint64("march_id", marchInfo.GetMarchID().Uint64()))
			return
		}
	}

	publishMarchEvent(e.ctx, &pb_redis_stream.MarchEvent{
		Type:      eventType,
		MarchId:   marchInfo.GetMarchID().Uint64(),
		RoleId:    marchInfo.GetFromRoleID(),
		ToMapId:   toMapID.Int32(),
		MarchType: int32(marchInfo.MarchType),
		State:     int32(marchInfo.MarchState),
		Ts:        time.Now().Unix(),
	})
}

// OnMarchCanceled 行军被取消时的回调
func (e *Engine) OnMarchCanceled(marchInfo *marchs.MarchInfo) {
	loggers.Logger.Info("march canceled",
		zap.Uint64("march_id", marchInfo.GetMarchID().Uint64()),
		zap.Uint64("role_id", marchInfo.GetFromRoleID()))

	publishMarchEvent(e.ctx, &pb_redis_stream.MarchEvent{
		Type:    pb_redis_stream.MarchEventType_MARCH_EVENT_CANCELED,
		MarchId: marchInfo.GetMarchID().Uint64(),
		RoleId:  marchInfo.GetFromRoleID(),
		Ts:      time.Now().Unix(),
	})
}

// publishMarchEvent 发布行军事件到 Redis Stream (XADD)
func publishMarchEvent(ctx context.Context, event *pb_redis_stream.MarchEvent) {
	data, err := proto.Marshal(event)
	if err != nil {
		loggers.Logger.Warn("march event marshal failed", zap.Error(err))
		return
	}

	if err := redisstream.ProtoXAdd(ctx, redisstream.StreamKeyMarchEvents, data); err != nil {
		loggers.Logger.Warn("publish march event to redis stream failed",
			zap.String("event_type", event.Type.String()),
			zap.Uint64("march_id", event.MarchId),
			zap.Error(err))
	}
}
