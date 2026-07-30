package worldmap_inits

import (
	"context"
	"time"

	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
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

	tickerChan := make(chan *marchs.MarchInfo, 1000)
	marchMgr := marchs.New(tickerChan, "march_info", mapConfig, cores_declarations.MarchTimeTypeStraight)

	e := &Engine{
		ctx:              ctx,
		Config:           mapConfig,
		MapDataManager:   mapData,
		MarchInfoManager: marchMgr,
	}

	marchDoFunc := func(mm *map_managers.MapManager, marchID cores_declarations.MarchID) {
		e.OnMarchArrived(mm, marchID)
	}
	marchDoHandleFunc := func(mm *map_managers.MapManager, info *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
		return nil
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

// OnMarchArrived 行军到达目标后的回调处理
func (e *Engine) OnMarchArrived(mm *map_managers.MapManager, marchID cores_declarations.MarchID) {
	marchInfo := e.MarchInfoManager.GetMarchInfo(marchID)
	if marchInfo == nil {
		loggers.Logger.Warn("march arrived but not found", zap.Uint64("march_id", marchID.Uint64()))
		return
	}

	_, toMapID, _ := marchInfo.GetMapIDs()
	loggers.Logger.Info("march arrived",
		zap.Uint64("march_id", marchID.Uint64()),
		zap.Uint64("role_id", marchInfo.GetFromRoleID()),
		zap.Int32("to", toMapID.Int32()),
		zap.Uint32("march_type", uint32(marchInfo.MarchType)))

	publishMarchEvent(e.ctx, &pb_redis_stream.MarchEvent{
		Type:      pb_redis_stream.MarchEventType_MARCH_EVENT_ARRIVED,
		MarchId:   marchID.Uint64(),
		RoleId:    marchInfo.GetFromRoleID(),
		ToMapId:   toMapID.Int32(),
		MarchType: int32(marchInfo.MarchType),
		Ts:        time.Now().Unix(),
	})

	_ = mm
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
