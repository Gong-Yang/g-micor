package syncx

import "sync"

type (
	call[T any] struct {
		wg  sync.WaitGroup
		val T
		err error
	}

	SingleFlight[T any] struct {
		calls map[string]*call[T]
		lock  sync.Mutex
	}
)

// NewSingleFlight returns a SingleFlight.
func NewSingleFlight[T any]() *SingleFlight[T] {
	return &SingleFlight[T]{
		calls: make(map[string]*call[T]),
	}
}

func (g *SingleFlight[T]) Do(key string, fn func() (T, error)) (T, error) {
	c, done := g.createCall(key)
	if done {
		return c.val, c.err
	}

	g.makeCall(c, key, fn)
	return c.val, c.err
}

func (g *SingleFlight[T]) DoEx(key string, fn func() (T, error)) (val any, fresh bool, err error) {
	c, done := g.createCall(key)
	if done {
		return c.val, false, c.err
	}

	g.makeCall(c, key, fn)
	return c.val, true, c.err
}

func (g *SingleFlight[T]) createCall(key string) (c *call[T], done bool) {
	g.lock.Lock()
	if c, ok := g.calls[key]; ok {
		g.lock.Unlock()
		c.wg.Wait()
		return c, true
	}

	c = new(call[T])
	c.wg.Add(1)
	g.calls[key] = c
	g.lock.Unlock()

	return c, false
}

func (g *SingleFlight[T]) makeCall(c *call[T], key string, fn func() (T, error)) {
	defer func() {
		g.lock.Lock()
		delete(g.calls, key)
		g.lock.Unlock()
		c.wg.Done()
	}()

	c.val, c.err = fn()
}
