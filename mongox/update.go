package mongox

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

var opFun = map[string]func(ctx context.Context, t any, key string, value any, res bson.M) error{
	"$set": func(ctx context.Context, obj any, key string, value any, res bson.M) error {
		t := reflect.TypeOf(obj)

		keyArr := strings.Split(key, ".")
		for i, keyItem := range keyArr {
			// 拿到具体的类型
			if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
				t = t.Elem()
				continue
			}
			// 本层级如果是数字 则跳过 进入下一层
			index, err := strconv.Atoi(keyItem)
			if err == nil {
				if index < 0 {
					return errors.New("invalid index")
				}
				continue
			}
			// 拿到字段

			t.FieldByName()
		}
	},
}

type structInfo struct {
	Type  reflect.Type
	Field map[string]*structInfo
}

type UpdateBuilder[T any] struct {
	Update []*UpdateInfo `json:"update,omitempty"`
}
type UpdateInfo struct {
	Key       string `json:"key,omitempty"`       // 操作Key,通过 “.” 分割
	Value     string `json:"value,omitempty"`     // 操作值，固定提供字符串
	Operation string `json:"operation,omitempty"` // 操作方式 $set $unset $rename $push
}

func (u UpdateBuilder[T]) ToUpdate(ctx context.Context) (res bson.M, err error) {
	var (
		t T
	)
	for _, one := range u.Update {
		if one.Key == "" {
			return nil, errors.New("key is empty")
		}
		fn, ok := opFun[one.Operation]
		if !ok {
			return nil, errors.New("无效的操作方式")
		}

		err = fn(ctx, t, one.Key, one.Value, res)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// 递归获取所有字段信息
func getStructInfo(obj any) (*structInfo, error) {
	t := reflect.TypeOf(obj)
	t = getRelType(t)
	fieldMap := map[string]*structInfo{}
	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("bson")
		if tag == "" || tag == "-" {
			continue
		}
		fieldName := strings.Split(tag, ",")[0]
		if field {

		}

		fieldMap[fieldName] = &structInfo{
			Type: field.Type,
		}
	}
	res := &structInfo{
		Type:  t,
		Field: fieldMap,
	}

}
func getRelType(t reflect.Type) reflect.Type {
	for {
		kind := t.Kind()
		if kind == reflect.Pointer || kind == reflect.Slice {
			t = t.Elem()
		} else {
			return t
		}
	}
}
