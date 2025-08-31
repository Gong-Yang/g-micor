package user

import (
	"github.com/Gong-Yang/g-micor/contract/notify_contract"
	"github.com/Gong-Yang/g-micor/contract/user_contract"
	"log/slog"
)

func Register(req *user_contract.RegisterReq) (res *user_contract.RegisterRes, err error) {
	slog.Info("register user", "req", req)

	response, err := notify_contract.SendEmail(&notify_contract.SendEmailRequest{
		Subject: "register",
		To:      "<EMAIL>",
	})
	if err != nil {
		slog.Error("send email error", "error", err)
		return
	}
	slog.Info("send email success", "response", response)
	return &user_contract.RegisterRes{
		Id:   1,
		Name: req.Name,
	}, nil
}
