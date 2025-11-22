package ginx

import (
	"errors"
	"time"

	"github.com/Gong-Yang/g-micor/errorx"
)

// 错误
var (
	ErrIsNotFunc = errors.New("not func")          // handler方法非法
	ErrDataType  = errors.New("invalid data type") // 参数类型非法
	ErrAuthFail  = errorx.New("system", "E001", "no auth")
)

// 上下文常量
var (
	ContextTraceID    = "TraceID"
	ContextFuncResult = "FuncResult"
	ContextAuthUser   = "AuthUser"
)

// 参数类型
var (
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"
	BOOL   = "BOOL"
)

// 权限等级常量
var (
	PermissionLevelOpen       = 0  // 开放接口，无需认证
	PermissionLevelLogin      = 10 // 需要登录认证
	PermissionLevelRole       = 11 // 需要特定角色权限
	PermissionLevelPermission = 12 // 需要特定权限控制
)

// 签名方式
var (
	SignWayNormal = "Normal"
)

const (
	DefaultTimeOut = 10 * time.Second
)
