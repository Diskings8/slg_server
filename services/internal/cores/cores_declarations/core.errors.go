package cores_declarations

import "errors"

// 行军相关的通用错误
var (
	ErrLockFailed = errors.New("cores: 锁定失败")
)
