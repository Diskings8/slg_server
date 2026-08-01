package worldmap_handlers

import (
	"server.slg.com/api/protocol/pb/pb_camera"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/conns/rpcconn/rpc_declarations"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/marchs"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Stream game→worldmap 玩家视野流
//
// 流程：
//  1. 首条消息为握手（MsgID_WorldMapConnect，body=WorldMapConnectReq），解析 roleID + 玩家所在地图ID
//  2. 注册到 RoleConnectManager（cores 的 AOI 连接），后续 cores push 走该流下推
//  3. 后台 receiveF 循环处理相机移动等上行消息
//  4. 连接断开时清理
func (s *WorldMapStream) Stream(stream pb_worldmap.WorldMapService_StreamServer) error {
	if s.engine == nil {
		return status.Error(codes.Internal, "engine not initialized")
	}

	// 1. 握手
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	roleID := req.GetRoleId()
	if roleID < 1 {
		return status.Error(codes.InvalidArgument, "invalid role_id")
	}
	connectReq := &pb_worldmap.WorldMapConnectReq{}
	if err := proto.Unmarshal(req.GetMessage().GetBody(), connectReq); err != nil {
		return status.Errorf(codes.InvalidArgument, "unmarshal connect req: %v", err)
	}

	// 2. 注册到 RoleConnectManager（内部启动 receiveF loop 处理上行）
	conn, err := s.engine.MapManager.GetRoleConnectManager().NewRoleConnect(
		rpc_declarations.RpcStreamGame2WorldMap,
		roleID,
		cores_declarations.MapID(connectReq.GetMapId()),
		stream,
		s.recv,
	)
	if err != nil {
		loggers.Logger.Warn("worldmap NewRoleConnect failed",
			zap.Uint64("role_id", roleID), zap.Error(err))
		return err
	}

	loggers.Logger.Info("worldmap connect",
		zap.Uint64("role_id", roleID),
		zap.Int32("map_id", connectReq.GetMapId()))

	// 3. 阻塞等待连接断开（ctx/stream 断开时返回）
	conn.GetStream().WaitDone()
	return nil
}

// recv 处理流上的上行消息（相机移动等）
func (s *WorldMapStream) recv(stream grpc.ServerStream) error {
	req := &pb_common.NodePacket{}
	if err := stream.RecvMsg(req); err != nil {
		return err
	}

	roleID := req.GetRoleId()
	switch req.GetMsgId() {
	case pb_protocol.MsgID_GameCameraMove:
		// 相机移动 → 更新角色视野位置 + 返回视野数据
		moveReq := &pb_camera.CameraMoveReq{}
		if err := proto.Unmarshal(req.GetMessage().GetBody(), moveReq); err != nil {
			return nil
		}
		mapID := s.engine.Config.XY2MapID(moveReq.GetX(), moveReq.GetY())
		s.engine.MapManager.GetRoleConnectManager().SetRoleScreen(roleID, mapID)

		resp := s.buildCameraMoveResp(mapID)
		s.engine.MapManager.GetRoleConnectManager().PushToRoleID(
			pb_protocol.MsgID_GameCameraMove, resp, roleID)
	}

	return nil
}

// buildCameraMoveResp 组装相机移动后的视野数据（九宫格内地块 + 行军）
func (s *WorldMapStream) buildCameraMoveResp(mapID cores_declarations.MapID) *pb_camera.CameraMoveResp {
	resp := &pb_camera.CameraMoveResp{}
	var mapInfos []*pb_camera.MapInfo

	seenCell := make(map[cores_declarations.MapID]struct{})
	seenMarch := make(map[cores_declarations.MarchID]struct{})

	for _, screen := range s.engine.MapDataManager.AOI.Around(mapID) {
		if screen == nil {
			continue
		}

		// 视野内地块
		var mapIDs []cores_declarations.MapID
		screen.MapDataRange(func(id cores_declarations.MapID) bool {
			if _, ok := seenCell[id]; ok {
				return true
			}
			seenCell[id] = struct{}{}
			mapIDs = append(mapIDs, id)
			return true
		})
		s.engine.MapManager.FormatMapInfo2Pb(
			s.engine.MapDataManager.GetMapInfoSlice(mapIDs), &mapInfos)

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
			resp.March = append(resp.March, s.engine.MapManager.FormatMarchInfo2Pb(march))
			return true
		})
	}

	resp.Map = mapInfos
	return resp
}
