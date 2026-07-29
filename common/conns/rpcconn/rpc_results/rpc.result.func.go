package rpc_results

import (
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/utils/util_bytes"
)

var resultPool = util_bytes.NewPool(func() ResultI {
	return &result{Status: &status.Status{}}
})

// Error 报错
func Error(code pb_error_code.ErrorCode, devMsg string) ResultI {
	s := resultPool.Get()
	s.SetCode(code)
	s.SetDevMsg(devMsg)
	return s
}

// ErrorParam 参数错误
func ErrorParam(msg string) ResultI {
	return Error(pb_error_code.ErrorCode_ParamError, msg)
}

func ErrorCode(code pb_error_code.ErrorCode) ResultI {
	return Error(code, "")
}
