package ginx

import (
	"github.com/gin-gonic/gin"
	"reflect"
)

func POST(group gin.IRouter, path string, fun any, params ...Param) {
	checkFun(fun)
	var handler = func(ctx *gin.Context) {
		resArr := process(ctx, fun, params)
		ctx.Set(ContextFuncResult, resArr)
	}
	group.POST(path, handler)
}

func GET(group gin.IRouter, path string, fun any, params ...Param) {
	checkFun(fun)
	var handler = func(ctx *gin.Context) {
		resArr := process(ctx, fun, params)
		ctx.Set(ContextFuncResult, resArr)
	}
	group.GET(path, handler)
}

func PUT(group gin.IRouter, path string, fun any, params ...Param) {
	checkFun(fun)
	var handler = func(ctx *gin.Context) {
		resArr := process(ctx, fun, params)
		ctx.Set(ContextFuncResult, resArr)
	}
	group.PUT(path, handler)
}

func DELETE(group gin.IRouter, path string, fun any, params ...Param) {
	checkFun(fun)
	var handler = func(ctx *gin.Context) {
		resArr := process(ctx, fun, params)
		ctx.Set(ContextFuncResult, resArr)
	}
	group.DELETE(path, handler)
}

func Any(group gin.IRouter, path string, fun any, params ...Param) {
	// 检查 fun 是否是可调用的
	checkFun(fun)
	var handler = func(ctx *gin.Context) {
		resArr := process(ctx, fun, params)
		ctx.Set(ContextFuncResult, resArr)
	}
	group.Any(path, handler)
}

func process(ctx *gin.Context, fun any, params []Param) (res []interface{}) {
	// 反射获取反射类型对象
	funValue := reflect.ValueOf(fun)

	// 准备参数
	args := make([]reflect.Value, len(params)+1)
	args[0] = reflect.ValueOf(ctx)
	for i, param := range params {
		var arg any
		arg, err := param.GetParam(ctx)
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
