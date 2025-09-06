package orderService

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/order"
	"google.golang.org/grpc"
)

type orderLocalClient struct {
	server Service
}

func (n orderLocalClient) Create(ctx context.Context, in *order.CreateReq, opts ...grpc.CallOption) (*order.CreateRes, error) {
	return n.server.Create(ctx, in)
}

func (n Service) Init(s grpc.ServiceRegistrar) string {
	order.Client = orderLocalClient{server: n} // 本地直接调
	order.RegisterOrderServer(s, n)            // 将服务注册
	return "order"                             // 服务名称
}
