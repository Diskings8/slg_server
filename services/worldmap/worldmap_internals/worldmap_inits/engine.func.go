package worldmap_inits

import (
	"context"
	"fmt"
	"time"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/game_conf/hero"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/redisstream"
	"server.slg.com/common/utils/asyncsave_entity"
	"server.slg.com/common/utils/crontabs"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_handler"
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
	MarchHandler     *map_handler.MarchHandler // 行军业务编排（校验 + 持久化 + AOI + 推送）

	battleHub *rpc_handlers.ClientHandler // 战斗节点客户端（battle 结算回调）
}

// NewEngine 初始化 cores 引擎
func NewEngine(ctx context.Context) *Engine {
	mapConfig := NewDefaultMapConfig()

	mapData := map_datas.NewMapDataManager(mapConfig, "map_data")

	// 初始化地图元素（限定元素集合 + 种子确定性生成），保证视野查询有数据、出生点可诞生。
	// 种子从配置读取（同种子 → 同底图），DB 动态状态随后由 InitMapData 覆盖加载。
	InitMapElements(mapData, resolveMapSeed())

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

	// 装配行军业务编排 handler（CreateMarch 等 RPC 走此链路：校验 → 持久化 → AOI → 推送）
	e.MarchHandler = &map_handler.MarchHandler{
		Manage: func() *map_managers.MapManager { return manager },
		March:  func() *marchs.MarchInfoManager { return marchMgr },
	}

	// 注入战斗结算回调（内部调用 battle 节点 RPC）
	e.initBattleSettle(manager)
	// 注入守军配置回调（开发行军战斗用，内部查 game_conf）
	e.initGuardConfig(manager)

	return e
}

// initBattleSettle 初始化战斗结算客户端并注入回调
func (e *Engine) initBattleSettle(mm *map_managers.MapManager) {
	e.battleHub = rpc_handlers.NewClientHandler(*vgc.CommonGlobalVarInstance)
	mm.SetBattleSettleFunc(e.settleBattle)
}

// initGuardConfig 注入守军配置回调：按地块等级返回守军队伍快照。
// 闭包内懒加载 game_conf（配置在 main 的 AsyncInit 中初始化，运行时查表）。
// 守军为 NPC（role_id=0），英雄属性由 hero 配置派生：base + growth*(level-1)。
func (e *Engine) initGuardConfig(mm *map_managers.MapManager) {
	mm.SetGuardConfigFunc(func(level cores_declarations.MapLevel) []*pb_battle.TeamSlotInfo {
		gc := game_conf.Load()
		if gc == nil || gc.Guard == nil {
			return nil
		}
		guardConf := gc.Guard.GetGuard(int32(level))
		if guardConf == nil {
			return nil
		}
		slots := make([]*pb_battle.TeamSlotInfo, 0, len(guardConf.Slots))
		for i, s := range guardConf.Slots {
			heroInfo := buildGuardHeroInfo(gc.Hero, s.HeroConfID, int32(level))
			// 守军为 NPC：HeroInfo 无属性时兜底空对象，兵力字段填入
			if heroInfo == nil {
				heroInfo = &pb_hero.HeroInfo{}
			}
			heroInfo.HeroId = uint64(s.HeroConfID)
			heroInfo.SoldierInfo = &pb_hero.SoldierInfo{
				MaxSoldierNum: s.SoldierNum,
				CurAliveNum:   s.SoldierNum,
			}
			slots = append(slots, &pb_battle.TeamSlotInfo{
				SlotId:   int32(i),
				HeroInfo: heroInfo,
			})
		}
		return slots
	})
}

// buildGuardHeroInfo 构造守军英雄快照：属性 = base + growth*(level-1)，填入 Cultivate 组件。
// 英雄配置不存在时返回 nil（battle 侧按空属性兜底，但守军等级通常配置存在）。
func buildGuardHeroInfo(heroConf *hero.Conf, confID int32, level int32) *pb_hero.HeroInfo {
	if heroConf == nil {
		return nil
	}
	hc, ok := heroConf.HeroConf(confID)
	if !ok {
		return nil
	}

	base := hc.Base
	growth := hc.Growth
	lv := uint32(1)
	if level > 1 {
		lv = uint32(level)
	}
	factor := lv - 1

	cur := func(b, g uint32) *pb_cultivate.Cultivate {
		return &pb_cultivate.Cultivate{CurVal: b + g*factor}
	}

	return &pb_hero.HeroInfo{
		ConfigId:         confID,
		CurLevel:         lv,
		CurStatus:        pb_hero.Status_Normal,
		AttrAttack:       cur(base.Attack, growth.Attack),
		AttrDefense:      cur(base.Defense, growth.Defense),
		AttrIntelligence: cur(base.Intelligence, growth.Intelligence),
		AttrMovement:     cur(base.Movement, growth.Movement),
		AttrRelocation:   cur(base.Relocation, growth.Relocation),
	}
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

	// TODO(车轮战编排，未实现)：车轮战 n 队应整合到一个主战报。
	// 需要先建主战报（parent_id=0）拿到 id，再通过行军链路透传 parent_id 到每次结算的 SaveBattleRecord。
	// 当前每次结算都存独立主战报（parent_id=0）。
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

// InitMarchs 重启恢复行军：注册异步保存实体 + 从 DB 加载 + 重做 AOI 注册。
// 必须在 DB 初始化后调用；NewEngine 内的 MapManager.Start() 已先行执行，
// loopTickAccept 正在消费 TickerChan，因此 Init 内的阻塞发送不会卡死。
func (e *Engine) InitMarchs() error {
	// 1. 注册 MarchInfoManager 异步保存实体（幂等；报错仅 warn，不阻断启动）
	if _, err := asyncsave_entity.NewAsyncSaveEntity(crontabs.Pre1Minutes, e.MarchInfoManager.Tag()); err != nil {
		loggers.Logger.Warn("march async save entity register failed", zap.Error(err))
	}

	// 2. 从 DB 恢复行军（内部自动挂 MapAttribute + 推 TickerChan 重挂定时器 + 重建集结）
	marchList, err := e.MarchInfoManager.Init(dbconn.GetWriteDbConn())
	if err != nil {
		return err
	}

	// 3. 重做 AOI 注册，否则恢复的行军不在任何视野屏幕中，MapData 查不到
	for _, mi := range marchList {
		e.MapManager.MarchAOISetupSingle(mi)
	}

	loggers.Logger.Info("march recovery done", zap.Int("count", len(marchList)))
	return nil
}

// InitMapData 启动时从 DB 加载地图动态状态（稀疏覆盖：DB 行覆盖种子生成的底图）。
// 必须在 DB 初始化后、且 NewEngine 已完成底图生成后调用。
func (e *Engine) InitMapData() error {
	return e.MapDataManager.Load(dbconn.GetWriteDbConn())
}

// Stop 停止引擎
func (e *Engine) Stop() {
	// 停机前刷盘，尽量落净待保存的行军/地块（DB 在生命周期关闭期间仍可用）
	e.MarchInfoManager.SaveDo()
	e.MapDataManager.SaveDo()
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
	// 注意：不对行军本体拿写锁。行军互斥由上面的 LockMarchDo（marchDoLocker）保证，
	// 若再对 march RwLock 加写锁，Do 内部所有 getter（GetMarchState/GetTeam/...）的 RLock 会自锁死。
	// 仅锁目标地块，Do 里的 getter 可正常 RLock，召回时的 TryLock 也能成功。
	handle.Lock(false, false, toMapLock)
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

	// 队伍快照（战后存活兵力）：BACKARRIVED/ARRIVED 携带，供 game 写回 formation
	var teamInfo *pb_battle.TeamInfo
	if team := marchInfo.GetTeam(); team != nil {
		teamInfo = team.Format2Pb()
	}

	// 目标地块结算后状态（资源地产出快照同步用）：回城/召回事件不带（tile 状态不变）
	var tileLevel, tileElement int32
	if eventType == pb_redis_stream.MarchEventType_MARCH_EVENT_ARRIVED {
		if toInfo, ok := e.MapDataManager.GetMapInfo(toMapID); ok && toInfo != nil {
			tileLevel = int32(toInfo.GetLevel())
			tileElement = int32(toInfo.GetElementType())
		}
	}

	publishMarchEvent(e.ctx, &pb_redis_stream.MarchEvent{
		Type:         eventType,
		MarchId:      marchInfo.GetMarchID().Uint64(),
		RoleId:       marchInfo.GetFromRoleID(),
		ToMapId:      toMapID.Int32(),
		MarchType:    int32(marchInfo.MarchType),
		State:        int32(marchInfo.MarchState),
		Ts:           time.Now().Unix(),
		BattleResult: marchInfo.BattleResult, // 攻击行军携带战果（英雄经验）回传 game
		TeamInfo:     teamInfo,
		TileLevel:    tileLevel,
		TileElement:  tileElement,
		PrevOwner:    marchInfo.PrevOwnerID, // 攻占夺地原归属者（清理原主资源地快照）
	})
}

// OnTileReleased 地块被放弃：发布释放事件（game 侧移除资源地快照，停止产出）
func (e *Engine) OnTileReleased(roleID uint64, mapID cores_declarations.MapID) {
	loggers.Logger.Info("tile released",
		zap.Uint64("role_id", roleID),
		zap.Int32("map_id", mapID.Int32()))

	var tileElement int32
	if toInfo, ok := e.MapDataManager.GetMapInfo(mapID); ok && toInfo != nil {
		tileElement = int32(toInfo.GetElementType())
	}

	publishMarchEvent(e.ctx, &pb_redis_stream.MarchEvent{
		Type:        pb_redis_stream.MarchEventType_MARCH_EVENT_TILE_RELEASED,
		RoleId:      roleID,
		ToMapId:     mapID.Int32(),
		TileElement: tileElement,
		Ts:          time.Now().Unix(),
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
