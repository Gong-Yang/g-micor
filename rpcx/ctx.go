package rpcx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/Gong-Yang/g-micor/logx"
)

type ctxEntry struct {
	key    string
	decode func([]byte) (any, error)
}

var (
	ctxMu      sync.RWMutex
	ctxEntries = map[string]ctxEntry{}
)

func init() {
	// 日志字段（含 traceID）始终随调用传递
	PassCtx[map[string]any](logx.LogAttrsKey)
}

// PassCtx 声明一个随调用传递的上下文字段。key 是 context.WithValue 使用的字符串 key，
// T 是值的类型。未声明的字段不会传给提供方，合并部署和拆分部署一致。
func PassCtx[T any](key string) {
	ctxMu.Lock()
	defer ctxMu.Unlock()
	ctxEntries[key] = ctxEntry{
		key: key,
		decode: func(b []byte) (any, error) {
			var v T
			dec := json.NewDecoder(bytes.NewReader(b))
			dec.UseNumber() // 避免日志里的整数变成科学计数法
			err := dec.Decode(&v)
			return v, err
		},
	}
}

func encodeCtx(ctx context.Context) map[string]json.RawMessage {
	ctxMu.RLock()
	defer ctxMu.RUnlock()
	var res map[string]json.RawMessage
	for key := range ctxEntries {
		v := ctx.Value(key)
		if v == nil {
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			slog.WarnContext(ctx, "rpcx: 上下文字段序列化失败", "key", key, "err", err)
			continue
		}
		if string(b) == "null" {
			continue
		}
		if res == nil {
			res = make(map[string]json.RawMessage)
		}
		res[key] = b
	}
	return res
}

// decodeCtx 构造提供方的 ctx：超时和取消沿用 parent，字符串 key 的值只保留声明过的字段。
func decodeCtx(parent context.Context, raw map[string]json.RawMessage) context.Context {
	values := make(map[string]any, len(raw))
	ctxMu.RLock()
	for key, b := range raw {
		entry, ok := ctxEntries[key]
		if !ok {
			continue
		}
		v, err := entry.decode(b)
		if err != nil {
			slog.WarnContext(parent, "rpcx: 上下文字段解析失败", "key", key, "err", err)
			continue
		}
		values[key] = v
	}
	ctxMu.RUnlock()
	return &providerCtx{Context: parent, values: values}
}

type providerCtx struct {
	context.Context
	values map[string]any
}

func (c *providerCtx) Value(key any) any {
	if k, ok := key.(string); ok {
		return c.values[k]
	}
	// 非字符串 key 是 context 包、gRPC 等内部使用的，交给 parent
	return c.Context.Value(key)
}
