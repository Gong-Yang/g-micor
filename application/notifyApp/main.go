package main

import (
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/notifyService"
)

func main() {
	app.Run(&notifyService.Service{})
}
