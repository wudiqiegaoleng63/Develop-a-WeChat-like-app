package config

import (
	"log"
	"time"

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

type StaticSrcConfig struct {
    StaticAvatarPath string `toml:"staticAvatarPath"`
    StaticFilePath   string `toml:"staticFilePath"`
}

// redis配置
type RedisConfig struct {
    Host     string `toml:"host"`
    Port     int    `toml:"port"`
    Password string `toml:"password"`
    Db       int    `toml:"db"`
}

// AuthCodeConfig - 阿里云短信验证配置
// type AuthCodeConfig struct {
//     AccessKeyID     string `toml:"accessKeyID"`     // 阿里云AccessKey ID
//     AccessKeySecret string `toml:"accessKeySecret"` // 阿里云AccessKey Secret
//     SignName        string `toml:"signName"`        // 短信签名
//     TemplateCode    string `toml:"templateCode"`    // 短信模板Code
// }

type EmailConfig struct {
    SmtpHost     string `toml:"smtpHost"`     // SMTP服务器
    SmtpPort     int    `toml:"smtpPort"`     // 端口
    SmtpUsername string `toml:"smtpUsername"` // 发件邮箱
    SmtpPassword string `toml:"smtpPassword"` // 授权码
    FromName     string `toml:"fromName"`     // 发件人名称
}

// JwtConfig - JWT配置
type JwtConfig struct {
	Secret      string `toml:"secret"`      // JWT签名密钥
	ExpireHours int    `toml:"expireHours"` // Token过期时间（小时）
}

// KafkaConfig - Kafka配置
type KafkaConfig struct {
    MessageMode string        `toml:"messageMode"` // channel 或 kafka
    HostPort    string        `toml:"hostPort"`    // Kafka地址
    LoginTopic  string        `toml:"loginTopic"`  // 登录Topic
    ChatTopic   string        `toml:"chatTopic"`   // 聊天消息Topic
    LogoutTopic string        `toml:"logoutTopic"` // 登出Topic
    Partition   int           `toml:"partition"`   // 分区号
    Timeout     time.Duration `toml:"timeout"`     // 超时秒数
}

// 总配置
type Config struct {
	MainConfig    	`toml:"mainConfig"`
    MysqlConfig  	`toml:"mysqlConfig"`
    LogConfig      	`toml:"logConfig"`
	StaticSrcConfig `toml:"staticSrcConfig"`
	RedisConfig     `toml:"redisConfig"`
	EmailConfig    `toml:"emailConfig"`
	KafkaConfig     `toml:"kafkaConfig"`
	JwtConfig       `toml:"jwtConfig"`
}



var config *Config


// 加载

func LoadConfig() error {
	_, err := toml.DecodeFile("configs/config_local.toml", config)
	if err != nil {
		log.Fatal("配置加载失败:", err.Error())
		return err
	}
	// 校验JWT密钥不能是默认占位符
	if config.JwtConfig.Secret == "gochat-jwt-secret-key-change-in-production" {
		log.Fatal("JWT secret 不能使用默认值，请在 config_local.toml 中修改 jwtConfig.secret")
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













