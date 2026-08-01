package worldmap_inits

import (
	"context"
	"time"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos/march_factory"
	"server.slg.com/services/internal/cores/marchs"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// marchStreamKey Redis Stream key
const marchStreamKey = "slg:march:events"

// Engine cores 引擎聚合
type Engine struct {
	ctx              context.Context
	Config           *DefaultMapConfig
	MapDataManager   *map_datas.MapDataManager
	MarchInfoManager *marchs.MarchInfoManager
	MapManager       *map_managers.MapManager
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
	return e
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

	publishMarchEvent(e.ctx, &pb_redis_stream.MarchEvent{
		Type:      pb_redis_stream.MarchEventType_MARCH_EVENT_ARRIVED,
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

	cache := cacheconn.Get()
	type streamPublishI interface {
		XAdd(ctx context.Context, a *redis.XAddArgs) *redis.StringCmd
	}
	pub, ok := cache.(streamPublishI)
	if !ok {
		loggers.Logger.Warn("cache does not support XADD")
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := pub.XAdd(ctxWithTimeout, &redis.XAddArgs{
		Stream: marchStreamKey,
		Values: []string{"data", string(data)},
	}).Err(); err != nil {
		loggers.Logger.Warn("publish march event to redis stream failed",
			zap.String("event_type", event.Type.String()),
			zap.Uint64("march_id", event.MarchId),
			zap.Error(err))
	}
}
