package worldmap_client

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/loggers"
	"server.slg.com/services/game/game_internals/gate_stream"
)

// RoleStream 玩家到 worldmap 的双向视野流
type RoleStream struct {
	stream pb_worldmap.WorldMapService_StreamClient
	cancel context.CancelFunc
}

// ConnectRoleStream 玩家建立到 worldmap 的视野流
//
// 握手后启动下行转发 goroutine：worldmap 的下推（cores push）经此流转发给客户端。
// 已有连接时先关闭旧连接。
func (c *Client) ConnectRoleStream(ctx context.Context, roleID uint64, mapID int32) error {
	c.CloseRoleStream(roleID)

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := c.NewStream(streamCtx)
	if err != nil {
		cancel()
		loggers.Logger.Warn("worldmap stream connect failed", zap.Uint64("role_id", roleID), zap.Error(err))
		return err
	}

	// 握手：首条消息携带角色位置
	connectReq, err := proto.Marshal(&pb_worldmap.WorldMapConnectReq{MapId: mapID})
	if err != nil {
		cancel()
		return err
	}
	if err := stream.Send(&pb_common.NodePacket{
		RoleId: roleID,
		MsgId:  pb_protocol.MsgID_WorldMapConnect,
		Message: &pb_common.MessagePacket{
			Body: connectReq,
		},
	}); err != nil {
		cancel()
		loggers.Logger.Warn("worldmap stream handshake failed", zap.Uint64("role_id", roleID), zap.Error(err))
		return err
	}

	rs := &RoleStream{stream: stream, cancel: cancel}
	c.streamLock.Lock()
	c.streams[roleID] = rs
	c.streamLock.Unlock()

	// 下行转发：worldmap 下推 → gate_stream → gateway → client
	go c.forwardDown(streamCtx, roleID, stream)

	loggers.Logger.Info("worldmap stream connected", zap.Uint64("role_id", roleID), zap.Int32("map_id", mapID))
	return nil
}

// CloseRoleStream 关闭玩家到 worldmap 的视野流
func (c *Client) CloseRoleStream(roleID uint64) {
	c.streamLock.Lock()
	rs, ok := c.streams[roleID]
	if ok {
		delete(c.streams, roleID)
	}
	c.streamLock.Unlock()

	if ok && rs != nil {
		rs.cancel()
	}
}

// SendToWorldMap 上行转发：把客户端的地图相关消息转发到玩家的 worldmap 流
func (c *Client) SendToWorldMap(roleID uint64, packet *pb_common.NodePacket) error {
	c.streamLock.RLock()
	rs, ok := c.streams[roleID]
	c.streamLock.RUnlock()
	if !ok || rs == nil {
		return fmt.Errorf("worldmap stream not found for role %d", roleID)
	}
	return rs.stream.Send(packet)
}

// forwardDown 下行转发 goroutine
func (c *Client) forwardDown(ctx context.Context, roleID uint64, stream pb_worldmap.WorldMapService_StreamClient) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp, err := stream.Recv()
		if err != nil {
			// 流断开，清理连接
			loggers.Logger.Info("worldmap stream down",
				zap.Uint64("role_id", roleID), zap.Error(err))
			c.CloseRoleStream(roleID)
			return
		}

		// 原样转发 NodePacket 给客户端（gateway → client）
		if err := gate_stream.PushNodePacket(roleID, resp); err != nil {
			loggers.Logger.Warn("worldmap push to gate failed",
				zap.Uint64("role_id", roleID), zap.Error(err))
		}
	}
}
