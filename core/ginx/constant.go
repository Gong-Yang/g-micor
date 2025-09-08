package ginx

import (
	"errors"
)

// 错误
var (
	ErrIsNotFunc = errors.New("not func")          // handler方法非法
	ErrDataType  = errors.New("invalid data type") // 参数类型非法
)

// 上下文常量
var (
	ContextTraceID    = "TraceID"
	ContextFuncResult = "FuncResult"
)

// 参数类型
var (
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"
	BOOL   = "BOOL"
)

var (
	RespSuccess = "S000"
	RespErr     = "F000"
)
