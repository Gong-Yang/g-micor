package redisx

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

func Init(consumerName string, opt *redis.Options) {
	Client = redis.NewClient(opt)
	ping := Client.Ping(context.Background())
	if ping.Err() != nil {
		panic(ping.Err())
	}
	for _, initialize := range initList {
		initialize.init()
	}
	slog.Info("Redis连接成功")
	ConsumerName = consumerName
}

var initList []Initialize

type Initialize interface {
	init()
}
