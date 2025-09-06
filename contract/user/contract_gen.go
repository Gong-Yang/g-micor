package user

import (
	"context"
	"github.com/Gong-Yang/g-micor/core/discover"
	"google.golang.org/grpc"
	"log/slog"
)

var Client UserClient = &userRemoteClient{}

type userRemoteClient struct {
	client UserClient
}

func (n *userRemoteClient) init() error {
	c, err := discover.Grpc("user")
	if err != nil {
		return err
	}
	client := NewUserClient(c)
	n.client = client
	slog.Info("user remote client init")
	return nil
}
func (n *userRemoteClient) Register(ctx context.Context, in *RegisterReq, opts ...grpc.CallOption) (*RegisterRes, error) {
	if n.client == nil {
		err := n.init()
		if err != nil {
			return nil, err
		}
	}
	return n.client.Register(ctx, in, opts...)
}
