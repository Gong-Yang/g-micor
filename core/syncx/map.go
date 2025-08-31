package syncx

import "sync"

func NewMap[T any]() Map[T] {
	return Map[T]{
		Map: new(sync.Map),
	}
}

type Map[T any] struct {
	*sync.Map
}

func (m Map[T]) Store(key string, value T) {
	m.Map.Store(key, value)
}
func (m Map[T]) Load(key string) (T, bool) {
	v, ok := m.Map.Load(key)
	if !ok {
		var zero T
		return zero, false
	}
	return v.(T), true
}
