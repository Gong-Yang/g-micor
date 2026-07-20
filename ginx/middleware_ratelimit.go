package ginx

import (
	"log/slog"
	"time"

	"github.com/Gong-Yang/g-micor/redisx"
	"github.com/gin-gonic/gin"
)

// RateLimitByIP 返回一个按客户端 IP 限流的中间件（Redis 固定窗口计数，多实例共享）。
//
//   - limit：单个窗口内每 IP 允许的最大请求数；
//   - window：限流时间窗口，例如 time.Hour 表示“每小时 limit 次”。
//
// 计数按「路由 + IP」维度隔离（key 含 c.FullPath()），因此不同接口即便复用同一中间件也各自独立计数。
// 客户端 IP 由 BasicMiddleware 在请求进入时写入上下文，此处直接读取。
// 为避免限流组件自身故障拖垮业务：无法获取 IP 或 Redis 异常时均放行。
func RateLimitByIP(limit int, window time.Duration) HandlerFunc {
	if limit < 1 {
		limit = 1
	}
	return func(c *gin.Context) error {
		ctx := c.Request.Context()
		ip := ClientIP(ctx)
		if ip == "" {
			slog.WarnContext(ctx, "RateLimitByIP-无法获取客户端IP，放行", "path", c.FullPath())
			return nil
		}
		key := "ratelimit:" + c.FullPath() + ":" + ip
		count, err := redisx.RateLimit(ctx, key, window)
		if err != nil {
			slog.ErrorContext(ctx, "RateLimitByIP-Redis限流异常，放行", "error", err, "key", key)
			return nil
		}
		if count > int64(limit) {
			return ErrTooManyRequests
		}
		return nil
	}
}
