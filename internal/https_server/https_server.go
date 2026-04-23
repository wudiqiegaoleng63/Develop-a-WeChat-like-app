package https_server

import (
	"kama-chat-server/internal/config"
	"strconv"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	v1 "kama-chat-server/api/v1"
)

// 全局gin实例
var GE *gin.Engine

func init() {
	// 创建 Gin 引擎
	GE = gin.Default()

	// CORS 跨域配置
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "PUT", "POST", "DELETE"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type"}
	GE.Use(cors.New(corsConfig))


	// 静态文件服务
	conf := config.GetConfig()
	GE.Static("/static/avatars", conf.StaticSrcConfig.StaticAvatarPath)
	GE.Static("/static/files", conf.StaticSrcConfig.StaticFilePath)

	// 注册路由
	registerRoutes()
}

func registerRoutes() {
	// 用户相关路由 (POST)
	GE.POST("/login", v1.Login)	//登录
}

// RunServer 启动HTTP服务器
func RunServer() {
	conf := config.GetConfig()

	// 拼接地址
	addr := conf.MainConfig.Host + ":" + strconv.Itoa(conf.MainConfig.Port)

	GE.Run(addr)
}