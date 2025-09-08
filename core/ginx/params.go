package ginx

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

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
