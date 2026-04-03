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

// convertSimpleValue 将简单类型转换为 string
func convertSimpleValue[T comparable](value T) (string, error) {
	switch v := any(value).(type) {
	case string:
		return v, nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", errors.New("unsupported simple type")
	}
}

// parseSimpleValue 将 string 转换为目标简单类型
func parseSimpleValue[T comparable](s string) (T, error) {
	var result T
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

// SetSimple 存储简单对象（string、int/uint、float、bool）
func SetSimple[T comparable](ctx context.Context, key string, value T, expire time.Duration) error {
	s, err := convertSimpleValue(value)
	if err != nil {
		slog.InfoContext(ctx, "SetSimple unsupported type", "err", err)
		return err
	}
	if err := Client.Set(ctx, key, s, expire).Err(); err != nil {
		slog.InfoContext(ctx, "SetSimple redis set error", "err", err)
		return err
	}
	return nil
}

// GetSimple 获取简单对象（string、int/uint、float、bool）
func GetSimple[T comparable](ctx context.Context, key string) (T, error) {
	s, err := Client.Get(ctx, key).Result()
	if err != nil {
		var res T
		return res, err
	}
	return parseSimpleValue[T](s)
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
		slog.ErrorContext(ctx, "SetProto marshal error", "err", err, "key", key)
		return err
	}

	err = Client.Set(ctx, key, byteArr, expire).Err()
	if err != nil {
		slog.ErrorContext(ctx, "SetProto redis set error", "err", err, "key", key)
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

// SetNXSimple 存储简单对象（string、int/uint、float、bool），仅当 key 不存在时
func SetNXSimple[T comparable](ctx context.Context, key string, value T, expire time.Duration) (bool, error) {
	s, err := convertSimpleValue(value)
	if err != nil {
		slog.InfoContext(ctx, "SetNXSimple unsupported type", "err", err)
		return false, err
	}
	result, err := Client.SetNX(ctx, key, s, expire).Result()
	if err != nil {
		slog.InfoContext(ctx, "SetNXSimple redis set error", "err", err)
		return false, err
	}
	return result, nil
}

// SetNXJSON 将任意对象序列化为 JSON 并存入 Redis，仅当 key 不存在时
func SetNXJSON(ctx context.Context, key string, value any, expire time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		slog.InfoContext(ctx, "SetNXJSON marshal error", "err", err)
		return false, err
	}
	result, err := Client.SetNX(ctx, key, data, expire).Result()
	if err != nil {
		slog.InfoContext(ctx, "SetNXJSON redis set error", "err", err)
		return false, err
	}
	return result, nil
}

// SetNXProto 将 proto.Message 对象序列化为字节数组并保存到 Redis，仅当 key 不存在时
func SetNXProto(ctx context.Context, key string, value proto.Message, expire time.Duration) (bool, error) {
	byteArr, err := proto.Marshal(value)
	if err != nil {
		slog.ErrorContext(ctx, "SetNXProto marshal error", "err", err, "key", key)
		return false, err
	}

	result, err := Client.SetNX(ctx, key, byteArr, expire).Result()
	if err != nil {
		slog.ErrorContext(ctx, "SetNXProto redis set error", "err", err, "key", key)
		return false, err
	}
	return result, nil
}

// Delete 删除一个或多个 key，返回删除的数量
func Delete(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	count, err := Client.Del(ctx, keys...).Result()
	if err != nil {
		slog.InfoContext(ctx, "Delete redis del error", "err", err, "keys", keys)
		return 0, err
	}
	return count, nil
}
