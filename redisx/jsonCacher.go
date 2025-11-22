package redisx

import (
	"context"
	"time"
)

// JSONCacher 基于JSON序列化的Redis缓存器
type JSONCacher[T any, K CacherKey] struct {
	expire time.Duration
	prefix string
}

// NewJSONCacher 创建一个新的JSON缓存器实例
func NewJSONCacher[T any, K CacherKey](prefix string, expire time.Duration) *JSONCacher[T, K] {
	return &JSONCacher[T, K]{
		expire: expire,
		prefix: prefix,
	}
}

// Get 从Redis获取并反序列化JSON对象
func (c *JSONCacher[T, K]) Get(ctx context.Context, key K) (T, error) {
	result := getEntity[T]()
	cacheKey := c.prefix + ":" + key.GenKey()
	err := GetJSON(ctx, cacheKey, &result)
	return result, err
}

// Set 将对象序列化为JSON并存入Redis
func (c *JSONCacher[T, K]) Set(ctx context.Context, key K, value T) error {
	cacheKey := c.prefix + ":" + key.GenKey()
	return SetJSON(ctx, cacheKey, value, c.expire)
}

// GetAndDelete 从Redis获取并反序列化JSON对象
func (c *JSONCacher[T, K]) GetAndDelete(ctx context.Context, key K) (T, error) {
	result := getEntity[T]()
	cacheKey := c.prefix + ":" + key.GenKey()
	err := GetJSON(ctx, cacheKey, &result)
	if err == nil {
		Client.Del(ctx, cacheKey)
	}
	return result, err
}

// GetWithFallback 使用回源模式获取数据，实现缓存穿透保护
func (c *JSONCacher[T, K]) GetWithFallback(ctx context.Context, key K, fallback func() (T, error)) (T, error) {
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
