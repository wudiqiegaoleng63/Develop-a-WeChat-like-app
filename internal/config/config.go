package config

import (
	"log"
	"github.com/BurntSushi/toml"
)

// 服务器信息
type MainConfig struct {
	AppName string 	`toml:"appName"`
	Host    string 	`toml:"host"`
	Port  	int		`toml:"port"`	
}

// Mysql配置
type MysqlConfig struct{
	Host	string	`toml:"host"`
	Port 	int		`toml:"port"`
	User	string 	`toml:"user"`
	Password	string	`toml:"password"`
	DatabaseName	string	`toml:"databaseName"`
}

// LogConfig 日志配置
type LogConfig struct {
	LogPath	string	`toml:"logPath"`
}

// 总配置
type Config struct {
	MainConfig  MainConfig  `toml:"mainConfig"`
    MysqlConfig MysqlConfig `toml:"mysqlConfig"`
    LogConfig   LogConfig   `toml:"logConfig"`
}

var config *Config


// 加载

func LoadConfig() error {
	_, err := toml.DecodeFile("configs/config_local.toml", config)
	if err != nil {
		log.Fatal("配置加载失败:", err.Error())
		return err
	}
	return nil
}

// 获取配置实例
func GetConfig() *Config{
	if config == nil {
		config = new(Config)
		_ = LoadConfig()
	}
	return config
}













