package game_handlers

import (
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/services/game/game_handlers/handler_servers"
)

// 协议注册 — 手工维护，后续可由 game_generates 自动生成
//
// 注册方式：在 init() 中调用 RegisterProto()
// 新协议三步：
//  1. protocol.proto 加 MsgID
//  2. handler_servers/ 加处理函数
//  3. 此文件追加 RegisterProto()

func init() {
	// ===== 英雄 (1000001~) =====
	RegisterProto(pb_protocol.MsgID_GameHeroList, &ProtoHandler{
		F:    Wrap(handler_servers.HandlerHeroList),
		Req:  &pb_hero.HeroListReq{},
		Resp: &pb_hero.HeroListResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroUpgradeLevel, &ProtoHandler{
		F:    Wrap(handler_servers.HandlerHeroUpgradeLevel),
		Req:  &pb_hero.HeroUpgradeLevelReq{},
		Resp: &pb_hero.HeroUpgradeLevelResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroCultivate, &ProtoHandler{
		F:    Wrap(handler_servers.HandlerHeroCultivate),
		Req:  &pb_hero.HeroCultivateReq{},
		Resp: &pb_hero.HeroCultivateResp{},
	})
}
