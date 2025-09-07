package app

import (
	"github.com/Gong-Yang/g-micor/core/discover"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"sync"
)

type WebServer interface {
	Init(s grpc.ServiceRegistrar) string
}

func RunWeb(addr string, gwAddr string, service ...WebServer) {
	// 初始化配置
	initConf()
	// 初始化日志
	initLog()

	engine := gin.Default()
	// 监听
	listener, _ := net.Listen("tcp", addr)
	// 初始化服务
	rpcApp := grpc.NewServer()
	// 注册中心的客户端服务
	discover.RegisterClientServer(rpcApp, discover.ClientService{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err := rpcApp.Serve(listener)
		if err != nil {
			slog.Error("TODO")
		}
		wg.Done()
	}()
	go func() {
		engine.Run()
	}()

	wg.Wait()
}
