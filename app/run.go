package app

import (
	"os"
	"sync"

	"github.com/Gong-Yang/g-micor/config"
	"github.com/Gong-Yang/g-micor/mongox"
	"github.com/Gong-Yang/g-micor/redisx"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type Module interface {
	Init(s grpc.ServiceRegistrar) string
	Router(router gin.IRouter)
}

func Run(modules ...Module) {
	var wg = &sync.WaitGroup{}
	// 初始化配置
	InitConf()
	var Hostname, _ = os.Hostname()
	Hostname = Conf.App.Name + ":" + Hostname
	// 初始化日志
	initLog()
	// 初始化mongo
	err := mongox.InitDB(Conf.Mongo.Uri, Conf.Mongo.Database)
	if err != nil {
		panic(err)
	}
	// 初始化Redis
	redisConf := Conf.Redis
	redisx.Init(Hostname, &redis.Options{Addr: redisConf.Addr, Password: redisConf.Password, DB: redisConf.Db})
	// 初始化web
	webStart(wg, modules)
	// 初始化RPC
	rpcStart(wg, modules)
	wg.Wait()
}

func InitConf() {
	Conf = &Config{}
	config.Init(Conf, "")
}

func TestInit(workDir string) {
	// 初始化配置
	Conf = &Config{}
	config.Init(Conf, workDir)
	var Hostname, _ = os.Hostname()
	Hostname = Conf.App.Name + ":" + Hostname
	// 初始化mongo
	err := mongox.InitDB(Conf.Mongo.Uri, Conf.Mongo.Database)
	if err != nil {
		panic(err)
	}
	// 初始化Redis
	redisConf := Conf.Redis
	redisx.Init(Hostname, &redis.Options{Addr: redisConf.Addr, Password: redisConf.Password, DB: redisConf.Db})
}
