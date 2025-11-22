package errorx

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type ErrorCode struct {
	Model string `json:"model,omitempty"`
	Code  string `json:"code,omitempty"`
	Msg   string `json:"msg,omitempty"`
	Data  any    `json:"data,omitempty"`
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

func (r ErrorCode) MsgParams(args ...string) ErrorCode {
	for i, arg := range args {
		strings.ReplaceAll(r.Msg, "{"+strconv.Itoa(i)+"}", arg)
	}
	return r
}
func (r ErrorCode) SetData(data any) ErrorCode {
	r.Data = data
	return r
}

func New(model, code, msg string) ErrorCode {
	return ErrorCode{
		Model: model,
		Code:  code,
		Msg:   msg,
	}
}
