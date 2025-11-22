package discover

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"log"
	"log/slog"
)

// Grpc 发现服务地址
func Grpc(server string) (c grpc.ClientConnInterface, err error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:///%s", "g-micor", server),
		// 通过服务配置设置负载均衡策略为round_robin
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // 设置初始负载均衡策略
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	return conn, err
}

type ClientService struct {
	UnimplementedClientServer
}

func (c ClientService) Ping(ctx context.Context, req *PingReq) (*PingRes, error) {
	return nil, nil
}

// SubscribeServerRegister 客户端订阅的服务发生了注册或者下线
func (c ClientService) SubscribeServerRegister(ctx context.Context, req *NotifyReq) (*NotifyRes, error) {
	slog.Info("SubscribeServerRegister", "info", req)
	// 触发重新解析
	r, ok := resolverMap[req.Server]
	if !ok {
		slog.Error("server not find", "server", req.Server)
		return nil, nil
	}
	r.updateStates()
	return nil, nil
}
func init() {
	resolver.Register(&resolverBuilder{})
}

var resolverMap = make(map[string]*resolve)
var localPort = ""

var centerClient RegisterClient

func RegisterCenter(addr string, req *RegisterReq) {
	//向注册中心发起注册
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to create grpc client", "error", err)
		panic(err)
	}
	centerClient = NewRegisterClient(conn)
	_, err = centerClient.Register(context.Background(), req)
	if err != nil {
		slog.Error("register error", "error", err)
		panic(err)
	}
	localPort = req.Port
}

type resolve struct {
	target resolver.Target
	cc     resolver.ClientConn
}

func (r *resolve) ResolveNow(options resolver.ResolveNowOptions) {
	r.updateStates()
}

func (r *resolve) Close() {
	return
}
func (r *resolve) updateStates() {
	target := r.target
	endpoint := target.Endpoint()
	discover, err := centerClient.Discover(context.Background(), &Req{
		Port:   localPort,
		Server: endpoint,
	})
	if err != nil {
		slog.Error("discover error", "error", err)
		return
	}
	addrs := make([]resolver.Address, len(discover.Addr))
	for i, addr := range discover.Addr {
		addrs[i] = resolver.Address{Addr: addr}
	}
	r.cc.UpdateState(resolver.State{
		Addresses: addrs,
	})
}

type resolverBuilder struct {
}

func (r *resolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	slog.Info("resolver Build")
	target.Endpoint()
	r2 := &resolve{
		target: target,
		cc:     cc,
	}
	resolverMap[target.Endpoint()] = r2
	r2.updateStates()
	return r2, nil
}

func (r *resolverBuilder) Scheme() string {
	return "g-micor"
}
