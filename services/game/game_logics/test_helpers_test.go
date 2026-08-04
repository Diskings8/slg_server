package game_logics

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
)

// assertErrorCode 断言 error 是 ResultI 且错误码等于 want
func assertErrorCode(t *testing.T, err error, want pb_error_code.ErrorCode) bool {
	t.Helper()
	r, ok := err.(rpc_results.ResultI)
	if !ok {
		t.Errorf("err type = %T, want rpc_results.ResultI", err)
		return false
	}
	if r.Code() != want {
		t.Errorf("code = %d, want %d", r.Code(), want)
		return false
	}
	return true
}
