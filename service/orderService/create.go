package orderService

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/notify"
	"github.com/Gong-Yang/g-micor/contract/order"
	"log/slog"
)

type Service struct {
	order.UnimplementedOrderServer
}

func (s Service) Create(ctx context.Context, req *order.CreateReq) (*order.CreateRes, error) {
	slog.Info("order create", "req", req)
	email, err := notify.Client.SendEmail(ctx, &notify.SendEmailRequest{
		Subject: "订单创建成功",
		To:      "<EMAIL2>",
	})
	slog.Info("email", "res", email, "err", err)
	return &order.CreateRes{
		OrderId: 1,
	}, nil
}
