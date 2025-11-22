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

func Init(conf any, workDir string) {
	flag.Parse()

	var err error
	if workDir == "" {
		workDir, err = os.Getwd()
		if err != nil {
			os.Exit(1)
		}
	}
	// 尝试全局配置文件
	var configFile = defaultConfigFile
	if *env != "" {
		configFile = fmt.Sprintf(configFileTemplate, *env)
	}

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
