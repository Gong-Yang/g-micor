package redisx

import (
	"context"
	"time"
)

func NewSimpleCacher[T comparable, K CacherKey](prefix string, expire time.Duration) *SimpleCacher[T, K] {
	return &SimpleCacher[T, K]{
		expire: expire,
		prefix: prefix,
	}
}

type SimpleCacher[T comparable, K CacherKey] struct {
	expire time.Duration
	prefix string
}

// Get 从Redis获取简单类型数据
func (c *SimpleCacher[T, K]) Get(ctx context.Context, key K) (T, error) {
	cacheKey := c.prefix + ":" + key.GenKey()
	return GetSimple[T](ctx, cacheKey)
}

// Set 将简单类型数据存入Redis
func (c *SimpleCacher[T, K]) Set(ctx context.Context, key K, value T) error {
	cacheKey := c.prefix + ":" + key.GenKey()
	return SetSimple(ctx, cacheKey, value, c.expire)
}

// GetAndDelete 从Redis获取简单类型数据并删除
func (c *SimpleCacher[T, K]) GetAndDelete(ctx context.Context, key K) (T, error) {
	var result T
	cacheKey := c.prefix + ":" + key.GenKey()
	result, err := GetSimple[T](ctx, cacheKey)
	if err == nil {
		Client.Del(ctx, cacheKey)
	}
	return result, err
}

// GetWithFallback 使用回源模式获取数据，实现缓存穿透保护
func (c *SimpleCacher[T, K]) GetWithFallback(ctx context.Context, key K, fallback func() (T, error)) (T, error) {
	// 先尝试从缓存获取
	result, err := c.Get(ctx, key)
	if err == nil {
		return result, nil
	}

	// 缓存未命中，调用回源函数
	result, err = fallback()
	if err != nil {
		return result, err
	}

	// 将结果存入缓存
	_ = c.Set(ctx, key, result)
	return result, nil
}
