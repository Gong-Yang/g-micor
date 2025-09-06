package main

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/order"
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/orderService"
	"log/slog"
	"time"
)

func main() {
	gwAddr := ":1234"
	go func() {
		for {
			time.Sleep(time.Second * 10)
			res, err := order.Client.Create(context.Background(), &order.CreateReq{
				GoodsId: 1,
				UserId:  1,
			})
			slog.Info("order create complete", "res", res, "err", err)
		}

	}()
	app.Run(":8002", gwAddr,
		orderService.Service{})
}
