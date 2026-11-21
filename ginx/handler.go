package ginx

import (
	"context"
	"fmt"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
)

func POST(group gin.IRouter, mid []HandlerFunc, path string, fun any, params ...Param) {
	checkFun(fun, params)
	var handler = getHandler(fun, params)
	group.POST(path, append(handlerConvert(mid), handler)...)
}
func GET(group gin.IRouter, mid []HandlerFunc, path string, fun any, params ...Param) {
	checkFun(fun, params)
	var handler = getHandler(fun, params)
	group.GET(path, append(handlerConvert(mid), handler)...)
}

func Any(group gin.IRouter, mid []HandlerFunc, path string, fun any, params ...Param) {
	checkFun(fun, params)
	var handler = getHandler(fun, params)
	group.Any(path, append(handlerConvert(mid), handler)...)
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

var (
	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errType = reflect.TypeOf((*error)(nil)).Elem()
)

// typedParam 能声明自身取值类型的参数，用于注册时校验签名
type typedParam interface {
	paramType() reflect.Type
}

// checkFun 注册路由时校验处理函数签名：
// func(ctx context.Context, 参数1, 参数2...) (结果, error)，参数顺序和类型与 params 一致。
func checkFun(fun any, params []Param) {
	funValue := reflect.ValueOf(fun)
	if funValue.Kind() != reflect.Func {
		panic(ErrIsNotFunc)
	}
	t := funValue.Type()
	name := runtime.FuncForPC(funValue.Pointer()).Name()
	if t.NumIn() != len(params)+1 {
		panic(fmt.Sprintf("ginx: %s 应有 %d 个参数（ctx + %d 个路由参数），实际 %d 个", name, len(params)+1, len(params), t.NumIn()))
	}
	if t.In(0) != ctxType {
		panic(fmt.Sprintf("ginx: %s 第 1 个参数必须是 context.Context", name))
	}
	for i, p := range params {
		tp, ok := p.(typedParam)
		if !ok || tp.paramType() == nil {
			continue
		}
		if !tp.paramType().AssignableTo(t.In(i + 1)) {
			panic(fmt.Sprintf("ginx: %s 第 %d 个参数应为 %s，实际为 %s", name, i+2, tp.paramType(), t.In(i+1)))
		}
	}
	if t.NumOut() != 2 || t.Out(1) != errType {
		panic(fmt.Sprintf("ginx: %s 返回值必须是 (结果, error)", name))
	}
}

func getHandler(fun any, params []Param) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		resArr := process(ctx.Request.Context(), ctx, fun, params)
		ctx.Set(ContextFuncResult, resArr)
	}
}
