// Package rpcx 模块间调用：同一份代码，合并部署时本地调用，拆分部署时远程调用。
//
// 契约里定义方法：  var FindById = rpcx.Method[FileId, FileInfo]("file", "FindById")
// 提供方注册实现：  fileC.FindById.Impl(service.FindById)
// 调用方发起调用：  info, err := fileC.FindById.Call(ctx, &fileC.FileId{Id: id})
package rpcx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/Gong-Yang/g-micor/errorx"
	"github.com/Gong-Yang/g-micor/logx"
)

// Empty 不需要返回值时使用。
type Empty struct{}

// ErrSystem 提供方返回非业务错误，或调用链路异常时，调用方收到的统一错误。
var ErrSystem = errorx.ErrorCode{Code: errorx.RespErr, Msg: "请联系管理员"}

// M 一个契约方法。Req、Res 是请求和响应的结构体类型（不带指针）。
type M[Req, Res any] struct {
	service string
	name    string // service.method
}

type handler func(ctx context.Context, body []byte) ([]byte, error)

var (
	mu       sync.RWMutex
	defined  = map[string]struct{}{}
	handlers = map[string]handler{}
	services = map[string]struct{}{}
)

// Method 定义契约方法，在契约包里用包级变量声明。service 是服务名，用于服务发现。
func Method[Req, Res any](service, method string) *M[Req, Res] {
	name := service + "." + method
	mu.Lock()
	defer mu.Unlock()
	if _, ok := defined[name]; ok {
		panic("rpcx: 方法重复定义 " + name)
	}
	defined[name] = struct{}{}
	return &M[Req, Res]{service: service, name: name}
}

// Impl 注册方法的实现，在提供方模块的 Init 中调用。
func (m *M[Req, Res]) Impl(fn func(context.Context, *Req) (*Res, error)) {
	h := func(ctx context.Context, body []byte) ([]byte, error) {
		req := new(Req)
		if err := json.Unmarshal(body, req); err != nil {
			return nil, fmt.Errorf("rpcx: 解析请求失败: %w", err)
		}
		res, err := fn(ctx, req)
		if err != nil {
			return nil, err
		}
		return json.Marshal(res)
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := handlers[m.name]; ok {
		panic("rpcx: 方法重复实现 " + m.name)
	}
	handlers[m.name] = h
	services[m.service] = struct{}{}
}

// Call 调用方法。本进程注册了实现就本地调用，否则通过服务发现远程调用。
// 两种方式都会对请求、响应、上下文做一次 JSON 编解码，行为一致。
func (m *M[Req, Res]) Call(ctx context.Context, req *Req) (*Res, error) {
	body, err := json.Marshal(req)
	if err != nil {
		slog.ErrorContext(ctx, "rpcx: 请求序列化失败", "method", m.name, "err", err)
		return nil, ErrSystem
	}
	in := &wireReq{Method: m.name, Body: body, Ctx: encodeCtx(ctx)}

	var out *wireRes
	if getHandler(m.name) != nil {
		out, err = callLocal(ctx, in)
	} else {
		out, err = callRemote(ctx, m.service, in)
	}
	if err != nil {
		slog.ErrorContext(ctx, "rpcx: 调用失败", "method", m.name, "err", err)
		return nil, ErrSystem
	}
	if out.Err != nil {
		return nil, *out.Err
	}
	if len(out.Body) == 0 || string(out.Body) == "null" {
		return nil, nil
	}
	res := new(Res)
	if err := json.Unmarshal(out.Body, res); err != nil {
		slog.ErrorContext(ctx, "rpcx: 解析响应失败", "method", m.name, "err", err)
		return nil, ErrSystem
	}
	return res, nil
}

// Services 返回本进程提供实现的服务名，用于注册到服务发现中心。
func Services() []string {
	mu.RLock()
	defer mu.RUnlock()
	res := make([]string, 0, len(services))
	for s := range services {
		res = append(res, s)
	}
	sort.Strings(res)
	return res
}

func getHandler(name string) handler {
	mu.RLock()
	defer mu.RUnlock()
	return handlers[name]
}

// serve 提供方执行入口，本地调用和远程调用都走这里。
func serve(parent context.Context, in *wireReq) (out *wireRes) {
	ctx := decodeCtx(parent, in.Ctx)
	h := getHandler(in.Method)
	if h == nil {
		slog.ErrorContext(ctx, "rpcx: 方法未实现", "method", in.Method)
		return sysErrRes()
	}
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "rpcx: 提供方 panic", "method", in.Method, "err", r, "stack", logx.CleanStack(2))
			out = sysErrRes()
		}
	}()
	body, err := h(ctx, in.Body)
	if err != nil {
		if code, ok := asErrorCode(err); ok {
			return &wireRes{Err: &code}
		}
		slog.ErrorContext(ctx, "rpcx: 提供方返回系统错误", "method", in.Method, "err", err)
		return sysErrRes()
	}
	return &wireRes{Body: body}
}

func sysErrRes() *wireRes {
	e := ErrSystem
	return &wireRes{Err: &e}
}

func asErrorCode(err error) (errorx.ErrorCode, bool) {
	var code errorx.ErrorCode
	if errors.As(err, &code) {
		return code, true
	}
	var codePtr *errorx.ErrorCode
	if errors.As(err, &codePtr) && codePtr != nil {
		return *codePtr, true
	}
	return code, false
}
