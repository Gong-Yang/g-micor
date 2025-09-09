package app

import (
	"github.com/Gong-Yang/g-micor/core/config"
	"github.com/Gong-Yang/g-micor/core/mongox"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"sync"
)

type Server interface {
	Init(s grpc.ServiceRegistrar) string
	Router(router gin.IRouter)
}

func Run(service ...Server) {
	var wg = &sync.WaitGroup{}
	// 初始化配置
	initConf()
	// 初始化日志
	initLog()
	// 初始化mongo
	err := mongox.InitDB(Conf.Mongo.Uri, Conf.Mongo.Database)
	if err != nil {
		panic(err)
	}

	// 初始化RPC
	rpcStart(wg, service)

	// 初始化web
	webStart(wg, service)

	wg.Wait()
}

func initConf() {
	Conf = &Config{}
	config.Init(Conf)
}
