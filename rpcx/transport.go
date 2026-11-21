package rpcx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/Gong-Yang/g-micor/discover"
	"github.com/Gong-Yang/g-micor/errorx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// 远程调用复用 gRPC 和 discover，整个框架只有一个通用方法，消息用 JSON 编码。

const (
	codecName  = "rpcxjson"
	fullMethod = "/gmicor.rpcx/Call"
	authHeader = "x-rpcx-auth"
)

// authToken 远程调用凭证，由同一套部署共享的密钥派生，防止能连上 RPC 端口的人直接调用契约方法。
var authToken string

func init() {
	SetAuthKey("")
}

// SetAuthKey 设置远程调用凭证的密钥，app 启动时传入 hmacKey。同一套部署的各进程必须一致。
func SetAuthKey(key string) {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte("g-micor/rpcx"))
	authToken = hex.EncodeToString(h.Sum(nil))
}

func authorized(ctx context.Context) bool {
	md, _ := metadata.FromIncomingContext(ctx)
	vals := md.Get(authHeader)
	return len(vals) == 1 && hmac.Equal([]byte(vals[0]), []byte(authToken))
}

type wireReq struct {
	Method string                     `json:"m"`
	Body   json.RawMessage            `json:"b,omitempty"`
	Ctx    map[string]json.RawMessage `json:"c,omitempty"`
}

type wireRes struct {
	Body json.RawMessage   `json:"b,omitempty"`
	Err  *errorx.ErrorCode `json:"e,omitempty"`
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (jsonCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (jsonCodec) Name() string                       { return codecName }

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

var serviceDesc = grpc.ServiceDesc{
	ServiceName: "gmicor.rpcx",
	HandlerType: (*any)(nil),
	Methods: []grpc.MethodDesc{{
		MethodName: "Call",
		Handler: func(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
			if !authorized(ctx) {
				return nil, status.Error(codes.Unauthenticated, "rpcx: 调用凭证无效")
			}
			in := new(wireReq)
			if err := dec(in); err != nil {
				return nil, err
			}
			return serve(ctx, in), nil
		},
	}},
	Metadata: "rpcx",
}

// Register 把通用调用服务注册到 gRPC server，由 app 启动时调用。
func Register(s grpc.ServiceRegistrar) {
	s.RegisterService(&serviceDesc, struct{}{})
}

// callLocal 本地调用。响应也做一次 JSON 编解码，保证错误里的 Data 等字段和远程调用一致。
func callLocal(ctx context.Context, in *wireReq) (*wireRes, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	req := new(wireReq)
	if err = json.Unmarshal(b, req); err != nil {
		return nil, err
	}
	b, err = json.Marshal(serve(ctx, req))
	if err != nil {
		return nil, err
	}
	out := new(wireRes)
	err = json.Unmarshal(b, out)
	return out, err
}

var (
	connMu sync.Mutex
	conns  = map[string]grpc.ClientConnInterface{}
)

func getConn(service string) (grpc.ClientConnInterface, error) {
	connMu.Lock()
	defer connMu.Unlock()
	if c, ok := conns[service]; ok {
		return c, nil
	}
	c, err := discover.Grpc(service)
	if err != nil {
		return nil, err
	}
	conns[service] = c
	return c, nil
}

func callRemote(ctx context.Context, service string, in *wireReq) (*wireRes, error) {
	c, err := getConn(service)
	if err != nil {
		return nil, err
	}
	out := new(wireRes)
	ctx = metadata.AppendToOutgoingContext(ctx, authHeader, authToken)
	err = c.Invoke(ctx, fullMethod, in, out, grpc.CallContentSubtype(codecName))
	return out, err
}
