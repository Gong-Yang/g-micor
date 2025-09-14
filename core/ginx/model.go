package ginx

import (
	"encoding/json"
	"errors"
)

// 响应码对象
type Response struct {
	Model string `json:"model,omitempty"`
	Code  string `json:"code,omitempty"`
	Msg   string `json:"msg,omitempty"`
	Data  any    `json:"data,omitempty"`
}

type ErrorCode struct {
	Model string `json:"model,omitempty"`
	Code  string `json:"code,omitempty"`
	Msg   string `json:"msg,omitempty"`
}

// 实现error接口
func (r ErrorCode) Error() string {
	marshal, _ := json.Marshal(r)
	return string(marshal)
}
func (r ErrorCode) Is(target error) bool {
	// 类型断言，确保target是MyError类型
	var t ErrorCode
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	// 比较关键字段Code
	return r.Model == t.Model && r.Code == t.Code
}
func (r ErrorCode) ToRes() Response {
	res := Response{
		Model: r.Model,
		Code:  r.Code,
		Msg:   r.Msg,
	}
	return res
}
func NewErr(model, code, msg string) error {
	return ErrorCode{
		Model: model,
		Code:  code,
		Msg:   msg,
	}
}
