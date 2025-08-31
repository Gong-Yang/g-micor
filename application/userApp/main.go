package main

import (
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/notify"
	"github.com/Gong-Yang/g-micor/service/user"
)

func main() {

	//go func() {
	//	for {
	//		time.Sleep(time.Second * 10)
	//		slog.Info(">>>>>>>>>>>>>>>>start")
	//		res, err := user.Register(&user_contract.RegisterReq{Name: "张三"})
	//		slog.Info("register complete", "res", res, "err", err)
	//	}
	//}()

	app.Run(":8000", ":1234",
		user.Service{},
		notify.Service{})
}
