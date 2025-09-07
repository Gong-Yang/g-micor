package config

import (
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path/filepath"
)

var configFile = "config.yml"

func Init(conf any) {
	workDir, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	// 尝试全局配置文件
	globConfigFile := filepath.Dir(workDir) + "/" + configFile
	file, err := os.Open(globConfigFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	err = yaml.NewDecoder(file).Decode(conf)
	if err != nil {
		panic(err)
	}

	// 尝试项目配置文件
	appConfigFile := workDir + "/" + configFile
	file, err = os.Open(appConfigFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	err = yaml.NewDecoder(file).Decode(conf)
	if err != nil {
		panic(err)
	}
	//slog.Info("Config loaded", "conf", conf)
	slog.Info("Config loaded", "appConfigFile", appConfigFile, "globConfigFile", globConfigFile)
}
