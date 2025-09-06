package app

import (
	"log/slog"
	"net"
	"sync"

	"github.com/Gong-Yang/g-micor/core/discover"
	"google.golang.org/grpc"
)

type Server interface {
	Init(s grpc.ServiceRegistrar) string
}

func Run(addr string, gwAddr string, service ...Server) {
	listener, _ := net.Listen("tcp", addr)
	// 初始化服务
	rpcApp := grpc.NewServer()
	var ss []string
	for _, s := range service {
		serviceName := s.Init(rpcApp)
		slog.Info("register service", "service", serviceName)
		ss = append(ss, serviceName)
	}
	// 注册中心的客户端服务
	discover.RegisterClientServer(rpcApp, discover.ClientService{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := rpcApp.Serve(listener)
		if err != nil {
			slog.Error("TODO")
		}
		wg.Done()
	}()

	//向注册中心发起注册
	discover.RegisterCenter(gwAddr, &discover.RegisterReq{
		Port:    addr,
		Servers: ss,
	})
	slog.Info("register success", "servers", ss)
	wg.Wait()
}
