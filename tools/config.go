package tools

import (
	"github.com/spf13/viper"
)

type GlobalConfig struct {
	StorageName string //存储名称
	AppName     string //应用名称，socket文件路径组成
	Ip          string //quota server IP
	Port        int    //quota server port
	Username    string //quota server username
	Password    string //quota server password

	FilesystemName string //文件系统名称，某些存储如曙光的单独配置
	RootPath       string //后端存储的根目录，绝对路径
}

var Config *GlobalConfig

func ParseConfig(projectName, configFile string) {
	Logger.Info("param:", "projectName", projectName, "configFile", configFile)
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		Logger.Fatal("ParseConfig, read config", "error", err)
	}
	Config = new(GlobalConfig)
	err = viper.UnmarshalKey(projectName, Config)
	if err != nil {
		Logger.Fatal("ParseConfig, Unmarshal Config", "error", err)
	}

	Logger.Printf("config is: \n %+v", Config)
}
