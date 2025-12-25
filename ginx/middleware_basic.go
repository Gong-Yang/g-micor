package ginx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/Gong-Yang/g-micor/errorx"
	"github.com/Gong-Yang/g-micor/logx"
	"github.com/Gong-Yang/g-micor/util/random"
	"github.com/gin-gonic/gin"
)

// BasicMiddleware 结果统一包装，异常捕获统一处理
func BasicMiddleware(ctx *gin.Context) {
	// 请求追踪号
	traceID := random.ShortUUID()
	ctx.Set(ContextTraceID, traceID)
	logx.RequestAddAttrs(ctx, ContextTraceID, traceID)
	// 添加panic恢复处理
	defer handlePanic(ctx)
	// 执行请求处理链
	ctx.Next()
	// 请求正常完成，统一包装响应
	wrapResponse(ctx)
}

// handlePanic 处理panic并返回适当的响应
func handlePanic(ctx *gin.Context) {
	a := recover()
	if a == nil {
		return
	}
	wrapError(ctx, a, true)
}

// wrapResponse 统一包装响应
func wrapResponse(c *gin.Context) {
	ctx := c.Request.Context()
	// 如果已经响应则不再处理
	if c.Writer.Written() {
		return
	}
	value, exists := c.Get(ContextFuncResult)
	if !exists {
		c.JSON(http.StatusOK, Response{Code: errorx.RespSuccess})
		return
	}
	funcResults := value.([]interface{})
	if len(funcResults) != 2 {
		slog.ErrorContext(ctx, "invalid func result", "result", funcResults)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if funcResults[1] != nil {
		wrapError(c, funcResults[1], false)
		return
	}
	c.JSON(http.StatusOK, Response{Code: errorx.RespSuccess, Data: funcResults[0]})
}

func wrapError(c *gin.Context, a any, isPanic bool) {
	ctx := c.Request.Context()
	appErr, ok := a.(errorx.ErrorCode)
	if !ok {
		stackTrace := getStackTrace()
		if isPanic {
			// 记录堆栈信息
			slog.ErrorContext(ctx, "panic",
				"err", a,
				"stack", stackTrace,
				"path", c.Request.URL.Path)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		} else {
			slog.WarnContext(ctx, "sysError",
				"err", a,
				"stack", stackTrace,
				"path", c.Request.URL.Path)
			appErr = errorx.ErrorCode{Code: errorx.RespErr, Msg: "请联系管理员"}
		}
	}

	// 业务错误
	slog.InfoContext(ctx, "business err response",
		"err", appErr,
		"response", appErr,
		"path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusOK, appErr)
}

// getStackTrace 获取堆栈跟踪信息
func getStackTrace() string {
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	return string(buf[512:n])
}

var timeOutMap = map[int]HandlerFunc{}

// MidTimeOut 处理请求超时控制
func MidTimeOut(seconds int) HandlerFunc {
	if hand, ok := timeOutMap[seconds]; ok {
		return hand
	}
	hand := func(ctx *gin.Context) error {
		// 创建带超时的上下文
		reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), time.Second*time.Duration(seconds))

		// 将超时上下文替换到请求中
		ctx.Request = ctx.Request.WithContext(reqCtx)

		// 使用 goroutine 监控超时，但不阻塞主流程
		go func() {
			<-reqCtx.Done()
			//slog.InfoContext(ctx, "reqCtx done", "path", ctx.FullPath(), "err", reqCtx.Err())
			if errors.Is(reqCtx.Err(), context.DeadlineExceeded) {
				slog.ErrorContext(reqCtx, "request timeout",
					"path", ctx.FullPath(),
					"method", ctx.Request.Method,
					"timeout_seconds", seconds)
				// 注意：这里的 AbortWithStatus 可能在请求已完成后调用
				// 但 Gin 会自动处理这种情况
				ctx.AbortWithStatus(http.StatusGatewayTimeout)
			}
		}()
		defer cancel()
		ctx.Next()
		return nil
	}
	timeOutMap[seconds] = hand
	return hand
}

type HandlerFunc func(ctx *gin.Context) error

func handlerConvert(in []HandlerFunc) (res []gin.HandlerFunc) {
	res = make([]gin.HandlerFunc, len(in))
	for i, handlerFunc := range in {
		res[i] = func(ctx *gin.Context) {
			if err := handlerFunc(ctx); err != nil {
				wrapError(ctx, err, false)
			}
		}
	}
	return
}

//func handAuth(ctx *gin.Context, conf *RouterConf) (AuthUser, error) {
//	if conf.author == nil { // 无认证者
//		return nil, nil
//	}
//
//	user, err := conf.author.Auth(conf.appId, ctx)
//	if err != nil {
//		if conf.needLogin {
//			return nil, err // 需要登录
//		}
//		return nil, nil
//	}
//
//	if conf.role != "" && user.GetRole() != conf.role {
//		return nil, ErrAuthFail
//	}
//
//	GinCtxSet(ctx, ContextAuthUser, user)
//	return user, nil
//}
