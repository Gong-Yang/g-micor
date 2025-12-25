package ginx

import (
	"github.com/gin-gonic/gin"
)

// 响应码对象
type Response struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data any    `json:"data,omitempty"`
}

// Author 鉴权者
type Author interface {
	Auth(appid string, c *gin.Context) (AuthUser, error)
}
type AuthUser interface {
	GetRole() string
}
