package ginx

import (
	"errors"
	"fmt"
)

// 响应码对象
type Response struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data any    `json:"data,omitempty"`
}

type ErrorCode struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

// 实现error接口
func (r ErrorCode) Error() string {
	return fmt.Sprintf("%s:%s", r.Code, r.Msg)
}
func (r ErrorCode) Is(target error) bool {
	// 类型断言，确保target是MyError类型
	var t ErrorCode
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	// 比较关键字段Code
	return r.Code == t.Code
}
func (r ErrorCode) ToRes() Response {
	res := Response{
		Code: r.Code,
		Msg:  r.Msg,
	}
	return res
}
