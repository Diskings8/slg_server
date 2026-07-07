package rpc_declarations

type RpcStreamName string

const (
	RpcStreamGame2WorldMap RpcStreamName = "game -> world_map"
	RpcStreamGate2Game     RpcStreamName = "gate -> game"
)
