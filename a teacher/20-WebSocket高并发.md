# 教学文档 20: WebSocket高并发（★重点）

### 前置条件：基础架构已完成

WebSocket是实时通信扩展，需要先完成基础开发顺序：

```
Model → DAO → Service → Controller → 路由注册 → WebSocket

①-⑤: 基础架构（01-09文档）已完成
⑥ WebSocket: ★当前文档★ 实时双向通信
⑦ Kafka: 高并发消息缓冲（19文档）
```

## 一、WebSocket基础概念

### 什么是WebSocket？

WebSocket是全双工通信协议：
- 服务器可以主动推送消息
- 连接保持长连接
- 实时聊天场景最适合

### WebSocket vs HTTP

| 特性 | HTTP | WebSocket |
|------|------|-----------|
| 通信方向 | 单向（客户端发起） | 双向 |
| 连接 | 短连接 | 长连接 |
| 实时性 | 需轮询 | 实时推送 |
| 适用场景 | API请求 | 聊天、游戏 |

---

## 二、WebSocket架构对比

### Channel模式（小规模）

```
┌────────┐  SendTo(chan)  ┌────────┐  Transmit(chan)  ┌────────┐
│ Client │ ────────────── │ Server │ ─────────────── │ 处理   │
└────────┘                └────────┘                  └────────┘

问题:
- Channel容量有限（constants.CHANNEL_SIZE = 100）
- 高峰期Channel满，消息发送失败
- Server内存压力大
```

### Kafka模式（大规模）

```
┌────────┐   WebSocket    ┌────────┐   写入Kafka   ┌────────┐
│ Client │ ────────────── │ Writer │ ──────────── │ Kafka  │
└────────┘                └────────┘               └────────┘
                                                       │
                              ┌─────────────────────────┘
                              │ 消费Kafka
                              │
                         ┌────────┐   SendBack(chan)   ┌────────┐
                         │ Server │ ────────────────── │ Client │
                         └────────┘                    └────────┘

优势:
- Kafka无限缓冲，不阻塞
- Server宕机消息不丢失
- 可多Server消费（分布式）
```

---

## 三、WebSocket Client结构

**文件位置:** `internal/service/chat/client.go`

### 完整代码（带详细注释）

```go
package chat

import (
    "context"
    "encoding/json"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "github.com/segmentio/kafka-go"
    "kama_chat_server/internal/config"
    "kama_chat_server/internal/dao"
    "kama_chat_server/internal/dto/request"
    "kama_chat_server/internal/model"
    myKafka "kama_chat_server/internal/service/kafka"
    "kama_chat_server/pkg/constants"
    "kama_chat_server/pkg/enum/message/message_status_enum"
    "kama_chat_server/pkg/zlog"
    "log"
    "net/http"
    "strconv"
)

// ============================================================
// MessageBack - 返回给前端的消息结构
// ============================================================
type MessageBack struct {
    Message []byte  // 序列化的消息（JSON）
    Uuid    string  // 消息UUID（用于更新消息状态）
}

// ============================================================
// Client - WebSocket客户端结构
// ============================================================
// ★每个WebSocket连接对应一个Client实例
type Client struct {
    Conn     *websocket.Conn      // WebSocket连接
    Uuid     string               // 用户唯一标识
    SendTo   chan []byte          // 给Server端的Channel（Channel模式缓冲）
    SendBack chan *MessageBack    // 发送给前端的Channel
}

// ============================================================
// upgrader - WebSocket升级器
// ============================================================
// ★用于将HTTP连接升级为WebSocket连接
var upgrader = websocket.Upgrader{
    ReadBufferSize:  2048,
    WriteBufferSize: 2048,
    // 检查连接的Origin头，允许跨域
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// ============================================================
// 全局变量
// ============================================================
var ctx = context.Background()

// ★从配置文件读取消息模式（channel 或 kafka）
var messageMode = config.GetConfig().KafkaConfig.MessageMode

// ============================================================
// Read - 从WebSocket读取消息
// ============================================================
// ★在单独协程中运行
// 流程: WebSocket读取 → 根据配置选择Channel模式或Kafka模式
func (c *Client) Read() {
    zlog.Info("ws read goroutine start")
    for {
        // 1. 从WebSocket读取消息（阻塞）
        _, jsonMessage, err := c.Conn.ReadMessage()
        if err != nil {
            zlog.Error(err.Error())
            return  // 直接断开websocket
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
            zlog.Info("已发送消息：" + string(jsonMessage))
        }
    }
}

// ============================================================
// Write - 发送消息给前端
// ============================================================
// ★在单独协程中运行
// 流程: 从SendBack Channel读取 → 发送WebSocket → 更新消息状态
func (c *Client) Write() {
    zlog.Info("ws write goroutine start")
    for messageBack := range c.SendBack {  // 阻塞状态
        // 1. 通过 WebSocket 发送消息
        err := c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
        if err != nil {
            zlog.Error(err.Error())
            return  // 直接断开websocket
        }
        
        // 2. 说明顺利发送，修改状态为已发送
        if res := dao.GormDB.Model(&model.Message{}).Where("uuid = ?", messageBack.Uuid).Update("status", message_status_enum.Sent); res.Error != nil {
            zlog.Error(res.Error.Error())
        }
    }
}

// ============================================================
// NewClientInit - 创建并初始化Client
// ============================================================
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

// ============================================================
// ClientLogout - WebSocket客户端登出
// ============================================================
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

## 四、Channel模式缓冲机制 ★重要

### 多级缓冲设计

```
消息缓冲优先级：
1. ChatServer.Transmit（最优先）
2. Client.SendTo（次优先）
3. 消息失败（最后）
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

### 为什么这样设计？

**问题**: Channel容量有限，高峰期可能满

**解决**: 多级缓冲
- Client.SendTo作为第一级缓冲（每个用户独立）
- ChatServer.Transmit作为第二级缓冲（全局共享）

**适用场景**:
- 小规模用户（<100）: Channel模式足够
- 大规模用户（>1000）: 使用Kafka模式

---

## 五、WebSocket Server结构

**文件位置:** `internal/service/chat/server.go`

### Server结构体定义

```go
package chat

import (
    "encoding/json"
    "errors"
    "fmt"
    "github.com/go-redis/redis/v8"
    "github.com/gorilla/websocket"
    "kama_chat_server/internal/dao"
    "kama_chat_server/internal/dto/request"
    "kama_chat_server/internal/dto/respond"
    "kama_chat_server/internal/model"
    myredis "kama_chat_server/internal/service/redis"
    "kama_chat_server/pkg/constants"
    "kama_chat_server/pkg/enum/message/message_status_enum"
    "kama_chat_server/pkg/enum/message/message_type_enum"
    "kama_chat_server/pkg/util/random"
    "kama_chat_server/pkg/zlog"
    "log"
    "strings"
    "sync"
    "time"
)

// ============================================================
// Server - Channel模式服务器结构
// ============================================================
type Server struct {
    Clients  map[string]*Client  // 在线用户列表，key为用户UUID
    mutex    *sync.Mutex         // 并发锁，保护Clients map
    Transmit chan []byte         // 消息转发通道
    Login    chan *Client        // 登录通道
    Logout   chan *Client        // 退出登录通道
}

// ============================================================
// ChatServer - 全局Server实例
// ============================================================
var ChatServer *Server

// ============================================================
// init() - 初始化Server
// ============================================================
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

// ============================================================
// normalizePath - 标准化头像路径
// ============================================================
// 将 https://127.0.0.1:8000/static/xxx 转为 /static/xxx
func normalizePath(path string) string {
    if path == "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" {
        return path
    }
    staticIndex := strings.Index(path, "/static/")
    if staticIndex < 0 {
        log.Println(path)
        zlog.Error("路径不合法")
    }
    return path[staticIndex:]
}

// ============================================================
// Close - 关闭Server
// ============================================================
func (s *Server) Close() {
    close(s.Login)
    close(s.Logout)
    close(s.Transmit)
}

// ============================================================
// SendClientToLogin - 将Client发送到登录通道
// ============================================================
func (s *Server) SendClientToLogin(client *Client) {
    s.mutex.Lock()
    s.Login <- client
    s.mutex.Unlock()
}

// ============================================================
// SendClientToLogout - 将Client发送到登出通道
// ============================================================
func (s *Server) SendClientToLogout(client *Client) {
    s.mutex.Lock()
    s.Logout <- client
    s.mutex.Unlock()
}

// ============================================================
// SendMessageToTransmit - 将消息发送到转发通道
// ============================================================
func (s *Server) SendMessageToTransmit(message []byte) {
    s.mutex.Lock()
    s.Transmit <- message
    s.mutex.Unlock()
}

// ============================================================
// RemoveClient - 从在线列表移除Client
// ============================================================
func (s *Server) RemoveClient(uuid string) {
    s.mutex.Lock()
    delete(s.Clients, uuid)
    s.mutex.Unlock()
}
```

---

## 六、Server.Start() - 消息处理主循环

### 完整代码

```go
// ============================================================
// Start - 启动Server
// ============================================================
// ★在main.go中调用
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
                // ★用户登录：加入Clients map
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
                // ★用户登出：从Clients map删除
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
                // ★★★ Channel模式: 从Transmit读取消息 ★★★
                var chatMessageReq request.ChatMessageRequest
                if err := json.Unmarshal(data, &chatMessageReq); err != nil {
                    zlog.Error(err.Error())
                }
                
                // 根据消息类型处理
                if chatMessageReq.Type == message_type_enum.Text {
                    // 处理文本消息
                    s.handleTextMessage(chatMessageReq)
                } else if chatMessageReq.Type == message_type_enum.File {
                    // 处理文件消息
                    s.handleFileMessage(chatMessageReq)
                } else if chatMessageReq.Type == message_type_enum.AudioOrVideo {
                    // 处理音视频通话消息
                    s.handleAVMessage(chatMessageReq)
                }
            }
        }
    }
}
```

### handleTextMessage - 处理文本消息

```go
func (s *Server) handleTextMessage(chatMessageReq request.ChatMessageRequest) {
    // 1. 存Message到数据库
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
    
    // 2. 构建响应消息
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
    
    // 3. ★判断是私聊还是群聊
    if message.ReceiveId[0] == 'U' {
        // 发送给User（私聊）
        s.forwardToUser(message, messageRsp, messageBack)
    } else if message.ReceiveId[0] == 'G' {
        // 发送给Group（群聊）
        s.forwardToGroup(message, chatMessageReq, messageRsp, messageBack)
    }
}
```

### forwardToUser - 私聊转发

```go
func (s *Server) forwardToUser(message model.Message, messageRsp respond.GetMessageListRespond, messageBack *MessageBack) {
    s.mutex.Lock()
    
    // 1. 转发给接收者（如果在线）
    if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
        receiveClient.SendBack <- messageBack
    }
    
    // 2. 转发给发送者（消息回显）
    sendClient := s.Clients[message.SendId]
    sendClient.SendBack <- messageBack
    
    s.mutex.Unlock()
    
    // 3. 更新Redis缓存
    s.updateMessageListRedis(message.SendId, message.ReceiveId, messageRsp)
}
```

### forwardToGroup - 群聊转发

```go
func (s *Server) forwardToGroup(message model.Message, chatMessageReq request.ChatMessageRequest, messageRsp respond.GetMessageListRespond, messageBack *MessageBack) {
    // 1. 获取群成员列表
    var group model.GroupInfo
    if res := dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
        zlog.Error(res.Error.Error())
    }
    var members []string
    if err := json.Unmarshal(group.Members, &members); err != nil {
        zlog.Error(err.Error())
    }
    
    s.mutex.Lock()
    
    // 2. 转发给所有群成员
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
    
    // 3. 更新Redis缓存
    s.updateGroupMessageListRedis(message.ReceiveId, messageRsp)
}
```

---

## 七、WebSocket Controller

**文件位置:** `api/v1/ws_controller.go`

### 完整代码

```go
package v1

import (
    "github.com/gin-gonic/gin"
    "kama_chat_server/internal/dto/request"
    "kama_chat_server/internal/service/chat"
    "kama_chat_server/pkg/constants"
    "kama_chat_server/pkg/zlog"
    "net/http"
)

// ============================================================
// WsLogin - WebSocket登录
// ============================================================
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

// ============================================================
// WsLogout - WebSocket登出
// ============================================================
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

## 八、Request结构体

### WsLogoutRequest - WebSocket登出请求

**文件位置:** `internal/dto/request/ws_logout_request.go`

```go
package request

// WsLogoutRequest - WebSocket登出请求
type WsLogoutRequest struct {
    OwnerId string `json:"owner_id"` // 登出用户的uuid
}
```

---

## 九、WebSocket连接生命周期

```
前端登录
    │
    ↓
GET /user/wsLogin?client_id=xxx
    │
    ↓
NewClientInit() 创建Client
    │
    ├── upgrader.Upgrade() 升级为WebSocket
    │
    ├── 注册到 ChatServer.Login 或 KafkaChatServer.Login
    │
    ├── go client.Read()  启动读协程
    │
    └── go client.Write() 启动写协程
            │
            ↓
        消息通信阶段
            │
            ↓
POST /user/wsLogout
    │
    ↓
ClientLogout() 关闭连接
    │
    ├── SendClientToLogout() 发送到登出通道
    │
    ├── client.Conn.Close() 关闭WebSocket
    │
    └── close(client.SendTo/SendBack) 关闭Channel
```

---

## 十、sync.Mutex详解

### 为什么需要mutex？

```go
// ★Clients map被多协程访问：
// 1. Login/Logout处理: 修改Clients map
// 2. 消息转发: 查询Clients获取Client实例
// 3. 多个Client.Read: 同时写入Transmit

// 不加锁会导致:
// - 数据竞争（race condition）
// - 程序崩溃或数据错乱

// 加锁保护:
s.mutex.Lock()           // 加锁
// ... 访问Clients map
s.mutex.Unlock()         // 解锁
```

---

## 十一、消息流转完整流程

### Channel模式 - 单聊流程

```
1. ClientA发送消息
   ↓
2. ClientA.Read() 从WebSocket读取
   ↓
3. ChatServer.SendMessageToTransmit() 写入Transmit Channel
   ↓
4. Server.Start() case data := <-s.Transmit
   ↓
5. handleTextMessage() 处理消息
   ↓
6. 存储到数据库
   ↓
7. forwardToUser() 转发
   ↓
8. ClientB.SendBack <- messageBack
   ↓
9. ClientB.Write() 发送WebSocket
   ↓
10. ClientB收到消息

同时: ClientA.SendBack <- messageBack（消息回显）
```

### Kafka模式 - 消息流程

```
1. ClientA发送消息
   ↓
2. ClientA.Read() 从WebSocket读取
   ↓
3. myKafka.KafkaService.ChatWriter.WriteMessages() 写入Kafka
   ↓
4. KafkaChatServer.Start() 消费Kafka
   ↓
5. 后续处理同Channel模式
```

---

## 十二、离线消息处理

### 设计思路

```
用户在线:
- WebSocket实时推送

用户离线:
- 消息存储到数据库
- 用户登录时调用getMessageList获取历史消息
```

### 代码实现

```go
// 判断用户是否在线
if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
    // 在线 → 推送
    receiveClient.SendBack <- messageBack
} else {
    // 离线 → 只存储，前端登录时调用getMessageList获取
}
```

---

## 十三、路由注册

**文件位置:** `internal/https_server/https_server.go`

```go
func registerRoutes() {
    // WebSocket相关路由
    GE.GET("/user/wsLogin", v1.WsLogin)      // WebSocket登录（GET请求）
    GE.POST("/user/wsLogout", v1.WsLogout)   // WebSocket登出
    
    // ... 其他路由
}
```

---

## 十四、安装依赖

```bash
# WebSocket库
go get github.com/gorilla/websocket

# Kafka Go客户端（如果使用Kafka模式）
go get github.com/segmentio/kafka-go
```

---

## 十五、创建文件步骤

### 步骤1: 创建目录

```bash
mkdir internal/service/chat
```

### 步骤2: 创建文件

创建以下文件：
- `internal/service/chat/client.go` - Client结构体和方法
- `internal/service/chat/server.go` - Channel模式Server
- `internal/service/chat/kafka_server.go` - Kafka模式Server（文档19）
- `api/v1/ws_controller.go` - WebSocket Controller
- `internal/dto/request/ws_logout_request.go` - 登出请求结构体

### 步骤3: 注册路由

在 `internal/https_server/https_server.go` 添加路由

### 步骤4: 启动Server

在main.go中调用：

```go
func main() {
    // ... 其他初始化
    
    // 根据消息模式启动对应的Server
    if kafkaConfig.MessageMode == "channel" {
        go chat.ChatServer.Start()
    } else {
        go chat.KafkaChatServer.Start()
    }
    
    // 启动HTTP服务器
    // ...
}
```

---

## 十六、下一步

WebSocket实现完成后，继续学习：
- **21-聊天室管理.md** - 聊天室功能实现
- **22-分布式系统扩展.md** - 多Server部署和负载均衡
