package dayu

import "errors"

const (
	DefaultPoolSize          = 1<<31 - 1
	DefaultCleanIntervalTime = 60
)

var (
	ErrInvalidPoolSize   = errors.New("invalid pool size")
	ErrInvalidPoolExpire = errors.New("invalid pool expire")
	ErrPoolClosed        = errors.New("pool was closed")
)
