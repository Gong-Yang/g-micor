package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// rateLimitScript 固定窗口计数脚本：
// 对 key 自增，若为窗口内首次自增则设置过期时间，返回当前累计计数。
// 用 Lua 保证 INCR 与 EXPIRE 的原子性，避免进程在两步之间中断导致 key 永不过期。
var rateLimitScript = redis.NewScript(`
local n = redis.call("INCR", KEYS[1])
if n == 1 then
    redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return n
`)

// RateLimit 固定窗口限流原语：在 window 时间窗口内对 key 计数并返回当前累计次数。
// 调用方据返回值与阈值比较判断是否放行。首次计数时自动设置窗口过期时间。
func RateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	seconds := int(window.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return rateLimitScript.Run(ctx, Client, []string{key}, seconds).Int64()
}
