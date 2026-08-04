package battle_servers

// 进程内 gRPC 冒烟 — 不依赖 etcd/redis，sqlite 支撑存储。
// 验证 SaveBattleRecord → GetBattleRecord → ListBattleRecords（role/tile）全链路。

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/battle_record/battle_internals/battle_records"
)

func TestBattleRecordRPC(t *testing.T) {
	// SaveRecord 用雪花 ID 生成主键，需初始化（loggers 不依赖配置，snowflake 空配置建节点 0）
	loggers.Init()
	snowflakes.Init()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store := battle_records.New(db)
	if err := store.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	BattleRecordServerHandler.SetStore(store)

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb_battle_record.RegisterBattleRecordHandlerServer(srv, BattleRecordServerHandler)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	cli := pb_battle_record.NewBattleRecordHandlerClient(conn)

	// Save
	save, err := cli.SaveBattleRecord(ctx, &pb_battle_record.SaveBattleRecordReq{
		MarchId:          1,
		AttackerRoleId:   100,
		AttackerUnionId:  10,
		DefenderRoleIds:  []uint64{200},
		DefenderUnionIds: []uint64{20},
		MapId:            55,
		MarchType:        10001,
		AttackerWin:      true,
		IsOccupied:       true,
		Results:          &pb_battle.BattleResults{ResultCount: 1},
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if save.GetRecordId() == 0 {
		t.Fatalf("record_id 为空")
	}

	// Get
	got, err := cli.GetBattleRecord(ctx, &pb_battle_record.GetBattleRecordReq{RecordId: save.GetRecordId()})
	if err != nil || got.GetRecord() == nil {
		t.Fatalf("get: %v %+v", err, got)
	}
	if got.GetRecord().GetAttackerRoleId() != 100 {
		t.Fatalf("attacker_role_id 回读错误: %d", got.GetRecord().GetAttackerRoleId())
	}

	// List by defender role
	list, err := cli.ListBattleRecords(ctx, &pb_battle_record.ListBattleRecordsReq{
		TagType:  pb_battle_record.TagType_TAG_TYPE_ROLE,
		TagId:    200,
		Page:     1,
		PageSize: 20,
	})
	if err != nil || list.GetTotal() != 1 || len(list.GetRecords()) != 1 {
		t.Fatalf("list role: %v total=%d len=%d", err, list.GetTotal(), len(list.GetRecords()))
	}

	// List by tile
	if _, err := cli.ListBattleRecords(ctx, &pb_battle_record.ListBattleRecordsReq{
		TagType:  pb_battle_record.TagType_TAG_TYPE_TILE,
		TagId:    55,
		Page:     1,
		PageSize: 20,
	}); err != nil {
		t.Fatalf("list tile: %v", err)
	}
}
