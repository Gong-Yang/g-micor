package ginx

import (
	"context"
	"reflect"

	"github.com/gin-gonic/gin"
)

func POST(group gin.IRouter, conf *RouterConf, path string, fun any, params ...Param) {
	if conf == nil {
		panic("conf is nil")
	}
	checkFun(fun)
	mid := conf.GetMiddleware()
	var handler = getHandler(fun, params)
	group.POST(path, mid, handler)
}
func GET(group gin.IRouter, conf *RouterConf, path string, fun any, params ...Param) {
	if conf == nil {
		panic("conf is nil")
	}
	checkFun(fun)
	mid := conf.GetMiddleware()
	var handler = getHandler(fun, params)
	group.GET(path, mid, handler)
}

func PUT(group gin.IRouter, conf *RouterConf, path string, fun any, params ...Param) {
	if conf == nil {
		panic("conf is nil")
	}
	checkFun(fun)
	mid := conf.GetMiddleware()
	var handler = getHandler(fun, params)
	group.PUT(path, mid, handler)
}

func DELETE(group gin.IRouter, conf *RouterConf, path string, fun any, params ...Param) {
	if conf == nil {
		panic("conf is nil")
	}
	checkFun(fun)
	mid := conf.GetMiddleware()
	var handler = getHandler(fun, params)
	group.DELETE(path, mid, handler)
}

func Any(group gin.IRouter, conf *RouterConf, path string, fun any, params ...Param) {
	if conf == nil {
		panic("conf is nil")
	}
	// 检查 fun 是否是可调用的
	checkFun(fun)
	mid := conf.GetMiddleware()
	var handler = getHandler(fun, params)
	group.Any(path, mid, handler)
}

func process(ctx context.Context, ginCtx *gin.Context, fun any, params []Param) (res []interface{}) {
	// 反射获取反射类型对象
	funValue := reflect.ValueOf(fun)

	// 准备参数
	args := make([]reflect.Value, len(params)+1)
	args[0] = reflect.ValueOf(ctx)
	for i, param := range params {
		var arg any
		arg, err := param.GetParam(ginCtx)
		if err != nil {
			panic(err)
			return
		}
		args[i+1] = reflect.ValueOf(arg)
	}

	//调用
	responses := funValue.Call(args)
	res = make([]interface{}, len(responses))
	for i, response := range responses {
		res[i] = response.Interface()
	}
	return
}

func checkFun(fun any) {
	// 检查 fun 是否是可调用的
	funValue := reflect.ValueOf(fun)
	if funValue.Kind() != reflect.Func {
		panic(ErrIsNotFunc)
		return
	}
}

func getHandler(fun any, params []Param) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		resArr := process(ctx.Request.Context(), ctx, fun, params)
		ctx.Set(ContextFuncResult, resArr)
	}
}
