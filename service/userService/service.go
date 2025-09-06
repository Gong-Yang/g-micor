package userService

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/notify"
	"github.com/Gong-Yang/g-micor/contract/user"
	"log/slog"
)

type Service struct {
	user.UnimplementedUserServer
}

func (s *Service) Register(ctx context.Context, req *user.RegisterReq) (*user.RegisterRes, error) {
	slog.Info("register user", "req", req)

	response, err := notify.Client.SendEmail(ctx, &notify.SendEmailRequest{
		Subject: "register",
		To:      "<EMAIL>",
	})
	if err != nil {
		slog.Error("send email error", "error", err)
		return nil, err
	}
	slog.Info("send email success", "response", response)
	return &user.RegisterRes{
		Id:   1,
		Name: req.Name,
	}, nil
}
