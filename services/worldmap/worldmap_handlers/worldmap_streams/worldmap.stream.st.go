package worldmap_handlers

import (
	"server.slg.com/api/protocol/pb/pb_worldmap"
	worldmap_inits "server.slg.com/services/worldmap/worldmap_internals/worldmap_inits"
)

// WorldMapStreamHandler 供 main.go 注册 gRPC 服务
var WorldMapStreamHandler = &WorldMapStream{}

// WorldMapStream WorldMap 流处理，接收 game 的行军等请求
type WorldMapStream struct {
	pb_worldmap.UnimplementedWorldMapServiceServer
	engine *worldmap_inits.Engine
}

// SetEngine 注入 cores 引擎
func (s *WorldMapStream) SetEngine(e *worldmap_inits.Engine) {
	s.engine = e
}
