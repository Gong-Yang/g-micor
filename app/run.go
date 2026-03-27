package app

import (
	"os"
	"reflect"
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
	Config() any
}

func Run(modules ...Module) {
	var wg = &sync.WaitGroup{}
	// 初始化配置
	InitConf(modules)
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

func InitConf(modules []Module) {
	Conf = &Config{}
	var conf = make([]any, 0, len(modules)+1)
	conf = append(conf, Conf)
	for _, module := range modules {
		configItem := module.Config()
		if configItem == nil {
			continue
		}
		rv := reflect.ValueOf(configItem)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			panic("config item must be a pointer")
		}
		conf = append(conf, configItem)
	}
	config.Init(conf)
}
