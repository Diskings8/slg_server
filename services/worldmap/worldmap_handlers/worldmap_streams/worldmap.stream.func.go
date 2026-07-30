package worldmap_handlers

import (
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/loggers"

	"go.uber.org/zap"
)

// Stream game 连接 worldmap 的双向流
// 用于实时数据推送（相机移动、地块更新等），业务请求走 Unary RPC
func (s *WorldMapStream) Stream(stream pb_worldmap.WorldMapService_StreamServer) error {
	loggers.Logger.Info("worldmap stream connected")

	for {
		req, err := stream.Recv()
		if err != nil {
			loggers.Logger.Info("worldmap stream disconnected", zap.Error(err))
			return err
		}

		// TODO: 转发实时数据（相机移动等）
		_ = req
	}
}
