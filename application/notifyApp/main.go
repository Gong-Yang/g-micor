package main

import (
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/notify"
)

func main() {
	gwAddr := ":1234"
	app.Run(":8001", gwAddr,
		notify.Service{})
}
