package gm_handler

import (
	"context"

	"server.slg.com/common/conns/rpcconn/rpc_results"
)

// GmHandleFunc GM 功能处理函数
//   - ctx: 上下文
//   - roleID: 目标角色ID（已由 HandlerGm 解析：GmReq.role_id，0=当前操作角色）
//   - args: 命令参数（各 cmd 按约定自行解析）
//
// 返回:
//   - msg: 成功结果文本（回填 GmResp.msg）
//   - result: nil 表示成功，ResultI 表示业务错误
type GmHandleFunc func(ctx context.Context, roleID uint64, args []string) (string, rpc_results.ResultI)

// GmCmdInfo GM 命令注册项
type GmCmdInfo struct {
	Cmd string
	F   GmHandleFunc
}

// gmCmdRegistry cmd 命令字 → 处理器映射表
var gmCmdRegistry = map[string]*GmCmdInfo{}

// RegisterGmCmd 注册 GM 命令处理器
func RegisterGmCmd(cmd string, f GmHandleFunc) {
	gmCmdRegistry[cmd] = &GmCmdInfo{Cmd: cmd, F: f}
}

// GetGmCmd 获取 GM 命令处理器
func GetGmCmd(cmd string) (*GmCmdInfo, bool) {
	info, ok := gmCmdRegistry[cmd]
	return info, ok
}
