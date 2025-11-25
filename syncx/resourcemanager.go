package syncx

import (
	"sync"
)

// A ResourceManager is a manager that used to manage resources.
type ResourceManager[T any] struct {
	resources    map[string]T
	singleFlight *SingleFlight[T]
	lock         sync.RWMutex
}

// NewResourceManager returns a ResourceManager.
func NewResourceManager[T any]() *ResourceManager[T] {
	flight := NewSingleFlight[T]()
	return &ResourceManager[T]{
		resources:    make(map[string]T),
		singleFlight: flight,
	}
}

// GetResource returns the resource associated with given key.
func (manager *ResourceManager[T]) GetResource(key string, create func() (T, error)) (
	res T, err error) {
	res, err = manager.singleFlight.Do(key, func() (res T, err error) {
		manager.lock.RLock()
		resource, ok := manager.resources[key]
		manager.lock.RUnlock()
		if ok {
			return resource, nil
		}

		res, err = create()
		if err != nil {
			return
		}

		manager.lock.Lock()
		defer manager.lock.Unlock()
		manager.resources[key] = resource

		return resource, nil
	})
	if err != nil {
		return
	}

	return res, nil
}

// Inject injects the resource associated with given key.
func (manager *ResourceManager[T]) Inject(key string, resource T) {
	manager.lock.Lock()
	manager.resources[key] = resource
	manager.lock.Unlock()
}
