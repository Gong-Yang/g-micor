package notifyService

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/notify"
	"github.com/gin-gonic/gin"
	"log/slog"
)

type Service struct {
	notify.UnimplementedNotifyServer
}

func (n *Service) Router(router gin.IRouter) {
	return
}

func (n *Service) SendEmail(ctx context.Context, req *notify.SendEmailRequest) (*notify.SendEmailResponse, error) {
	slog.Info("send email", "req", req)
	return &notify.SendEmailResponse{
		Code:    201,
		Message: "success",
	}, nil
}
