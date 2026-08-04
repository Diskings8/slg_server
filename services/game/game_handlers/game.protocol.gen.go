package game_handlers

import (
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/services/game/game_handlers/game_streams/battle_record_handler"
	"server.slg.com/services/game/game_handlers/game_streams/building_handler"
	"server.slg.com/services/game/game_handlers/game_streams/formation_handler"
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
	RegisterProto(pb_protocol.MsgID_GameHeroUpgradeLevel, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroUpgradeLevel),
		Req:  &pb_hero.HeroUpgradeLevelReq{},
		Resp: &pb_hero.HeroUpgradeLevelResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroCultivate, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroCultivate),
		Req:  &pb_hero.HeroCultivateReq{},
		Resp: &pb_hero.HeroCultivateResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroSkillUpgrade, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroSkillUpgrade),
		Req:  &pb_hero.HeroSkillUpgradeReq{},
		Resp: &pb_hero.HeroSkillUpgradeResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroEquipSkill, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroEquipSkill),
		Req:  &pb_hero.HeroEquipSkillReq{},
		Resp: &pb_hero.HeroEquipSkillResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroUnequipSkill, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroUnequipSkill),
		Req:  &pb_hero.HeroUnequipSkillReq{},
		Resp: &pb_hero.HeroUnequipSkillResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroUpgradeStar, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroUpgradeStar),
		Req:  &pb_hero.HeroUpgradeStarReq{},
		Resp: &pb_hero.HeroUpgradeStarResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroTroopTransform, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroTroopTransform),
		Req:  &pb_hero.HeroTroopTransformReq{},
		Resp: &pb_hero.HeroTroopTransformResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameHeroTroopUnlock, &ProtoHandler{
		F:    Wrap(hero_handler.HandlerHeroTroopUnlock),
		Req:  &pb_hero.HeroTroopUnlockReq{},
		Resp: &pb_hero.HeroTroopUnlockResp{},
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

	// ===== 编队 (1000008~1000010) =====
	RegisterProto(pb_protocol.MsgID_GameFormationField, &ProtoHandler{
		F:    Wrap(formation_handler.HandlerFormationField),
		Req:  &pb_maps_march.FormationFieldReq{},
		Resp: &pb_maps_march.FormationFieldResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameFormationRemove, &ProtoHandler{
		F:    Wrap(formation_handler.HandlerFormationRemove),
		Req:  &pb_maps_march.FormationRemoveReq{},
		Resp: &pb_maps_march.FormationRemoveResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameFormationList, &ProtoHandler{
		F:    Wrap(formation_handler.HandlerFormationList),
		Req:  &pb_maps_march.FormationListReq{},
		Resp: &pb_maps_march.FormationListResp{},
	})

	// ===== 建筑 (1000011~1000012) =====
	RegisterProto(pb_protocol.MsgID_GameBuildingBuild, &ProtoHandler{
		F:    Wrap(building_handler.HandlerBuildingBuild),
		Req:  &pb_city.BuildingBuildReq{},
		Resp: &pb_city.BuildingBuildResp{},
	})
	RegisterProto(pb_protocol.MsgID_GameBuildingList, &ProtoHandler{
		F:    Wrap(building_handler.HandlerBuildingList),
		Req:  &pb_city.BuildingListReq{},
		Resp: &pb_city.BuildingListResp{},
	})

	// ===== 战报 (1000013) =====
	RegisterProto(pb_protocol.MsgID_GameBattleRecordList, &ProtoHandler{
		F:    Wrap(battle_record_handler.HandlerBattleRecordList),
		Req:  &pb_battle_record.ListBattleRecordsReq{},
		Resp: &pb_battle_record.ListBattleRecordsRsp{},
	})
}
