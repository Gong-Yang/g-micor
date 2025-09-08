package app

import (
	"github.com/Gong-Yang/g-micor/core/logx"
	"log/slog"
	"os"
	"time"
)

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
