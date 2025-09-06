package userService

import (
	"log/slog"
)

func Register(req *user.RegisterReq) (res *user.RegisterRes, err error) {
	slog.Info("register user", "req", req)

	response, err := notify.SendEmail(&notify.SendEmailRequest{
		Subject: "register",
		To:      "<EMAIL>",
	})
	if err != nil {
		slog.Error("send email error", "error", err)
		return
	}
	slog.Info("send email success", "response", response)
	return &user.RegisterRes{
		Id:   1,
		Name: req.Name,
	}, nil
}
