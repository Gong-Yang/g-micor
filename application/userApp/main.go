package main

import (
	"github.com/Gong-Yang/g-micor/core/app"
	"github.com/Gong-Yang/g-micor/service/userService"
)

func main() {
	app.Run(&userService.Service{})
}
