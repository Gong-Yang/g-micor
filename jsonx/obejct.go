package jsonx

import "encoding/json"

func Convert[T any](obj any) (res *T, err error) {
	marshal, err := json.Marshal(obj)
	if err != nil {
		return
	}
	res = new(T)
	err = json.Unmarshal(marshal, res)
	return
}
