package rpc_results

import (
	"server.slg.com/api/protocol/pb/pb_error_code"
)

// ResultI type ResultI interface { ..
type ResultI interface {
	Code() pb_error_code.ErrorCode
	SetCode(code pb_error_code.ErrorCode)
	DevMsg() string
	SetDevMsg(msg string)
	// error 接口
	Error() string
	// pool 接口
	Reset()
}
