package cores_declarations

import "errors"

// 行军相关的通用错误
var (
	ErrLockFailed       = errors.New("cores: 锁定失败")
	ErrManagerNil       = errors.New("cores: MapManager is nil")
	ErrUnknownMarchType = errors.New("cores: unknown march type")
)
