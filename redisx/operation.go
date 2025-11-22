package redisx

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
)

// SetSimple 存储简单对象（string、int/uint、float、bool）
func SetSimple[T comparable](ctx context.Context, key string, value T, expire time.Duration) error {
	var s string
	switch v := any(value).(type) {
	case string:
		s = v
	case int:
		s = strconv.FormatInt(int64(v), 10)
	case int8:
		s = strconv.FormatInt(int64(v), 10)
	case int16:
		s = strconv.FormatInt(int64(v), 10)
	case int32:
		s = strconv.FormatInt(int64(v), 10)
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint:
		s = strconv.FormatUint(uint64(v), 10)
	case uint8:
		s = strconv.FormatUint(uint64(v), 10)
	case uint16:
		s = strconv.FormatUint(uint64(v), 10)
	case uint32:
		s = strconv.FormatUint(uint64(v), 10)
	case uint64:
		s = strconv.FormatUint(v, 10)
	case float32:
		s = strconv.FormatFloat(float64(v), 'g', -1, 32)
	case float64:
		s = strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		s = strconv.FormatBool(v)
	default:
		slog.InfoContext(ctx, "SetSimple unsupported type")
		return errors.New("SetSimple unsupported type")
	}
	if err := Client.Set(ctx, key, s, expire).Err(); err != nil {
		slog.InfoContext(ctx, "SetSimple redis set error", "err", err)
		return err
	}
	return nil
}

// GetSimple 获取简单对象（string、int/uint、float、bool）
func GetSimple[T comparable](ctx context.Context, key string) (T, error) {
	var result T
	s, err := Client.Get(ctx, key).Result()
	if err != nil {
		return result, err
	}
	switch any(result).(type) {
	case string:
		return any(s).(T), nil
	case int:
		x, err := strconv.ParseInt(s, 10, strconv.IntSize)
		if err != nil {
			return result, err
		}
		v := int(x)
		return any(v).(T), nil
	case int8:
		x, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			return result, err
		}
		v := int8(x)
		return any(v).(T), nil
	case int16:
		x, err := strconv.ParseInt(s, 10, 16)
		if err != nil {
			return result, err
		}
		v := int16(x)
		return any(v).(T), nil
	case int32:
		x, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return result, err
		}
		v := int32(x)
		return any(v).(T), nil
	case int64:
		x, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return result, err
		}
		v := int64(x)
		return any(v).(T), nil
	case uint:
		x, err := strconv.ParseUint(s, 10, strconv.IntSize)
		if err != nil {
			return result, err
		}
		v := uint(x)
		return any(v).(T), nil
	case uint8:
		x, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			return result, err
		}
		v := uint8(x)
		return any(v).(T), nil
	case uint16:
		x, err := strconv.ParseUint(s, 10, 16)
		if err != nil {
			return result, err
		}
		v := uint16(x)
		return any(v).(T), nil
	case uint32:
		x, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return result, err
		}
		v := uint32(x)
		return any(v).(T), nil
	case uint64:
		x, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return result, err
		}
		v := uint64(x)
		return any(v).(T), nil
	case float32:
		x, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return result, err
		}
		v := float32(x)
		return any(v).(T), nil
	case float64:
		x, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return result, err
		}
		v := float64(x)
		return any(v).(T), nil
	case bool:
		x, err := strconv.ParseBool(s)
		if err != nil {
			return result, err
		}
		v := bool(x)
		return any(v).(T), nil
	default:
		return result, errors.New("GetSimple unsupported type")
	}
}

// SetJSON 将任意对象序列化为 JSON 并存入 Redis
func SetJSON(ctx context.Context, key string, value any, expire time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		slog.InfoContext(ctx, "SetJSON marshal error", "err", err)
		return err
	}
	err = Client.Set(ctx, key, data, expire).Err()
	if err != nil {
		slog.InfoContext(ctx, "SetJSON redis set error", "err", err)
		return err
	}
	return nil
}

// GetJSON 泛型方法：从 Redis 获取并反序列化 JSON 对象
func GetJSON[T any](ctx context.Context, key string, out T) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}

// SetPorto 将 proto.Message 对象序列化为字节数组并保存到 Redis
func SetPorto(ctx context.Context, key string, value proto.Message, expire time.Duration) error {
	byteArr, err := proto.Marshal(value)
	if err != nil {
		slog.InfoContext(ctx, "TODO")
		return err
	}

	err = Client.Set(ctx, key, byteArr, expire).Err()
	if err != nil {
		// TODO
		return err
	}
	return nil
}

// GetProto 泛型方法：从 Redis 获取并反序列化 proto.Message
func GetProto[T proto.Message](ctx context.Context, key string, out T) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}
