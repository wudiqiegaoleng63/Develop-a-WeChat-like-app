package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "kama-chat-server/internal/agent" // AI Agent：初始化时幂等创建 AI助手 系统用户
	"kama-chat-server/internal/config"
	_ "kama-chat-server/internal/dao"        // 数据库连接
	"kama-chat-server/internal/https_server" // HTTP服务器（需要调用RunServer）
	"kama-chat-server/internal/service/chat"
	_ "kama-chat-server/internal/service/email" // 邮箱验证码服务
	"kama-chat-server/internal/service/kafka"
	_ "kama-chat-server/internal/service/redis" // Redis服务
	myredis "kama-chat-server/internal/service/redis"
	"kama-chat-server/pkg/zlog"
)

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port
	kafkaConfig := conf.KafkaConfig

	// 1. 如果使用Kafka模式，初始化Kafka连接
	if kafkaConfig.MessageMode == "kafka" {
		kafka.KafkaService.KafkaInit()
	}

	// 2. 根据消息模式启动对应的Server
	if kafkaConfig.MessageMode == "channel" {
		go chat.ChatServer.Start()
	} else {
		go chat.KafkaChatServer.Start()
	}

	// 3. 启动HTTP服务器
	go func() {
		if err := https_server.GE.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			zlog.Fatal("server running fault")
			return
		}
	}()

	// 4. 设置信号监听（优雅关闭）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 5. 等待信号
	<-quit

	// 6. 关闭Kafka连接
	if kafkaConfig.MessageMode == "kafka" {
		kafka.KafkaService.KafkaClose()
	}

	// 7. 关闭ChatServer
	chat.ChatServer.Close()

	zlog.Info("关闭服务器...")

	// 8. 删除所有Redis键
	if err := myredis.DeleteAllRedisKeys(); err != nil {
		zlog.Error(err.Error())
	} else {
		zlog.Info("所有Redis键已删除")
	}

	zlog.Info("服务器已关闭")
}
