package ginx

import (
	"context"

	"github.com/gin-gonic/gin"
)

func GinCtxSet(ctx *gin.Context, key string, value any) {
	ctx.Set(key, value)
	ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), key, value))
}
