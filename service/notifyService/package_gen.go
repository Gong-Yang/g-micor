package notifyService

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/notify"
	"google.golang.org/grpc"
)

type notifyLocalClient struct {
	server *Service
}

func (n *notifyLocalClient) SendEmail(ctx context.Context, in *notify.SendEmailRequest, opts ...grpc.CallOption) (*notify.SendEmailResponse, error) {
	return n.server.SendEmail(ctx, in)
}

func (n *Service) Init(s grpc.ServiceRegistrar) string {
	notify.Client = &notifyLocalClient{server: n} // 本地直接调
	notify.RegisterNotifyServer(s, n)             // 将服务注册
	return "notify"                               // 服务名称
}
