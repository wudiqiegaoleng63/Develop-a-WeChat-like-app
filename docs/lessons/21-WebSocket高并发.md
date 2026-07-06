# 教学文档 20: WebSocket高并发（★重点）

### 前置条件：基础架构已完成

WebSocket是实时通信扩展，需要先完成基础开发顺序：

```
Model → DAO → Service → Controller → 路由注册 → WebSocket

①-⑤: 基础架构（01-09文档）已完成
⑥ WebSocket: ★当前文档★ 实时双向通信
⑦ Kafka: 高并发消息缓冲（19文档）
```

---

## 一、文件创建清单

| 序号 | 文件路径 | 说明 |
|------|----------|------|
| 1 | `internal/service/chat/client.go` | WebSocket客户端（Read/Write/初始化/登出） |
| 2 | `internal/service/chat/server.go` | Channel模式Server |
| 3 | `internal/service/chat/kafka_server.go` | Kafka模式Server（文档20已创建） |
| 4 | `api/v1/ws_controller.go` | WebSocket Controller |
| 5 | `internal/dto/request/ws_logout_request.go` | 登出请求结构体 |

---

## 二、文件1: client.go

**文件位置:** `internal/service/chat/client.go`

```
internal/
└── service/
    └── chat/
        └── client.go  ← 创建此文件
```

### 完整代码

```go
package chat

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"gochat/internal/config"
	"gochat/internal/dao"
	"gochat/internal/dto/request"
	"gochat/internal/model"
	myKafka "gochat/internal/service/kafka"
	"gochat/pkg/constants"
	"gochat/pkg/enum/message/message_status_enum"
	"gochat/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

// MessageBack 返回给前端的消息结构
type MessageBack struct {
	Message []byte
	Uuid    string
}

// Client WebSocket客户端结构体
type Client struct {
	Conn     *websocket.Conn      // WebSocket连接
	Uuid     string               // 用户唯一标识
	SendTo   chan []byte          // 给Server端的Channel（Channel模式缓冲）
	SendBack chan *MessageBack    // 发送给前端的Channel
}

// upgrader WebSocket升级器，用于将HTTP连接升级为WebSocket连接
var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	// 检查连接的Origin头，允许跨域
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 全局变量
var ctx = context.Background()

// 从配置文件读取消息模式（channel 或 kafka）
var messageMode = config.GetConfig().KafkaConfig.MessageMode

// Read 从WebSocket读取消息
// ★在单独协程中运行
// 流程: WebSocket读取 → 根据配置选择Channel模式或Kafka模式
func (c *Client) Read() {
	zlog.Info("ws read goroutine start")
	for {
		// 1. 从WebSocket读取消息（阻塞）
		_, jsonMessage, err := c.Conn.ReadMessage()
		if err != nil {
			zlog.Error(err.Error())
			return // 直接断开websocket
		}

		// 2. 解析消息
		var message = request.ChatMessageRequest{}
		if err := json.Unmarshal(jsonMessage, &message); err != nil {
			zlog.Error(err.Error())
		}
		log.Println("接受到消息为: ", jsonMessage)

		// 3. ★根据配置选择消息处理模式
		if messageMode == "channel" {
			// ★★★ Channel模式: 多级缓冲机制 ★★★

			// 步骤A: 如果Server的Transmit没满，先把SendTo中的消息转发
			for len(ChatServer.Transmit) < constants.CHANNEL_SIZE && len(c.SendTo) > 0 {
				sendToMessage := <-c.SendTo
				ChatServer.SendMessageToTransmit(sendToMessage)
			}

			// 步骤B: 如果Server没满，SendTo空了，直接给Server.Transmit
			if len(ChatServer.Transmit) < constants.CHANNEL_SIZE {
				ChatServer.SendMessageToTransmit(jsonMessage)
			} else if len(c.SendTo) < constants.CHANNEL_SIZE {
				// 步骤C: 如果Server满了，先塞到SendTo缓冲
				c.SendTo <- jsonMessage
			} else {
				// 步骤D: 都满了，发送失败提示
				if err := c.Conn.WriteMessage(websocket.TextMessage, []byte("由于目前同一时间过多用户发送消息，消息发送失败，请稍后重试")); err != nil {
					zlog.Error(err.Error())
				}
			}
		} else {
			// ★★★ Kafka模式: 直接写入Kafka ★★★
			if err := myKafka.KafkaService.ChatWriter.WriteMessages(ctx, kafka.Message{
				Key:   []byte(strconv.Itoa(config.GetConfig().KafkaConfig.Partition)),
				Value: jsonMessage,
			}); err != nil {
				zlog.Error(err.Error())
			}
			zlog.Info("已发送信息: " + string(jsonMessage))
		}
	}
}

// Write 发送消息给前端
// ★在单独协程中运行
// 流程: 从SendBack Channel读取 → 发送WebSocket → 更新消息状态
func (c *Client) Write() {
	zlog.Info("ws write goroutine start")
	for messageBack := range c.SendBack { // 阻塞状态
		// 1. 通过 WebSocket 发送消息
		err := c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
		if err != nil {
			zlog.Error(err.Error())
			return // 直接断开websocket
		}

		// 2. 说明顺利发送，修改状态为已发送
		if res := dao.GormDB.Model(&model.Message{}).Where("uuid=?", messageBack.Uuid).Update("status", message_status_enum.Sent); res.Error != nil {
			zlog.Error(res.Error.Error())
		}
	}
}

// NewClientInit 创建并初始化Client
// ★当前端有登录消息时，会调用该函数
func NewClientInit(c *gin.Context, clientId string) {
	kafkaConfig := config.GetConfig().KafkaConfig

	// 1. 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zlog.Error(err.Error())
	}

	// 2. 创建Client实例
	client := &Client{
		Conn:     conn,
		Uuid:     clientId,
		SendTo:   make(chan []byte, constants.CHANNEL_SIZE),
		SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE),
	}

	// 3. ★根据消息模式注册到对应的Server
	if kafkaConfig.MessageMode == "channel" {
		ChatServer.SendClientToLogin(client)
	} else {
		KafkaChatServer.SendClientToLogin(client)
	}

	// 4. 启动Read和Write协程
	go client.Read()
	go client.Write()

	zlog.Info("ws连接成功")
}

// ClientLogout WebSocket客户端登出
// ★当前端用户退出登录时调用
func ClientLogout(clientId string) (string, int) {
	kafkaConfig := config.GetConfig().KafkaConfig

	// 1. 从Server的Clients map中获取客户端
	client := ChatServer.Clients[clientId]

	// 2. 如果客户端存在，关闭连接
	if client != nil {
		// ★根据消息模式发送到对应的登出通道
		if kafkaConfig.MessageMode == "channel" {
			ChatServer.SendClientToLogout(client)
		} else {
			KafkaChatServer.SendClientToLogout(client)
		}

		// 关闭WebSocket连接
		if err := client.Conn.Close(); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}

		// 关闭Channel
		close(client.SendTo)
		close(client.SendBack)
	}

	return "退出成功", 0
}
```

---

## 三、文件2: server.go

**文件位置:** `internal/service/chat/server.go`

```
internal/
└── service/
    └── chat/
        └── server.go  ← 创建此文件
```

### 完整代码

```go
package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"gochat/internal/dao"
	"gochat/internal/dto/request"
	"gochat/internal/dto/respond"
	"gochat/internal/model"
	myredis "gochat/internal/service/redis"
	"gochat/pkg/constants"
	"gochat/pkg/enum/message/message_status_enum"
	"gochat/pkg/enum/message/message_type_enum"
	"gochat/pkg/util/random"
	"gochat/pkg/zlog"
	"log"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Clients  map[string]*Client
	mutex    *sync.Mutex
	Transmit chan []byte  // 转发通道
	Login    chan *Client // 登录通道
	Logout   chan *Client // 退出登录通道
}

var ChatServer *Server

func init() {
	if ChatServer == nil {
		ChatServer = &Server{
			Clients:  make(map[string]*Client),
			mutex:    &sync.Mutex{},
			Transmit: make(chan []byte, constants.CHANNEL_SIZE),
			Login:    make(chan *Client, constants.CHANNEL_SIZE),
			Logout:   make(chan *Client, constants.CHANNEL_SIZE),
		}
	}
}

// 将https://127.0.0.1:8000/static/xxx 转为 /static/xxx
func normalizePath(path string) string {
	// 查找 "/static/" 的位置
	if path == "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" {
		return path
	}
	staticIndex := strings.Index(path, "/static/")
	if staticIndex < 0 {
		log.Println(path)
		zlog.Error("路径不合法")
	}
	// 返回从 "/static/" 开始的部分
	return path[staticIndex:]
}

// Start 启动函数，Server端用主进程起，Client端可以用协程起
func (s *Server) Start() {
	defer func() {
		close(s.Transmit)
		close(s.Logout)
		close(s.Login)
	}()
	for {
		select {
		case client := <-s.Login:
			{
				s.mutex.Lock()
				s.Clients[client.Uuid] = client
				s.mutex.Unlock()
				zlog.Debug(fmt.Sprintf("欢迎来到kama聊天服务器，亲爱的用户%s\n", client.Uuid))
				err := client.Conn.WriteMessage(websocket.TextMessage, []byte("欢迎来到kama聊天服务器"))
				if err != nil {
					zlog.Error(err.Error())
				}
			}

		case client := <-s.Logout:
			{
				s.mutex.Lock()
				delete(s.Clients, client.Uuid)
				s.mutex.Unlock()
				zlog.Info(fmt.Sprintf("用户%s退出登录\n", client.Uuid))
				if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("已退出登录")); err != nil {
					zlog.Error(err.Error())
				}
			}

		case data := <-s.Transmit:
			{
				var chatMessageReq request.ChatMessageRequest
				if err := json.Unmarshal(data, &chatMessageReq); err != nil {
					zlog.Error(err.Error())
				}
				// log.Println("原消息为：", data, "反序列化后为：", chatMessageReq)
				if chatMessageReq.Type == message_type_enum.Text {
					// 存message
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    chatMessageReq.Content,
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "0B",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := dao.GormDB.Create(&message); res.Error != nil {
						zlog.Error(res.Error.Error())
					}
					if message.ReceiveId[0] == 'U' { // 发送给User
						// 如果能找到ReceiveId，说明在线，可以发送，否则存表后跳过
						// 因为在线的时候是通过websocket更新消息记录的，离线后通过存表，登录时只调用一次数据库操作
						// 切换chat对象后，前端的messageList也会改变，获取messageList从第二次就是从redis中获取
						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 因为send_id肯定在线，所以这里在后端进行在线回显message，其实优化的话前端可以直接回显
						// 问题在于前后端的req和rsp结构不同，前端存储message的messageList不能存req，只能存rsp
						// 所以这里后端进行回显，前端不回显
						sendClient := s.Clients[message.SendId]
						sendClient.SendBack <- messageBack
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}

					} else if message.ReceiveId[0] == 'G' { // 发送给Group
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						var group model.GroupInfo
						if res := dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
						var members []string
						if err := json.Unmarshal(group.Members, &members); err != nil {
							zlog.Error(err.Error())
						}
						s.mutex.Lock()
						for _, member := range members {
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					}
				} else if chatMessageReq.Type == message_type_enum.File {
					// 存message
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        chatMessageReq.Url,
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   chatMessageReq.FileSize,
						FileType:   chatMessageReq.FileType,
						FileName:   chatMessageReq.FileName,
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := dao.GormDB.Create(&message); res.Error != nil {
						zlog.Error(res.Error.Error())
					}
					if message.ReceiveId[0] == 'U' { // 发送给User
						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							receiveClient.SendBack <- messageBack
						}
						sendClient := s.Clients[message.SendId]
						sendClient.SendBack <- messageBack
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					} else {
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						var group model.GroupInfo
						if res := dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
						var members []string
						if err := json.Unmarshal(group.Members, &members); err != nil {
							zlog.Error(err.Error())
						}
						s.mutex.Lock()
						for _, member := range members {
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zlog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zlog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zlog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zlog.Error(err.Error())
							}
						}
					}
				} else if chatMessageReq.Type == message_type_enum.AudioOrVideo {
					var avData request.AVData
					if err := json.Unmarshal([]byte(chatMessageReq.AVdata), &avData); err != nil {
						zlog.Error(err.Error())
					}
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     chatMessageReq.AVdata,
					}
					if avData.MessageId == "PROXY" && (avData.Type == "start_call" || avData.Type == "receive_call" || avData.Type == "reject_call") {
						// 存message
						message.SendAvatar = normalizePath(message.SendAvatar)
						if res := dao.GormDB.Create(&message); res.Error != nil {
							zlog.Error(res.Error.Error())
						}
					}

					if chatMessageReq.ReceiveId[0] == 'U' { // 发送给User
						messageRsp := respond.AVMessageRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: message.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
							AVdata:     message.AVdata,
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zlog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							receiveClient.SendBack <- messageBack
						}
						// 通话这不能回显，发回去的话就会出现两个start_call。
						s.mutex.Unlock()
					}
				}

			}
		}
	}
}

func (s *Server) Close() {
	close(s.Login)
	close(s.Logout)
	close(s.Transmit)
}

func (s *Server) SendClientToLogin(client *Client) {
	s.mutex.Lock()
	s.Login <- client
	s.mutex.Unlock()
}

func (s *Server) SendClientToLogout(client *Client) {
	s.mutex.Lock()
	s.Logout <- client
	s.mutex.Unlock()
}

func (s *Server) SendMessageToTransmit(message []byte) {
	s.mutex.Lock()
	s.Transmit <- message
	s.mutex.Unlock()
}

func (s *Server) RemoveClient(uuid string) {
	s.mutex.Lock()
	delete(s.Clients, uuid)
	s.mutex.Unlock()
}
```

---

## 四、文件3: ws_controller.go

**文件位置:** `api/v1/ws_controller.go`

```
api/
└── v1/
    └── ws_controller.go  ← 创建此文件
```

### 完整代码

```go
package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gochat/internal/dto/request"
	"gochat/internal/service/chat"
	"gochat/pkg/constants"
	"gochat/pkg/zlog"
)

// WsLogin WebSocket登录
// ★必须是GET请求（WebSocket协议要求）
// 路由: GET /user/wsLogin?client_id=xxx
func WsLogin(c *gin.Context) {
	// 1. 获取用户UUID（从查询参数）
	clientId := c.Query("client_id")
	if clientId == "" {
		zlog.Error("clientId获取失败")
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": "clientId获取失败",
		})
		return
	}

	// 2. 初始化WebSocket客户端
	chat.NewClientInit(c, clientId)
}

// WsLogout WebSocket登出
// 路由: POST /user/wsLogout
func WsLogout(c *gin.Context) {
	// 1. 绑定请求参数
	var req request.WsLogoutRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}

	// 2. 调用Chat服务层关闭WebSocket连接
	message, ret := chat.ClientLogout(req.OwnerId)

	// 3. 返回响应
	JsonBack(c, message, ret, nil)
}
```

---

## 五、文件4: ws_logout_request.go

**文件位置:** `internal/dto/request/ws_logout_request.go`

```
internal/
└── dto/
    └── request/
        └── ws_logout_request.go  ← 创建此文件
```

### 完整代码

```go
package request

// WsLogoutRequest WebSocket登出请求
type WsLogoutRequest struct {
	OwnerId string `json:"owner_id"` // 登出用户的uuid
}
```

---

## 六、路由注册

**文件位置:** `internal/https_server/https_server.go`

在 `registerRoutes()` 函数中添加：

```go
func registerRoutes() {
	// WebSocket相关路由
	GE.GET("/user/wsLogin", v1.WsLogin)      // WebSocket登录（GET请求）
	GE.POST("/user/wsLogout", v1.WsLogout)   // WebSocket登出

	// ... 其他路由
}
```

---

## 七、main.go 启动Server

**文件位置:** `cmd/gochat/main.go`

### 完整代码

```go
package main

import (
	"fmt"
	"gochat/internal/config"
	"gochat/internal/https_server"
	"gochat/internal/service/chat"
	"gochat/internal/service/kafka"
	myredis "gochat/internal/service/redis"
	"gochat/pkg/zlog"
	"os"
	"os/signal"
	"syscall"
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
```

### 关键步骤说明

| 步骤 | 说明 |
|------|------|
| 1. Kafka初始化 | 仅在Kafka模式下执行，创建Topic、连接生产者/消费者 |
| 2. 启动Server | 根据配置选择Channel模式或Kafka模式的Server |
| 3. HTTP服务器 | 启动Gin服务器，监听端口 |
| 4. 信号监听 | 监听 `Ctrl+C` (SIGINT) 和终止信号 (SIGTERM) |
| 5. 等待信号 | 阻塞等待，直到收到关闭信号 |
| 6. 关闭Kafka | 关闭生产者和消费者连接 |
| 7. 关闭ChatServer | 关闭所有Channel（Login/Logout/Transmit） |
| 8. 清理Redis | 删除所有缓存键，避免数据残留 |

---

## 八、Channel模式缓冲机制详解

### 多级缓冲设计

```
消息缓冲优先级：
1. ChatServer.Transmit（最优先）- 全局转发通道
2. Client.SendTo（次优先）- 用户个人缓冲
3. 消息失败（最后）- 提示用户
```

### 缓冲流程图

```
WebSocket接收消息
       │
       ↓
┌──────────────────────────────────────┐
│ 步骤A: ChatServer.Transmit未满?      │
│        ├─→ YES: 先清空Client.SendTo  │
│        │      (把缓冲的消息发送出去)  │
│        └─→ NO: 跳过                  │
└──────────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────┐
│ 步骤B: ChatServer.Transmit未满?      │
│        ├─→ YES: 直接写入Transmit     │
│        │      (消息进入处理流程)      │
│        └─→ NO: 进入步骤C             │
└──────────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────┐
│ 步骤C: Client.SendTo未满?            │
│        ├─→ YES: 写入SendTo缓冲       │
│        │      (等下次有机会再发送)    │
│        └─→ NO: 进入步骤D             │
└──────────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────┐
│ 步骤D: 发送失败提示                   │
│        "由于目前同一时间过多用户发送   │
│         消息，消息发送失败，请稍后重试" │
└──────────────────────────────────────┘
```

---

## 九、消息流转完整流程

### Channel模式 - 私聊

```
用户A发送消息
    │
    ▼
ClientA.Read() 读取WebSocket
    │
    ▼
ChatServer.SendMessageToTransmit() 写入Transmit
    │
    ▼
Server.Start() case data := <-s.Transmit
    │
    ▼
handleTextMessage() 处理
    │
    ├── 存储到数据库
    │
    ├── 判断 receiveId[0] == 'U' (私聊)
    │
    └── forwardToUser() 转发
         │
         ├── receiveClient.SendBack (用户B在线则发送)
         │
         └── sendClient.SendBack (消息回显给用户A)
              │
              ▼
         Client.Write() 发送WebSocket
              │
              ▼
         用户A/B收到消息
```

### Channel模式 - 群聊

```
用户A发送群消息
    │
    ▼
(同上处理流程)
    │
    ▼
判断 receiveId[0] == 'G' (群聊)
    │
    ▼
forwardToGroup()
    │
    ├── 查询群成员列表
    │
    └── 遍历所有成员 → SendBack发送
         │
         ▼
    所有在线群成员收到消息
```

---

## 十、安装依赖

```bash
# WebSocket库
go get github.com/gorilla/websocket

# Kafka Go客户端（如果使用Kafka模式）
go get github.com/segmentio/kafka-go
```

---

## 十一、下一步

WebSocket实现完成后，继续学习：
- **21-聊天室管理.md** - 聊天室功能实现
- **22-分布式系统扩展.md** - 多Server部署和负载均衡
