package app

import (
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/Gong-Yang/g-micor/discover"
	"github.com/Gong-Yang/g-micor/rpcx"
	"google.golang.org/grpc"
)

func rpcStart(wg *sync.WaitGroup) {
	conf := Conf.App
	addr := fmt.Sprintf(":%v", conf.RpcPort)
	gwAddr := conf.CenterAddr
	// 监听
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Errorf("rpc listen %s: %w", addr, err))
	}
	// 初始化服务
	rpcApp := grpc.NewServer()
	rpcx.Register(rpcApp)
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
	ss := rpcx.Services()
	discover.RegisterCenter(gwAddr, &discover.RegisterReq{
		Port:    addr,
		Servers: ss,
	})
	slog.Info("register success", "servers", ss)
}
