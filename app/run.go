package app

import (
	"os"
	"reflect"
	"sync"

	"github.com/Gong-Yang/g-micor/config"
	"github.com/Gong-Yang/g-micor/mongox"
	"github.com/Gong-Yang/g-micor/pgsql"
	"github.com/Gong-Yang/g-micor/redisx"
	"github.com/Gong-Yang/g-micor/rpcx"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Module interface {
	// Init 模块初始化：注册 rpcx 方法实现、启动后台任务等。
	// 各模块的 Init 按 app.Run 的参数顺序执行，此时服务发现还未就绪，
	// 不要在 Init 里同步调用其他模块的契约方法。
	Init()
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
	if Conf.Mongo.Uri != "" {
		err := mongox.InitDB(Conf.Mongo.Uri, Conf.Mongo.Database)
		if err != nil {
			panic(err)
		}
	}
	if Conf.PGSQL.Uri != "" {
		err := pgsql.Init(Conf.PGSQL.Uri)
		if err != nil {
			panic(err)
		}
	}
	// 初始化Redis
	redisConf := Conf.Redis
	redisx.Init(Hostname, &redis.Options{Addr: redisConf.Addr, Password: redisConf.Password, DB: redisConf.Db})
	// 远程调用凭证
	rpcx.SetAuthKey(Conf.App.HmacKey)
	// 初始化模块：先于 HTTP 和 RPC 启动，保证请求进来时本地实现都已注册
	for _, module := range modules {
		module.Init()
	}
	// 初始化web
	webStart(wg, modules)
	// 初始化RPC
	rpcStart(wg)
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
