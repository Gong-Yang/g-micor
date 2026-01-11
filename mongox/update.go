package mongox

import "go.mongodb.org/mongo-driver/bson"

type UpdateBuilder[T any] []*UpdateInfo
type UpdateInfo struct {
	Key       string // 操作Key,通过 “.” 分割
	Value     string // 操作值，固定提供字符串
	Operation string // 操作方式 $set $unset $rename $push
}

func (u UpdateBuilder[T]) ToUpdate() bson.M {
	var (
		t   T
		res bson.M
	)
	for _, info := range u {
		//
	}
}
