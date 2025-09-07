package notify

import (
	"context"
	"github.com/Gong-Yang/g-micor/core/discover"
	"google.golang.org/grpc"
	"log/slog"
)

var Client NotifyClient = &notifyRemoteClient{}

type notifyRemoteClient struct {
	client NotifyClient
}

func (n *notifyRemoteClient) init() error {
	c, err := discover.Grpc("notify")
	if err != nil {
		return err
	}
	client := NewNotifyClient(c)
	n.client = client
	slog.Info("notify remote client init")
	return nil
}

func (n *notifyRemoteClient) SendEmail(ctx context.Context, in *SendEmailRequest, opts ...grpc.CallOption) (*SendEmailResponse, error) {
	if n.client == nil {
		err := n.init()
		if err != nil {
			return nil, err
		}
	}
	return n.client.SendEmail(ctx, in, opts...)
}
