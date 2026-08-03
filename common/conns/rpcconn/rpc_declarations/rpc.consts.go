package rpc_declarations

import "time"

type RpcStreamName string

const (
	RpcStreamGame2WorldMap RpcStreamName = "game -> world_map"
	RpcStreamGate2Game     RpcStreamName = "gate -> game"
)

// DefaultRPCTimeout 内置默认 RPC 连接（拨号）超时
const DefaultRPCTimeout = 3 * time.Second
