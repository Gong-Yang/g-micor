package redisx

import "reflect"

func getEntity[T any]() T {
	var result T
	// 如果T是指针类型，需要创建一个新的实例
	resultValue := reflect.ValueOf(&result).Elem()
	if resultValue.Kind() == reflect.Ptr && resultValue.IsNil() {
		// 创建一个新的实例
		elemType := resultValue.Type().Elem()
		newInstance := reflect.New(elemType)
		resultValue.Set(newInstance)
	}
	return result
}
