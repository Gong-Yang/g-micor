package app

import (
	"fmt"
	"github.com/Gong-Yang/g-micor/core/config"
	"github.com/Gong-Yang/g-micor/core/logx"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Gong-Yang/g-micor/core/discover"
	"google.golang.org/grpc"
)

type Server interface {
	Init(s grpc.ServiceRegistrar) string
}

func Run(service ...Server) {
	// 初始化配置
	initConf()
	// 初始化日志
	initLog()

	conf := Conf.App
	addr := fmt.Sprintf(":%v", conf.Port)
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

func initConf() {
	Conf = &Config{}
	config.Init(Conf)
}

func initLog() {
	//初始化日志
	// 创建OpenObserve日志选项，只配置必要参数
	conf := Conf.Observe
	opts := logx.OpenObserveOptions{
		Endpoint:      conf.Endpoint,
		Organization:  conf.Organization,
		Stream:        conf.Stream,
		Username:      conf.Username,
		Password:      conf.Password,
		Timeout:       5 * time.Second,
		FlushInterval: 2 * time.Second,
		Handler:       slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	}
	// OpenObserve处理器
	openObserveHandler := logx.NewOpenObserveHandler(opts, slog.LevelWarn)
	// 创建logger
	logger := slog.New(openObserveHandler)
	// 设置为默认logger
	slog.SetDefault(logger)
	slog.Info("log init complete")
}
