package redisx

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
)

type Cacher[T proto.Message, K CacherKey] struct {
	expire time.Duration
	prefix string
}

func NewCacher[T proto.Message, K CacherKey](prefix string, expire time.Duration) *Cacher[T, K] {
	return &Cacher[T, K]{
		expire: expire,
		prefix: prefix,
	}
}

type CacherKey interface {
	GenKey() string
}

func (c *Cacher[T, K]) Get(ctx context.Context, key K) (T, error) {
	result := getEntity[T]()
	cacheKey := c.prefix + ":" + key.GenKey()
	err := GetProto(ctx, cacheKey, result)
	return result, err
}

func (c *Cacher[T, K]) Set(ctx context.Context, key K, value T) error {
	cacheKey := c.prefix + ":" + key.GenKey()
	return SetPorto(ctx, cacheKey, value, c.expire)
}

// GetAndDelete 从Redis获取并反序列化JSON对象
func (c *Cacher[T, K]) GetAndDelete(ctx context.Context, key K) (T, error) {
	result := getEntity[T]()
	cacheKey := c.prefix + ":" + key.GenKey()
	err := GetProto(ctx, cacheKey, result)
	if err == nil {
		Client.Del(ctx, cacheKey)
	}
	return result, err
}

// GetWithFallback 使用回源模式获取数据，实现缓存穿透保护
func (c *Cacher[T, K]) GetWithFallback(ctx context.Context, key K, fallback func() (T, error)) (T, error) {
	// 先尝试从缓存获取s
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

type StringKey string

func (s StringKey) GenKey() string {
	return string(s)
}
