package order

import (
	"github.com/Gong-Yang/g-micor/contract/notify_contract"
	"github.com/Gong-Yang/g-micor/contract/order_contract"
	"log/slog"
)

func Create(req *order_contract.CreateReq) (res *order_contract.CreateRes, err error) {
	slog.Info("order create", "req", req)
	notify_contract.SendEmail(&notify_contract.SendEmailRequest{
		Subject: "订单创建成功",
		To:      "<EMAIL2>",
	})
	return &order_contract.CreateRes{
		OrderId: 1,
	}, nil
}
