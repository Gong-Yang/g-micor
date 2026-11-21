package app

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/Gong-Yang/g-micor/ginx"
	"github.com/gin-gonic/gin"
)

func webStart(wg *sync.WaitGroup, service []Module) {
	conf := Conf.App
	addr := fmt.Sprintf(":%v", conf.Port)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(ginx.BasicMiddleware)
	// 路由注册放在主协程，处理函数签名写错时直接启动失败
	for _, server := range service {
		server.Router(engine)
	}
	// 监听
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := engine.Run(addr)
		if err != nil {
			slog.Error("gin run error", "error", err)
		}
	}()
}
