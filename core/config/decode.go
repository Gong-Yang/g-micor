package config

import (
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
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
	slog.Info("Config loaded", "conf", conf)
	slog.Info("Config loaded", "appConfigFile", appConfigFile, "globConfigFile", globConfigFile)
}

// GetParentDir 获取指定目录的上一级目录
func GetParentDir(dir string) string {
	return filepath.Dir(dir)
}

// GetWorkDirParent 获取当前工作目录的上一级目录
func GetWorkDirParent() (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Dir(workDir), nil
}

// GetMainDir 获取调用此函数的main.go文件所在的目录
func GetMainDir() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	return filepath.Dir(file)
}
