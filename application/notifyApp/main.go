package main

import (
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/notifyService"
)

func main() {
	gwAddr := ":1234"
	app.Run(":8001", gwAddr,
		notifyService.Service{})
}
