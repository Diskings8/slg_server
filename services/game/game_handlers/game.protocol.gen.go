package game_handlers

import (
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/services/game/game_handlers/game_streams/hero_handler"
	"server.slg.com/services/game/game_handlers/game_streams/item_handler"
	"server.slg.com/services/game/game_handlers/game_streams/map_handler"
	"server.slg.com/services/game/game_handlers/game_streams/march_handler"
)

// 协议注册 — 手工维护，后续可由 game_generates 自动生成
//
// 注册方式：在 init() 中调用 RegisterProto()
// 新协议三步：
//  1. protocol.proto 加 MsgID
//  2. worldmap_handlers 包加处理函数
//  3. 此文件追加 RegisterProto()

func init() {
	// ===== 英雄 (1000001~) =====
	RegisterProto(pb_protocol.MsgID_GameHeroList, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroList),
		Req:  &pb_hero.HeroListReq{},
		Resp: &pb_hero.HeroListResp{},
	})

	// ===== 道具 (1000005) =====
	RegisterProto(pb_protocol.MsgID_GameUseItem, &ProtoHandler{
		F:    Wrap(item_handler.HandlerUseItem),
		Req:  &pb_item.UseItemReq{},
		Resp: &pb_item.UseItemResp{},
	})

	// ===== 出征 (1000006) =====
	RegisterProto(pb_protocol.MsgID_GameMarchCreate, &ProtoHandler{
		F:    Wrap(march_handler.HandlerMarchCreate),
		Req:  &pb_maps_march.MarchCreateReq{},
		Resp: &pb_maps_march.MarchCreateResp{},
	})

	// ===== 视野地图 (1000007) =====
	RegisterProto(pb_protocol.MsgID_GameMapData, &ProtoHandler{
		F:    Wrap(map_handler.HandlerMapData),
		Req:  &pb_worldmap.MapDataReq{},
		Resp: &pb_worldmap.MapDataRsp{},
	})
}
