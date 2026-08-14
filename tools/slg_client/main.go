package main

// slg_client — 模拟 SLG 客户端，连 gateway 打通完整流程：
//
//	创建账号 → 登录 → 区服列表 → 进服建角 → 抽卡 → 上阵编队
//	→ 查主城坐标 → 开发出征（PvE）→ 等 worldmap 自动结算战斗
//
// 帧格式（与 common/conns/netconn/tcp_conn 对等）：
//
//	12 字节定长头（length/seq/msgID，BigEndian uint32）+ protobuf body
//	上行 body = 业务请求 proto；下行 body = pb_common.MessagePacket 信封
//
// 用法: go run ./tools/slg_client   （默认连 127.0.0.1:13001）

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/api/protocol/pb/pb_recruit"
	"server.slg.com/api/protocol/pb/pb_worldmap"
)

const (
	marchTypeDevelop = 10005 // MarchTypeDevelop：开发自己的地，到达触发 PvE 打守军
	marchTypeSweep    = 10003 // MarchTypeSweep：扫荡，到达有事件走事件分支，无事件打守军
)

func main() {
	sweepMode := false
	flag.BoolVar(&sweepMode, "sweep", false, "扫荡模式：目标选等级0格，MarchType=10003")
	flag.Parse()

	addr := "127.0.0.1:13001"
	if flag.NArg() > 0 {
		addr = flag.Arg(0)
	}

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Println("连接 gateway 失败:", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("== 已连接 gateway", addr)

	var seq uint32
	next := func() uint32 { seq++; return seq }

	acctName := fmt.Sprintf("slg_tester_%d", time.Now().Unix())
	password := "123456"

	// 1. 创建账号（已存在会报错，忽略后走登录）
	_ = call(conn, next(), uint32(pb_protocol.MsgID_LoginCreateAccount),
		&pb_account.CreateAccountReq{ChannelType: pb_account.ChannelType_Mine, AccountName: acctName, Password: password},
		nil)
	fmt.Println("[1] 创建账号:", acctName)

	// 2. 登录
	var accountID uint64
	var token string
	if err := call(conn, next(), uint32(pb_protocol.MsgID_LoginAccount),
		&pb_account.LoginAccountReq{ChannelType: pb_account.ChannelType_Mine, AccountName: acctName, Password: password},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_account.LoginAccountResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			accountID = resp.GetAccountId()
			token = resp.GetToken()
			fmt.Printf("[2] 登录成功 account_id=%d\n", accountID)
			return nil
		}); err != nil {
		fmt.Println("登录失败:", err)
		os.Exit(1)
	}

	// 3. 区服列表
	var serverID uint32
	if err := call(conn, next(), uint32(pb_protocol.MsgID_LoginServerList), &pb_account.ServerListReq{},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_account.ServerListResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			if len(resp.GetServers()) == 0 {
				return fmt.Errorf("无可用区服")
			}
			serverID = resp.GetServers()[0].GetServerId()
			fmt.Printf("[3] 区服列表, 选 server_id=%d (%s)\n", serverID, resp.GetServers()[0].GetServerName())
			return nil
		}); err != nil {
		fmt.Println("获取区服失败:", err)
		os.Exit(1)
	}

	// 4. 进服建角（role_id=0 新建；角色名唯一，带时间戳避免重复）
	var roleID uint64
	roleName := fmt.Sprintf("模拟侠%d", time.Now().Unix()%100000)
	if err := call(conn, next(), uint32(pb_protocol.MsgID_LoginEnterServer),
		&pb_account.EnterServerReq{AccountId: accountID, ServerId: serverID, RoleId: 0, RoleName: roleName, Token: token},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_account.EnterServerResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			roleID = resp.GetRoleId()
			fmt.Printf("[4] 进服建角成功 role_id=%d\n", roleID)
			return nil
		}); err != nil {
		fmt.Println("进服失败:", err)
		os.Exit(1)
	}

	// 5. 抽卡（新手池 1001 免费单抽）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameRecruit),
		&pb_recruit.RecruitReq{Id: 1001, Times: 1},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_recruit.RecruitResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			fmt.Printf("[5] 抽卡成功 rewards=%d\n", len(resp.GetRewards()))
			return nil
		}); err != nil {
		fmt.Println("抽卡失败:", err)
		os.Exit(1)
	}

	// 6. 英雄列表，取第一个英雄
	var heroID uint64
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameHeroList), &pb_hero.HeroListReq{},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_hero.HeroListResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			if len(resp.GetHeroes()) == 0 {
				return fmt.Errorf("无英雄")
			}
			heroID = resp.GetHeroes()[0].GetHeroId()
			fmt.Printf("[6] 英雄列表, 取 hero_id=%d (conf=%d)\n", heroID, resp.GetHeroes()[0].GetConfigId())
			return nil
		}); err != nil {
		fmt.Println("英雄列表失败:", err)
		os.Exit(1)
	}

	// 7. 编队列表（进服自动分配 1 个队列）
	var formationID, cityID uint64
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameFormationList),
		&pb_maps_march.FormationListReq{CityId: 0},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_maps_march.FormationListResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			if len(resp.GetFormations()) == 0 {
				return fmt.Errorf("无编队")
			}
			formationID = resp.GetFormations()[0].GetFormationId()
			cityID = resp.GetFormations()[0].GetCityId()
			fmt.Printf("[7] 编队列表 formation_id=%d city_id=%d\n", formationID, cityID)
			return nil
		}); err != nil {
		fmt.Println("编队列表失败:", err)
		os.Exit(1)
	}

	// 8. 上阵英雄到 1 号位（SlotPos=1 → HeroSlots[1] → worldmap SlotId=1 → CheckCanFight 通过）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameFormationField),
		&pb_maps_march.FormationFieldReq{CityId: cityID, FormationId: formationID, SlotPos: 1, HeroId: heroID, SoldierNum: 100},
		nil); err != nil {
		fmt.Println("上阵失败:", err)
		os.Exit(1)
	}
	fmt.Println("[8] 上阵英雄到 1 号位 成功")

	// 9. 查主城核心格坐标
	var coreMapID int32
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameBuildingList), &pb_city.BuildingListReq{},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_city.BuildingListResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			for _, b := range resp.GetBuildings() {
				if b.GetType() == pb_city.BuildingType_RoleMainCity {
					coreMapID = b.GetMapId()
					fmt.Printf("[9] 主城核心格 map_id=%d\n", coreMapID)
					return nil
				}
			}
			return fmt.Errorf("未找到主城")
		}); err != nil {
		fmt.Println("查主城失败:", err)
		os.Exit(1)
	}

	// 10. 在地图数据里找一块「等级0」的格作为目标，优先离主城近的（缩短行程）
	//    - 开发模式：须己方（可开发）；扫荡模式：任意（有守军即可，打守军PvE）
	coreX, coreY := coreMapID%1000, coreMapID/1000
	var toMapID int32
	bestDist := int32(1<<30)
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameMapData),
		&pb_worldmap.MapDataReq{MapId: coreMapID, Range: 2},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_worldmap.MapDataRsp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			for _, c := range resp.GetCells() {
				if c.GetLevel() != 0 || c.GetMapId() == coreMapID {
					continue
				}
				if !sweepMode && c.GetOwnerId() != roleID {
					continue // 开发模式须己方
				}
				cx, cy := c.GetMapId()%1000, c.GetMapId()/1000
				d := abs32(cx-coreX) + abs32(cy-coreY)
				if d < bestDist {
					bestDist = d
					toMapID = c.GetMapId()
				}
			}
			if toMapID == 0 {
				return fmt.Errorf("未找到等级0的格子")
			}
			fmt.Printf("[10] 找到目标格 map_id=%d 距主城%d格 (sweep=%v)\n", toMapID, bestDist, sweepMode)
			return nil
		}); err != nil {
		fmt.Println("找目标格失败:", err)
		os.Exit(1)
	}

	// 11. 出征：From=主城核心，To=目标格，MarchType=10005（开发）或 10003（扫荡）
	marchType := int32(marchTypeDevelop)
	if sweepMode {
		marchType = marchTypeSweep
	}
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameMarchCreate),
		&pb_maps_march.MarchCreateReq{FromMapId: coreMapID, ToMapId: toMapID, MarchType: marchType, FormationId: formationID},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_maps_march.MarchCreateResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			fmt.Printf("[11] 出征成功 march_id=%d end_time=%d\n", resp.GetMarchId(), resp.GetEndTime())
			return nil
		}); err != nil {
		fmt.Println("出征失败:", err)
		os.Exit(1)
	}

	// 12. 等待行军到达 + worldmap 自动结算战斗（相邻格耗时 ~10s）
	fmt.Println("[12] 等待行军到达并结算战斗（观察 worldmap/battle 日志）...")
	time.Sleep(15 * time.Second)
	fmt.Println("== 流程结束，请检查服务端日志与 battle_record/redis stream")
}

func abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// packFrame 组 TCP 帧：12 字节头 + body
func packFrame(seq, msgID uint32, body []byte) []byte {
	buf := make([]byte, 12+len(body))
	binary.BigEndian.PutUint32(buf[0:4], uint32(12+len(body)))
	binary.BigEndian.PutUint32(buf[4:8], seq)
	binary.BigEndian.PutUint32(buf[8:12], msgID)
	copy(buf[12:], body)
	return buf
}

// readFrame 读一帧：12 字节头 + length-12 字节 body
func readFrame(r io.Reader) (seq, msgID uint32, body []byte, err error) {
	header := make([]byte, 12)
	if _, err = io.ReadFull(r, header); err != nil {
		return
	}
	length := binary.BigEndian.Uint32(header[0:4])
	seq = binary.BigEndian.Uint32(header[4:8])
	msgID = binary.BigEndian.Uint32(header[8:12])
	if length < 12 {
		return 0, 0, nil, fmt.Errorf("非法帧长 %d", length)
	}
	body = make([]byte, length-12)
	if _, err = io.ReadFull(r, body); err != nil {
		return
	}
	return
}

// call 发送请求并读取对应 msgID 的响应信封（跳过 seq=0 的下推帧）
func call(conn net.Conn, seq, msgID uint32, req proto.Message, decode func(*pb_common.MessagePacket) error) error {
	body, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	if _, err = conn.Write(packFrame(seq, msgID, body)); err != nil {
		return err
	}

	for {
		rSeq, rMsgID, frameBody, err := readFrame(conn)
		if err != nil {
			return err
		}
		env := &pb_common.MessagePacket{}
		if err := proto.Unmarshal(frameBody, env); err != nil {
			return fmt.Errorf("解包信封: %w", err)
		}
		if rMsgID != msgID {
			// 服务端主动下推（seq=0）或其他帧，跳过
			fmt.Printf("  [skip] 收到下推 msgID=%d seq=%d\n", rMsgID, rSeq)
			continue
		}
		if env.GetErrCode() != 0 {
			return fmt.Errorf("msgID=%d errCode=%d devMsg=%q", msgID, env.GetErrCode(), env.GetDevMsg())
		}
		if decode != nil {
			return decode(env)
		}
		return nil
	}
}
