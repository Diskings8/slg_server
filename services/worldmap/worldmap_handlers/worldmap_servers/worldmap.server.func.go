package worldmap_servers

import (
	"context"
	"time"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/marchs"
	"server.slg.com/services/worldmap/worldmap_internals/worldmap_inits"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var WorldMapServerHandler = &WorldMapServer{}

var marchIDCounter cores_declarations.MarchID

func init() {
	marchIDCounter = cores_declarations.MarchID(time.Now().UnixNano())
}

// nextMarchID 生成单调递增的行军 ID
func nextMarchID() cores_declarations.MarchID {
	marchIDCounter++
	return marchIDCounter
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

	if err := s.engine.March.CreateMarch(marchInfo); err != nil {
		loggers.Logger.Error("CreateMarch failed",
			zap.Uint64("march_id", marchID.Uint64()),
			zap.Error(err))
		return nil, status.Errorf(codes.Internal, "create march failed: %v", err)
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

	marchInfo := s.engine.March.GetMarchInfo(cores_declarations.MarchID(req.GetMarchId()))
	if marchInfo == nil {
		return nil, status.Error(codes.NotFound, "march not found")
	}

	if err := s.engine.March.DeleteMarch(marchInfo); err != nil {
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

	marchInfo := s.engine.March.GetMarchInfo(cores_declarations.MarchID(req.GetMarchId()))
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

// MapData 查询地图数据
func (s *WorldMapServer) MapData(ctx context.Context, req *pb_worldmap.MapDataReq) (*pb_worldmap.MapDataRsp, error) {
	loggers.Logger.Info("MapData",
		zap.Int32("map_id", req.GetMapId()),
		zap.Int32("range", req.GetRange()))

	if s.engine == nil {
		return nil, status.Error(codes.Internal, "engine not initialized")
	}

	// TODO: 返回地图格子数据
	return &pb_worldmap.MapDataRsp{}, nil
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
