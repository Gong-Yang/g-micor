package ginx

import (
	"mime/multipart"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
)

func FormFile(name string) *FormFileParam {
	return &FormFileParam{name: name}
}

type FormFileParam struct {
	name string
}

func (f *FormFileParam) GetParam(ctx *gin.Context) (res any, err error) {
	file, err := ctx.FormFile(f.name)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Body 参数
func Body[T any]() *BodyParam[T] {
	return &BodyParam[T]{}
}

type BodyParam[T any] struct{}

func (b *BodyParam[T]) GetParam(ctx *gin.Context) (res any, err error) {
	res = new(T)
	err = ctx.ShouldBindBodyWithJSON(res)
	return
}

// Header 参数
func Header(key string) *HeaderParam {
	return &HeaderParam{Key: key}
}

type HeaderParam struct {
	Key string
}

func (h *HeaderParam) GetParam(ctx *gin.Context) (res any, err error) {
	res = ctx.GetHeader(h.Key)
	return
}

// Query 参数
func Query(key, valueType string) *QueryParamItem {
	return &QueryParamItem{Key: key, Type: valueType}
}

type QueryParamItem struct {
	Key  string
	Type string
}

func (b *QueryParamItem) GetParam(ctx *gin.Context) (res any, err error) {
	value := ctx.Query(b.Key)
	return strConvert(value, b.Type)
}

// Path 参数
func Path(key, valueType string) *PathParamItem {
	return &PathParamItem{Key: key, Type: valueType}
}

type PathParamItem struct {
	Key  string
	Type string
}

func (b *PathParamItem) GetParam(ctx *gin.Context) (res any, err error) {
	value := ctx.Param(b.Key)
	return strConvert(value, b.Type)
}

type Param interface {
	GetParam(ctx *gin.Context) (res any, err error)
}

func strConvert(value string, Type string) (res any, err error) {
	if Type == STRING {
		return value, nil
	}
	if Type == INT {
		res, err = strconv.Atoi(value)
		return
	}
	if Type == FLOAT {
		res, err = strconv.ParseFloat(value, 64)
		return
	}
	if Type == BOOL {
		res, err = strconv.ParseBool(value)
		return
	}
	err = ErrDataType
	return
}

// 以下方法声明各参数的取值类型，供注册路由时校验处理函数签名

func (f *FormFileParam) paramType() reflect.Type {
	return reflect.TypeOf((*multipart.FileHeader)(nil))
}

func (b *BodyParam[T]) paramType() reflect.Type { return reflect.TypeOf((*T)(nil)) }

func (h *HeaderParam) paramType() reflect.Type { return reflect.TypeOf("") }

func (b *QueryParamItem) paramType() reflect.Type { return strConvertType(b.Type) }

func (b *PathParamItem) paramType() reflect.Type { return strConvertType(b.Type) }

func strConvertType(Type string) reflect.Type {
	switch Type {
	case STRING:
		return reflect.TypeOf("")
	case INT:
		return reflect.TypeOf(0)
	case FLOAT:
		return reflect.TypeOf(float64(0))
	case BOOL:
		return reflect.TypeOf(false)
	}
	return nil
}
