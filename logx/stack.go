package logx

import (
	"runtime"
	"strings"
)

// frameNoise 用于判断一帧是否属于框架/标准库噪音，过滤后只保留业务调用链。
var frameNoise = []string{
	"runtime.",
	"net/http.",
	"github.com/gin-gonic/gin",
	"github.com/Gong-Yang/g-micor/ginx",
}

// isNoiseFrame 判断该函数名是否属于需要过滤掉的框架/标准库帧。
func isNoiseFrame(function string) bool {
	for _, p := range frameNoise {
		if strings.Contains(function, p) {
			return true
		}
	}
	return false
}

// CleanStack 采集当前 goroutine 的调用栈并清洗成简短、按业务关心程度排序的多行文本，
// 仅用于 panic 场景（此时出错帧仍在栈上）。
//
// 相比 runtime.Stack 的原始输出，这里：
//   - 用结构化帧而非字符串切片，不会误砍栈顶；
//   - 复用 shortSource 输出 "module/xxx/service/score.go:159" 短路径；
//   - 过滤 runtime / net/http / gin / ginx 等框架噪音；
//   - 去掉偏移量(+0x312)与 goroutine 指针。
//
// skip 表示需要跳过的栈顶帧数（不含 CleanStack 自身，CleanStack 自身已自动跳过）。
func CleanStack(skip int) string {
	// 最多采集 32 帧，足够定位业务问题
	pcs := make([]uintptr, 32)
	// +2：跳过 runtime.Callers 与 CleanStack 自身
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return ""
	}

	// 一遍遍历，同时收集"业务帧"与"全部帧"
	frames := runtime.CallersFrames(pcs[:n])
	var business, all []string
	for {
		f, more := frames.Next()
		if f.Function != "" {
			line := shortSource(f.Function, f.File, f.Line)
			all = append(all, line)
			if !isNoiseFrame(f.Function) {
				business = append(business, line)
			}
		}
		if !more {
			break
		}
	}

	// 兜底：若 panic 直接发生在框架/标准库内部，业务帧会被全部过滤掉，
	// 此时退回打印未过滤的完整帧，避免日志里出现空堆栈反而更难排查。
	lines := business
	if len(lines) == 0 {
		lines = all
	}
	return strings.Join(lines, "\n")
}
