package main

import (
	"fmt"

	// ★用 _ 导入，只触发init()，不使用包内的函数
	_ "kama-chat-server/internal/config"         // 配置加载
	_ "kama-chat-server/internal/dao"            // 数据库连接
	_ "kama-chat-server/internal/service/redis"  // Redis服务
	_ "kama-chat-server/internal/service/email"  // 邮箱验证码服务
	"kama-chat-server/internal/https_server"     // HTTP服务器（需要调用RunServer）
)

func main() {
	// ★导入包后，各包的init()已自动执行：
	// 1. config.init() → 加载配置
	// 2. dao.init() → 连接数据库
	// 3. redis.init() → 连接Redis
	// 4. https_server.init() → 注册路由

	// 验证服务已初始化
	fmt.Println("服务启动中...")

	// ★启动HTTP服务器（阻塞运行）
	https_server.RunServer()
}