package ginx

import (
	"context"
	"errors"
	"github.com/Gong-Yang/g-micor/core/errorx"
	"github.com/Gong-Yang/g-micor/core/util/random"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"runtime"
	"time"
)

// BasicMiddleware 结果统一包装，异常捕获统一处理
func BasicMiddleware(ctx *gin.Context) {
	// 请求追踪号
	traceID := random.ShortUUID()
	ctx.Set(ContextTraceID, traceID)
	// 创建带超时的上下文
	reqCtx, cancelFunc := context.WithTimeout(ctx, time.Second*time.Duration(20))
	defer cancelFunc()
	// 将超时上下文替换到请求中，确保后续处理能感知超时
	ctx.Request = ctx.Request.WithContext(reqCtx)

	// 使用单独的 goroutine 监控超时
	timeoutChan := make(chan bool, 1)
	go func() {
		<-reqCtx.Done()
		if errors.Is(reqCtx.Err(), context.DeadlineExceeded) {
			timeoutChan <- true
		}
	}()

	// 使用 goroutine 处理请求
	doneChan := make(chan bool, 1)
	go func() {
		// 添加panic恢复处理
		defer handlePanic(ctx, doneChan)
		// 执行请求处理链
		ctx.Next()
	}()

	// 等待请求完成或超时
	select {
	case <-doneChan:
		// 请求正常完成，统一包装响应
		wrapResponse(ctx)
	case <-timeoutChan:
		// 请求超时，记录日志并返回超时响应
		slog.ErrorContext(ctx, "request timeout",
			"path", ctx.Request.URL.Path,
			"method", ctx.Request.Method,
			"timeout_seconds", 20)
		ctx.AbortWithStatus(http.StatusGatewayTimeout)
	}
}

// handlePanic 处理panic并返回适当的响应
func handlePanic(ctx *gin.Context, doneChan chan bool) {
	defer func() { doneChan <- true }()
	a := recover()
	if a == nil {
		return
	}
	wrapError(ctx, a, true)
}

// wrapResponse 统一包装响应
func wrapResponse(ctx *gin.Context) {
	// 如果已经响应则不再处理
	if ctx.Writer.Written() {
		return
	}
	value, exists := ctx.Get(ContextFuncResult)
	if !exists {
		ctx.JSON(http.StatusOK, Response{Code: errorx.RespSuccess})
		return
	}
	funcResults := value.([]interface{})
	if len(funcResults) != 2 {
		slog.ErrorContext(ctx, "invalid func result", "result", funcResults)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if funcResults[1] != nil {
		wrapError(ctx, funcResults[1], false)
		return
	}
	ctx.JSON(http.StatusOK, Response{Code: errorx.RespSuccess, Data: funcResults[0]})
}

func wrapError(ctx *gin.Context, a any, isPanic bool) {
	appErr, ok := a.(errorx.ErrorCode)
	if !ok {
		buf := make([]byte, 1024)
		n := runtime.Stack(buf, false)
		if isPanic {
			// 记录堆栈信息
			slog.ErrorContext(ctx, "panic",
				"err", a,
				"stack", string(buf[:n]),
				"path", ctx.Request.URL.Path)
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		} else {
			slog.WarnContext(ctx, "sysError",
				"err", a,
				"stack", string(buf[:n]),
				"path", ctx.Request.URL.Path)
			appErr = errorx.ErrorCode{Code: errorx.RespErr, Msg: "请联系管理员"}
		}
	}

	// 业务错误
	slog.InfoContext(ctx, "business err response",
		"err", appErr,
		"response", appErr,
		"path", ctx.Request.URL.Path)
	ctx.AbortWithStatusJSON(http.StatusOK, appErr)
}
