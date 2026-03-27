package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var env = flag.String("env", "", "env")

const defaultConfigFile = "config.yml"
const configFileTemplate = "config-%s.yml"

func Init(conf []any) {
	flag.Parse()

	var err error
	workDir, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	InitByDir(conf, workDir)
}
func InitByDir(conf []any, workDir string) {
	// 尝试全局配置文件
	var configFile = defaultConfigFile
	if *env != "" {
		configFile = fmt.Sprintf(configFileTemplate, *env)
	}

	globConfigFile := filepath.Dir(workDir) + "/" + configFile
	data, err := os.ReadFile(globConfigFile)
	if err != nil {
		panic(err)
	}
	for _, item := range conf {
		err = yaml.Unmarshal(data, item)
		if err != nil {
			panic(err)
		}
	}

	// 尝试项目配置文件
	appConfigFile := workDir + "/" + configFile
	data, err = os.ReadFile(appConfigFile)
	if err != nil {
		panic(err)
	}
	for _, item := range conf {
		err = yaml.Unmarshal(data, item)
		if err != nil {
			panic(err)
		}
	}
	//slog.Info("Config loaded", "conf", conf)
	slog.Info("Config loaded", "appConfigFile", appConfigFile, "globConfigFile", globConfigFile)
}
