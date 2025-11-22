package app

import (
	"fmt"
	"github.com/Gong-Yang/g-micor/ginx"
	"github.com/gin-gonic/gin"
	"log/slog"
	"sync"
)

func webStart(wg *sync.WaitGroup, service []Module) {
	conf := Conf.App
	addr := fmt.Sprintf(":%v", conf.Port)
	// 监听
	wg.Add(1)
	go func() {
		defer wg.Done()
		engine := gin.Default()
		engine.Use(ginx.BasicMiddleware)
		engine.Use()
		for _, server := range service {
			server.Router(engine)
		}
		err := engine.Run(addr)
		if err != nil {
			slog.Error("gin run error", "error", err)
		}
	}()
}
