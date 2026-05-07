package chat

import (
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"log"
	"strings"
	"sync"
)

// Server Channel模式服务器结构体
type Server struct {
	Clients	map[string]*Client
	mutex	*sync.Mutex   //并发锁
	Transmit	chan []byte
	Login		chan *Client
	Logout		chan *Client
}

var ChatServer *Server

// 初始化
func init() {
	if ChatServer == nil {
		ChatServer = &Server{
			Clients: make(map[string]*Client),
			mutex:    &sync.Mutex{},
            Transmit: make(chan []byte, constants.CHANNEL_SIZE),
            Login:    make(chan *Client, constants.CHANNEL_SIZE),
            Logout:   make(chan *Client, constants.CHANNEL_SIZE),
		}
	}
}

// Client 关闭Server
func (s *Server) Close() {
	close(s.Login)
	close(s.Logout)
	close(s.Transmit)
}

// SendClientToLogin 将Login发送到登录通道
func (s *Server) SendClientToLogin(client *Client) {
	s.mutex.Lock()
	s.Login <- client
	s.mutex.Unlock()
}

// SendClientToLogout 将Client发送到登出通道
func (s *Server) SendClientToLogout(client *Client) {
    s.mutex.Lock()
    s.Logout <- client
    s.mutex.Unlock()
}

// SendMessageToTransmit 将消息发送到转发通道
func (s *Server) SendMessageToTransmit(message []byte) {
    s.mutex.Lock()
    s.Transmit <- message
    s.mutex.Unlock()
}

// RemoveClient 从在线列表移除Client
func (s *Server) RemoveClient(uuid string) {
    s.mutex.Lock()
    delete(s.Clients, uuid)
    s.mutex.Unlock()
}

func normalizePath(path string) string {
    // 特殊处理：Element UI 默认头像
    if path == "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" {
        return path
    }

    // 查找 "/static/" 的位置
    staticIndex := strings.Index(path, "/static/")
    if staticIndex < 0 {
        log.Println(path)
        zlog.Error("路径不合法")
    }

    // 返回从 "/static/" 开始的部分
    return path[staticIndex:]
}