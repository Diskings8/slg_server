package battle_servers

// 进程内 gRPC 冒烟 — 不依赖 etcd/redis，验证 BattleSettle 的
// proto 生成、handler 注册、序列化、battle_logics 全链路。

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_hero"
)

func TestBattleSettleRPC(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb_battle.RegisterBattleHandlerServer(srv, BattleServerHandler)
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
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	cli := pb_battle.NewBattleHandlerClient(conn)

	// 攻城：拆迁 150 > 耐久 100 → 占领
	rsp, err := cli.BattleSettle(context.Background(), &pb_battle.BattleSettleReq{
		AttackerTeam: &pb_battle.TeamInfo{
			SlotInfo: []*pb_battle.TeamSlotInfo{
				{
					SlotId:        0, // 大营
					MaxSoldierNum: 100,
					CurAliveNum:   100,
					AttackRange:   5,
					HeroInfo: &pb_hero.HeroInfo{
						ConfigId:       1,
						CurLevel:       1,
						CurStatus:      pb_hero.Status_Normal,
						AttrRelocation: &pb_cultivate.Cultivate{CurVal: 20, AddValCamp: 130}, // 基础拆20 + 加点130 = 150
					},
				},
			},
		},
		HasBuilding: true,
		BuildingHp:  100,
	})
	if err != nil {
		t.Fatalf("battle settle rpc failed: %v", err)
	}
	if !rsp.GetAttackerWin() {
		t.Fatalf("期望攻击方获胜")
	}
	if !rsp.GetOccupied() {
		t.Fatalf("期望占领（拆迁 150 > 耐久 100）")
	}
	if rsp.GetBuildingDamage() != 150 {
		t.Fatalf("期望建筑伤害 150，实际 %d", rsp.GetBuildingDamage())
	}
}
