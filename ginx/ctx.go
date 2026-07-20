package ginx

import (
	"context"

	"github.com/gin-gonic/gin"
)

func GinCtxSet(ctx *gin.Context, key string, value any) {
	ctx.Set(key, value)
	ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), key, value))
}

// ClientIP 读取由 BasicMiddleware 在请求进入时写入的真实客户端 IP。
// 若未能解析到 IP，返回空字符串。
func ClientIP(ctx context.Context) string {
	v, _ := ctx.Value(ContextClientIP).(string)
	return v
}
