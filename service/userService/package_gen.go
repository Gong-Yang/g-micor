package userService

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/user"
	"google.golang.org/grpc"
)

type userLocalClient struct {
	server *Service
}

func (n *userLocalClient) Register(ctx context.Context, in *user.RegisterReq, opts ...grpc.CallOption) (*user.RegisterRes, error) {
	return n.server.Register(ctx, in)
}

func (n *Service) Init(s grpc.ServiceRegistrar) string {
	user.Client = &userLocalClient{server: n} // 本地直接调
	user.RegisterUserServer(s, n)             // 将服务注册
	return "user"                             // 服务名称
}
