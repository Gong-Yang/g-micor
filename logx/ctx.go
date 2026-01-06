package logx

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func RequestAddAttrs(ctx *gin.Context, key string, value any) {
	res := AddAttrs(ctx.Request.Context(), key, value)
	ctx.Request = ctx.Request.WithContext(res)
}

var LogAttrsKey = "logAttrs"

func AddAttrs(ctx context.Context, key string, value any) (res context.Context) {
	// 从标准 context 中获取现有属性
	raw := ctx.Value(LogAttrsKey)
	var attrs map[string]any
	if raw == nil {
		attrs = make(map[string]any)
	} else {
		// 复制现有 map 保证并发安全
		existing := raw.(map[string]any)
		attrs = make(map[string]any, len(existing)+1)
		for k, v := range existing {
			attrs[k] = v
		}
	}
	attrs[key] = value
	return context.WithValue(ctx, LogAttrsKey, attrs)
}
func recordAddAttrs(ctx context.Context, record *slog.Record) {
	value := ctx.Value(LogAttrsKey)
	if value == nil {
		return
	}
	logAttrs := value.(map[string]any)
	for k, v := range logAttrs {
		record.AddAttrs(slog.Any(k, v))
	}
}
