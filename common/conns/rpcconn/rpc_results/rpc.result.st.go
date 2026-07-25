package rpc_results

import (
	"fmt"

	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_error_code"
)

type result struct {
	*status.Status
	code   pb_error_code.ErrorCode
	devMsg string
}

func (r *result) Code() pb_error_code.ErrorCode {
	return r.code
}

func (r *result) SetCode(code pb_error_code.ErrorCode) {
	r.code = code
}

func (r *result) DevMsg() string {
	return r.devMsg
}

func (r *result) SetDevMsg(msg string) {
	r.devMsg = msg
}

func (r *result) Error() string {
	return fmt.Sprintf("code: %d, devMsg: %s", r.Code(), r.DevMsg())
}

func (r *result) Reset() {
	//TODO implement me
	panic("implement me")
}
