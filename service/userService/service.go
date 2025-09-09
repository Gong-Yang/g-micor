package userService

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/user"
	"github.com/Gong-Yang/g-micor/core/mongox"
	"github.com/Gong-Yang/g-micor/store"
	"log/slog"
)

type Service struct {
	user.UnimplementedUserServer
}

func (s *Service) Register(ctx context.Context, req *user.RegisterReq) (*user.RegisterRes, error) {
	slog.Info("register user", "req", req)

	user_ := &store.User{
		Base:     &mongox.Base{},
		UserName: req.Name,
	}
	_, err := store.UserStore.InsertOne(ctx, user_)
	if err != nil {
		slog.Error("insert user error", "error", err)
		return nil, err
	}

	return &user.RegisterRes{
		Id:   user_.Id,
		Name: req.Name,
	}, nil
}
