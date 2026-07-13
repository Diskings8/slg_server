package common_declarations

import "errors"

// ErrPollerTimeout 轮询器超时错误
var ErrPollerTimeout = errors.New("poller timeout")
