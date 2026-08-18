package main

// slg_client — 模拟 SLG 客户端，连 gateway 打通完整流程：
//
//	创建账号 → 登录 → 区服列表 → 进服建角 → 抽卡 → 上阵编队
//	→ 查主城坐标 → 攻占中立资源地（打守军→占领）→ 开发升级（+3）
//	→ 查资源余额（惰性产出）→ 放弃地块
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
	"server.slg.com/api/protocol/pb/pb_gm"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/api/protocol/pb/pb_recruit"
	"server.slg.com/api/protocol/pb/pb_review"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/api/protocol/pb_confs"
)

const (
	marchTypeAttack   = 10001 // MarchTypeAttack：攻占（打守军 PvE → 胜利占领）
	marchTypeDevelop  = 10005 // MarchTypeDevelop：开发自己的地（lv2~4 → +3）
	marchTypeSweep    = 10003 // MarchTypeSweep：扫荡（保留，未在本流程使用）
	gmHeroLevel       = 20    // GM 演示：英雄升到该等级（保证攻占能打赢守军）
)

func main() {
	reviewMode := false
	farMode := false
	flag.BoolVar(&reviewMode, "review", false, "审查模式：触发审查 + 选择任务刷事件")
	flag.BoolVar(&farMode, "far", false, "崩溃恢复测试：选最远中立资源地（行军长时间在途，供杀服务验证恢复）")
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

	// 8. 上阵英雄到大营（SlotPos=1 → 编队 slot 1 → 出征 SlotId=1 大营 → CheckCanFight 通过）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameFormationField),
		&pb_maps_march.FormationFieldReq{CityId: cityID, FormationId: formationID, SlotPos: 1, HeroId: heroID, SoldierNum: 100},
		nil); err != nil {
		fmt.Println("上阵失败:", err)
		os.Exit(1)
	}
	fmt.Println("[8] 上阵英雄到 1 号位 成功")

	// 8.5 GM：英雄升到 20 级（演示 GM 指令 + 保证能打赢守军完成攻占）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameGm),
		&pb_gm.GmReq{Cmd: "hero.set_level", Args: []string{fmt.Sprintf("%d", heroID), fmt.Sprintf("%d", gmHeroLevel)}},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_gm.GmResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			fmt.Printf("[8.5] GM: %s\n", resp.GetMsg())
			return nil
		}); err != nil {
		fmt.Println("GM 升级失败:", err)
		os.Exit(1)
	}

	// 8.6 补满兵力（等级 20 无兵营上限 350），保证攻占能打赢守军
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameFormationField),
		&pb_maps_march.FormationFieldReq{CityId: cityID, FormationId: formationID, SlotPos: 1, HeroId: heroID, SoldierNum: 350},
		nil); err != nil {
		fmt.Println("补兵失败:", err)
		os.Exit(1)
	}
	fmt.Println("[8.6] 补满兵力 350 成功")

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

	// 10. 在地图数据里找一块中立资源地（无主、lv2~4、资源元素）作为攻占目标
	coreX, coreY := coreMapID%1000, coreMapID/1000
	var toMapID, targetDist int32
	bestDist := int32(-1) // far 模式：最远距离
	var lowID, lowDist, highID, highDist int32 // 非 far：lv2-3 近处优先，lv4 兜底
	lowDist, highDist = 1<<30, 1<<30
	searchRange := int32(3)
	if farMode {
		searchRange = 8 // -far：扩大查询范围找远目标（Range=Screen 粒度，响应受 gRPC 4MB 限制）
	}
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameMapData),
		&pb_worldmap.MapDataReq{MapId: coreMapID, Range: searchRange},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_worldmap.MapDataRsp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			for _, c := range resp.GetCells() {
				if c.GetMapId() == coreMapID || c.GetOwnerId() != 0 {
					continue
				}
				if c.GetElementType() < 1 || c.GetElementType() > 4 {
					continue // 非资源元素（地形）
				}
				if c.GetLevel() < 2 || c.GetLevel() > 4 {
					continue // 可攻占 + 可开发区间
				}
				cx, cy := c.GetMapId()%1000, c.GetMapId()/1000
				d := abs32(cx-coreX) + abs32(cy-coreY)
				if farMode {
					if d > bestDist {
						bestDist = d
						toMapID = c.GetMapId()
						targetDist = d
					}
					continue
				}
				// 非 far：lv2-3（守军弱）近处优先，lv4 兜底
				if c.GetLevel() >= 4 {
					if d < highDist {
						highDist = d
						highID = c.GetMapId()
					}
				} else if d < lowDist {
					lowDist = d
					lowID = c.GetMapId()
				}
			}
			if farMode {
				if toMapID == 0 {
					return fmt.Errorf("附近未找到中立资源地（无主 lv2~4 资源元素）")
				}
			} else {
				if lowID != 0 {
					toMapID, targetDist = lowID, lowDist
				} else if highID != 0 {
					toMapID, targetDist = highID, highDist
				} else {
					return fmt.Errorf("附近未找到中立资源地（无主 lv2~4 资源元素）")
				}
			}
			fmt.Printf("[10] 找到中立资源地 map_id=%d 距主城%d格 (far=%v)\n", toMapID, targetDist, farMode)
			return nil
		}); err != nil {
		fmt.Println("找目标资源地失败:", err)
		os.Exit(1)
	}

	// 行军耗时 ≈ 距离×1000/速度(100) 秒，加缓冲
	marchWait := time.Duration(targetDist*10+8) * time.Second

	// 11. 攻占：MarchType=10001（攻击，打守军 PvE → 胜利占领空地）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameMarchCreate),
		&pb_maps_march.MarchCreateReq{FromMapId: coreMapID, ToMapId: toMapID, MarchType: int32(marchTypeAttack), FormationId: formationID},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_maps_march.MarchCreateResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			fmt.Printf("[11] 攻占出征 march_id=%d end_time=%d\n", resp.GetMarchId(), resp.GetEndTime())
			return nil
		}); err != nil {
		fmt.Println("攻占出征失败:", err)
		os.Exit(1)
	}

	// 12. 等待攻占行军到达 + 战斗 + 占领（观察 worldmap/battle 日志）
	fmt.Printf("[12] 等待攻占行军到达并结算（约 %s）...\n", marchWait)
	time.Sleep(marchWait)

	// 13. 复查目标格归属
	occupied := false
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameMapData),
		&pb_worldmap.MapDataReq{MapId: toMapID, Range: 0},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_worldmap.MapDataRsp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			for _, c := range resp.GetCells() {
				if c.GetMapId() == toMapID && c.GetOwnerId() == roleID {
					occupied = true
				}
			}
			return nil
		}); err != nil {
		fmt.Println("复查目标格失败:", err)
		os.Exit(1)
	}
	if occupied {
		fmt.Println("[13] 攻占成功！地块已归己，开始产出")
	} else {
		fmt.Println("[13] 攻占可能未生效（战败/时间不足），后续开发可能失败，请检查 worldmap 日志")
	}

	// 14. 开发：MarchType=10005（自有 lv2~4 → +3 级，提升产量）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameMarchCreate),
		&pb_maps_march.MarchCreateReq{FromMapId: coreMapID, ToMapId: toMapID, MarchType: int32(marchTypeDevelop), FormationId: formationID},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_maps_march.MarchCreateResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			fmt.Printf("[14] 开发出征 march_id=%d end_time=%d\n", resp.GetMarchId(), resp.GetEndTime())
			return nil
		}); err != nil {
		fmt.Println("开发出征失败:", err)
		os.Exit(1)
	}
	fmt.Printf("[14] 等待开发行军到达（地块升级 +3，约 %s）...\n", marchWait)
	time.Sleep(marchWait)

	// 15. 查询资源余额（先惰性结算产出，看经济闭环）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameItemList), &pb_item.ItemListReq{},
		func(env *pb_common.MessagePacket) error {
			resp := &pb_item.ItemListResp{}
			if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
				return err
			}
			fmt.Printf("[15] 资源余额: %s\n", formatResources(resp.GetItems()))
			return nil
		}); err != nil {
		fmt.Println("查背包失败:", err)
	}

	// 16. 放弃地块（释放归属，停止产出）
	if err := call(conn, next(), uint32(pb_protocol.MsgID_GameTileAbandon),
		&pb_worldmap.AbandonTileReq{MapId: toMapID},
		nil); err != nil {
		fmt.Println("放弃地块失败:", err)
		os.Exit(1)
	}
	fmt.Println("[16] 放弃地块成功（观察 worldmap 释放事件 / game 快照移除）")

	// 13. 审查玩法（-review）：触发审查 + 选择任务刷事件
	if reviewMode {
		var tasks []*pb_review.ReviewTaskInfo
		if err := call(conn, next(), uint32(pb_protocol.MsgID_GameReviewStart), &pb_review.ReviewStartReq{},
			func(env *pb_common.MessagePacket) error {
				resp := &pb_review.ReviewStartResp{}
				if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
					return err
				}
				fmt.Printf("[R1] 审查开始 chances=%d exp=%d level=%d expGained=%d leveledUp=%v tasks=%d\n",
					resp.Chances, resp.Exp, resp.Level, resp.ExpGained, resp.LeveledUp, len(resp.Tasks))
				tasks = resp.Tasks
				return nil
			}); err != nil {
			fmt.Println("审查失败:", err)
		}

		// 选一个点击类任务（采集 type=1 / 寻宝 type=3），刷事件后点 4 次完成
		for _, task := range tasks {
			if task.EventType == 1 || task.EventType == 3 { // 采集/寻宝 = 点击
				var clickMapID int32
				if err := call(conn, next(), uint32(pb_protocol.MsgID_GameReviewTaskSelect),
					&pb_review.ReviewTaskSelectReq{TaskId: task.TaskId},
					func(env *pb_common.MessagePacket) error {
						resp := &pb_review.ReviewTaskSelectResp{}
						if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
							return err
						}
						clickMapID = resp.MapId
						fmt.Printf("[R2] 选点击任务 type=%d 刷事件 map_id=%d\n", task.EventType, resp.MapId)
						return nil
					}); err != nil {
					fmt.Println("选择任务失败:", err)
				}
				// 点 4 次（每次 +26%，超 100% 完成）
				if clickMapID != 0 {
					for i := 0; i < 4; i++ {
						if err := call(conn, next(), uint32(pb_protocol.MsgID_GameEventClick),
							&pb_worldmap.EventClickReq{MapId: clickMapID},
							func(env *pb_common.MessagePacket) error {
								resp := &pb_worldmap.EventClickRsp{}
								if err := proto.Unmarshal(env.GetBody(), resp); err != nil {
									return err
								}
								fmt.Printf("[R3] 点击进度=%d completed=%v event_type=%d\n",
									resp.Progress, resp.Completed, resp.EventType)
								return nil
							}); err != nil {
						fmt.Println("点击失败:", err)
						break
						}
					}
				}
				break
			}
		}
	}

	fmt.Println("== 流程结束，请检查服务端日志与 battle_record/redis stream")
}

func abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// formatResources 格式化资源/货币余额（按 ConfigID：粮/木/石/铁/金币/钻石）
func formatResources(items []*pb_item.ItemUse) string {
	bal := map[int32]int64{}
	for _, it := range items {
		bal[it.GetConfId()] = it.GetCount()
	}
	return fmt.Sprintf("粮=%d 木=%d 石=%d 铁=%d 金币=%d 钻石=%d",
		bal[int32(pb_confs.ResourceFoodConfID)],
		bal[int32(pb_confs.ResourceWoodConfID)],
		bal[int32(pb_confs.ResourceStoneConfID)],
		bal[int32(pb_confs.ResourceIronConfID)],
		bal[int32(pb_confs.Currency2ConfID)],
		bal[int32(pb_confs.Currency1ConfID)])
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
