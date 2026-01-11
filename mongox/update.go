package mongox

import "go.mongodb.org/mongo-driver/bson"

type UpdateBuilder[T any] struct {
	Update []*UpdateInfo `json:"update,omitempty"`
}
type UpdateInfo struct {
	Key       string `json:"key,omitempty"`       // 操作Key,通过 “.” 分割
	Value     string `json:"value,omitempty"`     // 操作值，固定提供字符串
	Operation string `json:"operation,omitempty"` // 操作方式 $set $unset $rename $push
}

func (u UpdateBuilder[T]) ToUpdate() bson.M {
	var (
		t   T
		res bson.M
	)
	// 检测key合法

}
