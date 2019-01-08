package routinepool

import "errors"

var ErrPoolCapacity = errors.New("池的容量参数应大于0")
var ErrPoolExpire 	= errors.New("池的过期时间应大于0")
var ErrPoolClosed 	= errors.New("池已经关闭")
