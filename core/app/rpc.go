package app

import (
	"fmt"
	"github.com/Gong-Yang/g-micor/core/discover"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"sync"
)

func rpcStart(wg *sync.WaitGroup, service []Server) {
	conf := Conf.App
	addr := fmt.Sprintf(":%v", conf.RpcPort)
	gwAddr := conf.CenterAddr
	// 监听
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
	wg.Add(1)
	go func() {
		err := rpcApp.Serve(listener)
		if err != nil {
			slog.Error("rpc start error", "error", err)
		}
		wg.Done()
	}()

	//向注册中心发起注册
	discover.RegisterCenter(gwAddr, &discover.RegisterReq{
		Port:    addr,
		Servers: ss,
	})
	slog.Info("register success", "servers", ss)
}
