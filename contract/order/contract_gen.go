package order

import (
	"context"
	"github.com/Gong-Yang/g-micor/core/discover"
	"google.golang.org/grpc"
	"log/slog"
)

var Client OrderClient = &orderRemoteClient{}

type orderRemoteClient struct {
	client OrderClient
}

func (n *orderRemoteClient) init() error {
	c, err := discover.Grpc("order")
	if err != nil {
		return err
	}
	client := NewOrderClient(c)
	n.client = client
	slog.Info("order remote client init")
	return nil
}
func (n *orderRemoteClient) Create(ctx context.Context, in *CreateReq, opts ...grpc.CallOption) (*CreateRes, error) {
	if n.client == nil {
		err := n.init()
		if err != nil {
			return nil, err
		}
	}
	return n.client.Create(ctx, in, opts...)
}
