package ginx

import (
	"time"

	"github.com/gin-gonic/gin"
)

// 响应码对象
type Response struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data any    `json:"data,omitempty"`
}

type RouterConf struct {
	timeOut    time.Duration // 路由处理超时时间
	sinWay     string        // 签名方式, 空串无需签名， "Normal" 默认签名方式
	needLogin  bool          // 接口权限等级（开放/登录/角色）
	role       string
	appId      string
	author     Author
	middleware func(ctx *gin.Context)
}

func (r *RouterConf) GetMiddleware() func(ctx *gin.Context) {
	if r.middleware != nil {
		return r.middleware
	}
	if r.needLogin == true && (r.appId == "" || r.author == nil) {
		panic("appId/author is must")
	}
	r.middleware = func(ctx *gin.Context) {
		// TODO 接口验签
		// TODO 接口鉴权
		_, err := handAuth(ctx, r)
		if err != nil {
			wrapError(ctx, err, false)
			return
		}
		// 处理超时控制
		cancel := handleTimeout(ctx, r)
		defer cancel()
		//slog.InfoContext(ctx, "request start", "path", ctx.FullPath())
		ctx.Next()
		//slog.InfoContext(ctx, "request done", "path", ctx.FullPath())
	}
	return r.middleware
}

// Author 鉴权者
type Author interface {
	Auth(appid string, c *gin.Context) (AuthUser, error)
}
type AuthUser interface {
	GetRole() string
}
