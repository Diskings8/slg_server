package util_nodepacket

// 统一 NodePacket 信封构造：login / game 的 Do 统一入口共用，收敛
// `NodePacket{MsgId, Message{Body, ErrCode, DevMsg}}` 的重复展开。

import (
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_protocol"
)

// Build 构造统一 NodePacket 信封响应：成功时 body 非空 + err_code=0；错误时 code 非 0 + dev_msg
func Build(msgID pb_protocol.MsgID, body []byte, code pb_error_code.ErrorCode, devMsg string) *pb_common.NodePacket {
	return &pb_common.NodePacket{
		MsgId: msgID,
		Message: &pb_common.MessagePacket{
			Body:    body,
			ErrCode: code,
			DevMsg:  devMsg,
		},
	}
}

// Success 成功响应（err_code=0）
func Success(msgID pb_protocol.MsgID, body []byte) *pb_common.NodePacket {
	return Build(msgID, body, pb_error_code.ErrorCode_NoneErr, "")
}

// Error 错误响应（code 非 0 + dev_msg）
func Error(msgID pb_protocol.MsgID, code pb_error_code.ErrorCode, devMsg string) *pb_common.NodePacket {
	return Build(msgID, nil, code, devMsg)
}
