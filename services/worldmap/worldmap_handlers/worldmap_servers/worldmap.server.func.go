package worldmap_servers

import (
	"context"
	"time"

	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_aois"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_datas/map_events"
	"server.slg.com/services/internal/cores/marchs"
	"server.slg.com/services/worldmap/worldmap_internals/worldmap_inits"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var WorldMapServerHandler = &WorldMapServer{}

// nextMarchID 生成全局唯一行军 ID（雪花算法，跨重启/多实例稳定）。
// 依赖 worldmap 启动时 AsyncInit 中的 snowflakes.Init()（gRPC 服务在其后启动，无 nil 风险）。
func nextMarchID() cores_declarations.MarchID {
	return cores_declarations.MarchID(snowflakes.GenUUID())
}

// WorldMapServer Unary RPC 处理器，实现 WorldMapHandlerServer 接口
type WorldMapServer struct {
	pb_worldmap.UnimplementedWorldMapHandlerServer
	engine *worldmap_inits.Engine
}

// SetEngine 注入 cores 引擎
func (s *WorldMapServer) SetEngine(e *worldmap_inits.Engine) {
	s.engine = e
}

// CreateMarch 创建行军
func (s *WorldMapServer) CreateMarch(ctx context.Context, req *pb_worldmap.CreateMarchReq) (*pb_worldmap.CreateMarchRsp, error) {
	loggers.Logger.Info("CreateMarch",
		zap.Uint64("role_id", req.GetRoleId()),
		zap.Int32("from", req.GetFromMapId()),
		zap.Int32("to", req.GetToMapId()),
		zap.Int32("march_type", req.GetMarchType()))

	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}

	if req.GetRoleId() == 0 || req.GetFromMapId() == req.GetToMapId() {
		return nil, status.Error(codes.InvalidArgument, "invalid params")
	}

	nowTime := time.Now().Unix()

	// 计算行军耗时（基于基础速度）
	marchTimeSec := calcMarchTime(
		req.GetFromMapId(), req.GetToMapId(),
		req.GetBaseSpeed(), s.engine.Config,
	)

	marchID := nextMarchID()

	marchInfo := &marchs.MarchInfo{
		MarchID:         marchID,
		MarchType:       cores_declarations.MarchType(req.GetMarchType()),
		FromServerID:    req.GetServerId(),
		ToServerID:      req.GetToServerId(),
		FromRoleID:      req.GetRoleId(),
		ExecRoleID:      req.GetRoleId(),
		SrcFromMapID:    cores_declarations.MapID(req.GetFromMapId()),
		FromMapID:       cores_declarations.MapID(req.GetFromMapId()),
		ToMapID:         cores_declarations.MapID(req.GetToMapId()),
		MarchState:      pb_maps_march.MarchState_Move,
		StartTimeUx:     nowTime,
		EndTimeUx:       nowTime + marchTimeSec,
		UnionID:         req.GetUnionId(),
		BaseMarchSpeed:  req.GetBaseSpeed(),
		FinalMarchSpeed: req.GetBaseSpeed(),
		Path: []cores_declarations.MapID{
			cores_declarations.MapID(req.GetFromMapId()),
			cores_declarations.MapID(req.GetToMapId()),
		},
	}

	// 填充出征队伍（英雄 + 士兵），供到达后的战斗/采集结算使用
	if teamSlots := req.GetTeamSlots(); len(teamSlots) > 0 {
		marchInfo.Team = &marchs.Team{Slots: teamSlots}
	}

	// 无队伍数据时给空队伍兜底，避免结算时 nil 指针
	if marchInfo.Team == nil {
		marchInfo.Team = &marchs.Team{Slots: []*pb_battle.TeamSlotInfo{}}
	}

	// 扫荡行军：出发前记录目标地块的事件ID（到达时校对；无事件则走守军PvE）
	if marchInfo.MarchType == cores_declarations.MarchTypeSweep {
		if info, ok := s.engine.MapDataManager.GetMapInfo(marchInfo.ToMapID); ok {
			if ev := info.GetOverlayEvent(); ev != nil {
				marchInfo.TargetEventID = ev.EventID
			}
		}
	}

	// 走 MarchHandler.CreateMarch：严格校验（队伍可战斗/地块存在/攻击目标合法/开发归属）
	// → 持久化 → 挂 MapAttribute → AOI 注册 → 视野推送。此前直接调 MarchInfoManager.CreateMarch，
	// 既绕过校验层，也缺 AOI 注册（新行军不出现在任何视野屏幕）。
	if err := s.engine.MarchHandler.CreateMarch(marchInfo); err != nil {
		loggers.Logger.Error("CreateMarch failed",
			zap.Uint64("march_id", marchID.Uint64()),
			zap.Error(err))
		// MarchHandler.CreateMarch 的返回均来自校验层（空/不可战斗队伍、目标非法等），映射为参数错误
		return nil, status.Errorf(codes.InvalidArgument, "create march failed: %v", err)
	}

	loggers.Logger.Info("march created",
		zap.Uint64("march_id", marchID.Uint64()),
		zap.Int64("end_time", marchInfo.EndTimeUx))

	return &pb_worldmap.CreateMarchRsp{
		MarchId: marchID.Uint64(),
		EndTime: marchInfo.EndTimeUx,
	}, nil
}

// CancelMarch 取消行军
func (s *WorldMapServer) CancelMarch(ctx context.Context, req *pb_worldmap.CancelMarchReq) (*pb_worldmap.CancelMarchRsp, error) {
	loggers.Logger.Info("CancelMarch",
		zap.Uint64("role_id", req.GetRoleId()),
		zap.Uint64("march_id", req.GetMarchId()))

	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}

	marchInfo := s.engine.MarchInfoManager.GetMarchInfo(cores_declarations.MarchID(req.GetMarchId()))
	if marchInfo == nil {
		return nil, status.Error(codes.NotFound, "march not found")
	}

	if err := s.engine.MarchInfoManager.DeleteMarch(marchInfo); err != nil {
		loggers.Logger.Error("CancelMarch failed",
			zap.Uint64("march_id", req.GetMarchId()),
			zap.Error(err))
		return nil, status.Errorf(codes.Internal, "cancel march failed: %v", err)
	}

	// 通知 game 行军取消
	s.engine.OnMarchCanceled(marchInfo)

	return &pb_worldmap.CancelMarchRsp{}, nil
}

// MarchInfo 查询行军信息
func (s *WorldMapServer) MarchInfo(ctx context.Context, req *pb_worldmap.MarchInfoReq) (*pb_worldmap.MarchInfoRsp, error) {
	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}

	marchInfo := s.engine.MarchInfoManager.GetMarchInfo(cores_declarations.MarchID(req.GetMarchId()))
	if marchInfo == nil {
		return nil, status.Error(codes.NotFound, "march not found")
	}

	fromMapID, toMapID, _ := marchInfo.GetMapIDs()
	startTime, endTime := marchInfo.GetMarchStartAndEndTimeUx()

	return &pb_worldmap.MarchInfoRsp{
		FromMapId: fromMapID.Int32(),
		ToMapId:   toMapID.Int32(),
		MarchType: int32(marchInfo.MarchType),
		State:     marchInfo.MarchState,
		StartTime: startTime,
		EndTime:   endTime,
		RoleId:    marchInfo.GetFromRoleID(),
		UnionId:   marchInfo.GetUnionID(),
	}, nil
}

// MapData 查询视野内地图数据（AOI 九宫格）
func (s *WorldMapServer) MapData(ctx context.Context, req *pb_worldmap.MapDataReq) (*pb_worldmap.MapDataRsp, error) {
	loggers.Logger.Info("MapData",
		zap.Int32("map_id", req.GetMapId()),
		zap.Int32("range", req.GetRange()))

	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}

	mapID := cores_declarations.MapID(req.GetMapId())
	if mapID.IsInvalid() {
		return nil, status.Error(codes.InvalidArgument, "invalid map_id")
	}

	// 获取视野屏幕块：range<=0 用九宫格，否则用 cover 范围
	var screens []*map_aois.Screen[cores_declarations.ScreenID]
	if req.GetRange() <= 0 {
		screens = s.engine.MapDataManager.AOI.Around(mapID)
	} else {
		screens = s.engine.MapDataManager.AOI.Cover(mapID, req.GetRange())
	}

	resp := &pb_worldmap.MapDataRsp{}

	// 去重集合
	seenCell := make(map[cores_declarations.MapID]struct{})
	seenMarch := make(map[cores_declarations.MarchID]struct{})

	for _, screen := range screens {
		if screen == nil {
			continue
		}

		// 非空地地块
		screen.MapDataRange(func(cellMapID cores_declarations.MapID) bool {
			if _, ok := seenCell[cellMapID]; ok {
				return true
			}
			seenCell[cellMapID] = struct{}{}

			if info, ok := s.engine.MapDataManager.GetMapInfo(cellMapID); ok {
				resp.Cells = append(resp.Cells, buildMapCellInfo(info))
			}
			return true
		})

		// 视野内行军
		screen.MarchRange(func(info cores_declarations.MarchInfoI) bool {
			march, ok := info.(*marchs.MarchInfo)
			if !ok || march == nil {
				return true
			}
			if _, ok := seenMarch[march.GetMarchID()]; ok {
				return true
			}
			seenMarch[march.GetMarchID()] = struct{}{}

			resp.Marches = append(resp.Marches, buildMarchBrief(march))
			return true
		})
	}

	return resp, nil
}

// CreateRole 创建角色主城（分配出生点并落主城），返回主城核心 MapID
func (s *WorldMapServer) CreateRole(ctx context.Context, req *pb_worldmap.CreateRoleReq) (*pb_worldmap.CreateRoleRsp, error) {
	loggers.Logger.Info("CreateRole",
		zap.Uint64("role_id", req.GetRoleId()),
		zap.Uint32("server_id", req.GetServerId()))

	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}
	if req.GetRoleId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid params")
	}

	// 构造最小 RoleBrief（cores 仅使用 server_id / role_id）
	roleBrief := &pb_role.RoleBrief{
		RoleBaseInfo: &pb_role.RoleBaseInfo{
			SimpleInfo: &pb_role.RoleSimpleInfo{
				ServerId: req.GetServerId(),
				RoleId:   req.GetRoleId(),
				RoleName: req.GetRoleName(),
			},
		},
	}

	coreMapID, err := s.engine.MapManager.CreateRole(roleBrief)
	if err != nil {
		loggers.Logger.Error("CreateRole failed",
			zap.Uint64("role_id", req.GetRoleId()),
			zap.Error(err))
		return nil, status.Errorf(codes.Internal, "create role main city failed: %v", err)
	}

	return &pb_worldmap.CreateRoleRsp{MapId: coreMapID.Int32()}, nil
}

// SpawnReviewEvent 审查任务刷事件：在主城 5×5 外圈（12 格）找非建筑、非已有事件地块刷出 OverlayEvent
func (s *WorldMapServer) SpawnReviewEvent(ctx context.Context, req *pb_worldmap.SpawnReviewEventReq) (*pb_worldmap.SpawnReviewEventRsp, error) {
	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}
	core := cores_declarations.MapID(req.GetCoreMapId())
	if core.IsInvalid() {
		return nil, status.Error(codes.InvalidArgument, "invalid core_map_id")
	}

	scope := s.engine.Config.MapScope()
	cx, cy := s.engine.Config.MapID2XY(core)
	eventType := map_events.EventType(req.GetEventType())

	// 5×5 外圈：|dx|,|dy| ≤ 2 且非内圈 3×3（主城建筑格）
	var spawned int32
	for dy := int32(-2); dy <= 2 && spawned == 0; dy++ {
		for dx := int32(-2); dx <= 2; dx++ {
			if abs(dx) <= 1 && abs(dy) <= 1 {
				continue // 内圈 3×3 = 主城建筑格，跳过
			}
			tx, ty := cx+dx, cy+dy
			if tx < 0 || tx >= scope || ty < 0 || ty >= scope {
				continue // 越界
			}
			mid := s.engine.Config.XY2MapID(tx, ty)
			info, ok := s.engine.MapDataManager.GetMapInfo(mid)
			if !ok || info == nil {
				continue
			}
			if info.GetOverlayBuilding() != nil || info.GetOverlayEvent() != nil {
				continue // 建筑格 / 已有事件
			}

			info.SetOverlayEvent(&map_events.OverlayEvent{
				EventID:     snowflakes.GenUUID(),
				EventType:   eventType,
				Interaction: eventType.Interaction(), // 寻宝/采集=点击；打怪=行军
				Progress:    0,
				ExpireTime:  time.Now().Add(time.Hour).Unix(), // 事件有效期 1 小时
			})
			spawned = int32(mid)
		}
	}

	loggers.Logger.Info("review event spawned",
		zap.Uint64("role_id", req.GetRoleId()),
		zap.Int32("core_map_id", int32(core)),
		zap.Int32("map_id", spawned),
		zap.Int32("event_type", int32(eventType)))
	return &pb_worldmap.SpawnReviewEventRsp{MapId: spawned}, nil
}

// EventClick 气泡点击事件（采集/寻宝）：每次 +进度，超 100% 完成并清除事件
func (s *WorldMapServer) EventClick(ctx context.Context, req *pb_worldmap.EventClickReq) (*pb_worldmap.EventClickRsp, error) {
	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}
	mid := cores_declarations.MapID(req.GetMapId())
	if mid.IsInvalid() {
		return nil, status.Error(codes.InvalidArgument, "invalid map_id")
	}
	info, ok := s.engine.MapDataManager.GetMapInfo(mid)
	if !ok || info == nil {
		return nil, status.Error(codes.InvalidArgument, "地块不存在")
	}
	ev := info.GetOverlayEvent()
	if ev == nil {
		return nil, status.Error(codes.InvalidArgument, "该地块无事件")
	}
	if ev.Interaction != map_events.EventInteractionClick {
		return nil, status.Error(codes.InvalidArgument, "该事件需行军处理")
	}

	ev.Progress += map_events.EventClickProgressStep
	completed := ev.Progress > 100
	if completed {
		info.SetOverlayEvent(nil) // 完成清除事件
	}
	loggers.Logger.Info("review event click",
		zap.Uint64("role_id", req.GetRoleId()),
		zap.Int32("map_id", req.GetMapId()),
		zap.Int32("progress", ev.Progress),
		zap.Bool("completed", completed))
	return &pb_worldmap.EventClickRsp{Progress: ev.Progress, Completed: completed, EventType: int32(ev.EventType)}, nil
}

// buildMapCellInfo 组装地块数据
func buildMapCellInfo(info *map_datas.MapInfo) *pb_worldmap.MapCellInfo {
	return &pb_worldmap.MapCellInfo{
		MapId:       info.GetMapID().Int32(),
		ElementType: int32(info.GetElementType()),
		ElementId:   info.GetElementID(),
		Level:       int32(info.GetLevel()),
		OwnerId:     info.GetOwnerID(),
		ServerId:    info.GetServerID(),
	}
}

// buildMarchBrief 组装行军简况
func buildMarchBrief(march *marchs.MarchInfo) *pb_worldmap.MarchBrief {
	fromMapID, toMapID, _ := march.GetMapIDs()
	_, endTime := march.GetMarchStartAndEndTimeUx()

	return &pb_worldmap.MarchBrief{
		MarchId:     march.GetMarchID().Uint64(),
		FromMapId:   fromMapID.Int32(),
		ToMapId:     toMapID.Int32(),
		FromRoleId:  march.GetFromRoleID(),
		MarchType:   int32(march.MarchType),
		State:       march.MarchState,
		EndTime:     endTime,
	}
}

// calcMarchTime 计算行军耗时（秒）
func calcMarchTime(fromMapID, toMapID int32, baseSpeed uint32, config *worldmap_inits.DefaultMapConfig) int64 {
	if baseSpeed == 0 {
		baseSpeed = 100 // default speed
	}

	fx, fy := config.MapID2XY(cores_declarations.MapID(fromMapID))
	tx, ty := config.MapID2XY(cores_declarations.MapID(toMapID))

	// 曼哈顿距离
	dist := int32(abs(fx-tx) + abs(fy-ty))
	if dist <= 0 {
		dist = 1
	}

	// 耗时 = 距离 * 1000 / 速度
	return int64(int64(dist) * 1000 / int64(baseSpeed))
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
