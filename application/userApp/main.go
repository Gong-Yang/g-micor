package main

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/user"
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/notifyService"
	"github.com/Gong-Yang/g-micor/service/userService"
	"log/slog"
	"time"
)

func main() {

	go func() {
		for {
			time.Sleep(time.Second * 10)
			slog.Info(">>>>>>>>>>>>>>>>start")
			res, err := user.Client.Register(context.Background(), &user.RegisterReq{Name: "张三"})
			slog.Info("register complete", "res", res, "err", err)
		}
	}()

	app.Run(":8000", ":1234",
		userService.Service{},
		notifyService.Service{})
}
