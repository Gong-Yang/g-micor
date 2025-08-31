package main

import (
	"github.com/Gong-Yang/g-micor/contract/order_contract"
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/order"
	"log/slog"
	"time"
)

func main() {
	gwAddr := ":1234"
	go func() {
		for {
			time.Sleep(time.Second * 10)
			res, err := order.Create(&order_contract.CreateReq{
				GoodsId: 1,
				UserId:  1,
			})
			slog.Info("order create complete", "res", res, "err", err)
		}

	}()
	app.Run(":8002", gwAddr,
		order.Service{})
}
