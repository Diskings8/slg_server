package battle_servers

import (
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/services/battle_record/battle_internals/battle_records"
)

// BattleRecordServer 战斗记录 RPC 服务（Save/Get/List）
var BattleRecordServerHandler = &BattleRecordServer{}

type BattleRecordServer struct {
	pb_battle_record.UnimplementedBattleRecordHandlerServer
	store *battle_records.Store
}

// SetStore 注入战报存储（main.go AsyncInit 时调用）
func (s *BattleRecordServer) SetStore(store *battle_records.Store) {
	s.store = store
}
