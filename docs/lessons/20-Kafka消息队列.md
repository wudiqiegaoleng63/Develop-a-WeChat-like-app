# 教学文档 19: Kafka消息队列（★重点）

### 前置条件：基础架构已完成

Kafka是消息队列扩展，需要先完成基础开发顺序：

```
Model → DAO → Service → Controller → 路由注册 → WebSocket → Kafka

①-⑤: 基础架构（01-09文档）已完成
⑥ WebSocket: 实时通信基础（20文档）
⑦ Kafka: ★当前文档★ 高并发消息缓冲
```

## 一、为什么需要Kafka？

### 问题背景

**场景:** 10000用户同时发送消息

```
没有Kafka (Channel模式):
┌─────────┐   channel满   ┌─────────┐
│ Client  │ ────X──────── │ Server  │  消息发送失败
└─────────┘                └─────────┘
问题: Channel容量有限，高峰期阻塞

有Kafka:
┌─────────┐   写入Kafka   ┌─────────┐   消费消息   ┌─────────┐
│ Client  │ ──────────── │ Kafka   │ ─────────── │ Server  │
└─────────┘               └─────────┘             └─────────┘
    ↑ 消息不丢失            ↑ 流量缓冲              ↑ 稳定处理
```

### Kafka核心价值

1. **流量削峰** - 高峰期消息写入Kafka缓冲，Server按自己节奏消费
2. **系统解耦** - Client和Server通过Kafka中转，不直接依赖
3. **消息持久化** - Kafka存储消息，Server宕机不丢消息

---

## 二、Kafka基础概念

### 核心概念

| 概念 | 说明 | 本项目应用 |
|------|------|-----------|
| **Topic** | 消息主题，消息分类 | `chat_message` 聊天消息主题、`login` 登录主题、`logout` 登出主题 |
| **Partition** | 分区，消息有序存储 | 单分区，消息按时间顺序 |
| **Producer** | 生产者，写入消息 | Client发送消息 → 写入Kafka |
| **Consumer** | 消费者，读取消息 | Server从Kafka读取 → 转发 |

### 消息流转图

```
                    Topic: chat_message
                    ┌────────────────────┐
                    │ Partition 0        │
                    │ ┌────────────────┐ │
Client1 ─写消息──→ │ │ Message 1      │ │ ─读消息──→ Server
Client2 ─写消息──→ │ │ Message 2      │ │
Client3 ─写消息──→ │ │ Message 3      │ │
                    │ │ ...            │ │
                    │ └────────────────┘ │
                    └────────────────────┘
```

---

## 三、Kafka配置添加

### 更新config_local.toml

```toml
# ----------------------
# [kafkaConfig] Kafka配置
# ----------------------
[kafkaConfig]
messageMode = "channel"          # 消息模式：channel 或 kafka
hostPort = "127.0.0.1:9092"      # Kafka地址，多个服务器用逗号分隔
loginTopic = "login"             # 登录Topic
chatTopic = "chat_message"       # 聊天消息Topic
logoutTopic = "logout"           # 登出Topic
partition = 0                    # 分区号
timeout = 1                      # 超时秒数，单位秒
```

### 更新config.go

```go
// internal/config/config.go

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

// Config - 总配置结构体
type Config struct {
    MainConfig      `toml:"mainConfig"`
    MysqlConfig     `toml:"mysqlConfig"`
    RedisConfig     `toml:"redisConfig"`
    AuthCodeConfig  `toml:"authCodeConfig"`
    LogConfig       `toml:"logConfig"`
    KafkaConfig     `toml:"kafkaConfig"`  // ★新增
    StaticSrcConfig `toml:"staticSrcConfig"`
}
```

### 导入time包

```go
import (
    "github.com/BurntSushi/toml"
    "log"
    "time"  // ★KafkaConfig.Timeout使用time.Duration类型
)
```

---

## 四、Kafka服务文件

**文件位置:** `internal/service/kafka/kafka_service.go`

### 完整代码（带详细注释）

```go
package kafka

import (
    "context"
    "time"
    "github.com/segmentio/kafka-go"
    myconfig "gochat/internal/config"
    "gochat/pkg/zlog"
)

// 全局上下文
var ctx = context.Background()

// Kafka服务结构体
// ChatWriter: 生产者，写入消息
// ChatReader: 消费者，读取消息
// KafkaConn: Kafka连接，用于创建Topic
type kafkaService struct {
    ChatWriter *kafka.Writer
    ChatReader *kafka.Reader
    KafkaConn  *kafka.Conn
}

// 全局Kafka服务实例
var KafkaService = new(kafkaService)

// KafkaInit 初始化Kafka服务
// 在main.go中根据messageMode配置决定是否调用
func (k *kafkaService) KafkaInit() {
    //k.CreateTopic()  // 已有Topic时不需要重复创建
    
    kafkaConfig := myconfig.GetConfig().KafkaConfig
    
    // 创建Writer（生产者）
    k.ChatWriter = &kafka.Writer{
        Addr:                   kafka.TCP(kafkaConfig.HostPort),
        Topic:                  kafkaConfig.ChatTopic,
        Balancer:               &kafka.Hash{},
        WriteTimeout:           kafkaConfig.Timeout * time.Second,
        RequiredAcks:           kafka.RequireNone,
        AllowAutoTopicCreation: false,
    }
    
    // 创建Reader（消费者）
    k.ChatReader = kafka.NewReader(kafka.ReaderConfig{
        Brokers:        []string{kafkaConfig.HostPort},
        Topic:          kafkaConfig.ChatTopic,
        CommitInterval: kafkaConfig.Timeout * time.Second,
        GroupID:        "chat",
        StartOffset:    kafka.LastOffset,
    })
}

// KafkaClose 关闭Kafka服务
func (k *kafkaService) KafkaClose() {
    if err := k.ChatWriter.Close(); err != nil {
        zlog.Error(err.Error())
    }
    if err := k.ChatReader.Close(); err != nil {
        zlog.Error(err.Error())
    }
}

// CreateTopic 创建Topic
// 如果已经有Topic了，就不创建了。只能执行1次，再次创建会报错。
func (k *kafkaService) CreateTopic() {
    kafkaConfig := myconfig.GetConfig().KafkaConfig
    chatTopic := kafkaConfig.ChatTopic
    
    // 连接至任意kafka节点
    var err error
    k.KafkaConn, err = kafka.Dial("tcp", kafkaConfig.HostPort)
    if err != nil {
        zlog.Error(err.Error())
    }
    
    topicConfigs := []kafka.TopicConfig{
        {
            Topic:             chatTopic,
            NumPartitions:     kafkaConfig.Partition,
            ReplicationFactor: 1,
        },
    }
    
    // 创建topic
    if err = k.KafkaConn.CreateTopics(topicConfigs...); err != nil {
        zlog.Error(err.Error())
    }
}
```

### 导入路径说明

```go
import (
    "context"
    "time"
    "github.com/segmentio/kafka-go"
    myconfig "gochat/internal/config"  // ★用别名避免与config包名冲突
    "gochat/pkg/zlog"
)
```

---

## 五、RequiredAcks详解

### 确认模式

```go
RequiredAcks: kafka.RequireNone  // 当前配置
```

| 模式 | 值 | 说明 | 性能 |
|------|----|------|------|
| `RequireNone` | 0 | 不等待确认，最低延迟 | 最高 |
| `RequireOne` | 1 | 等待至少一个副本确认 | 中等 |
| `RequireAll` | -1 | 等待所有副本确认 | 最低 |

### 本项目为什么用RequireNone？

```
场景: 聊天消息

消息丢失风险:
- RequireNone: 极低概率丢失（Kafka宕机瞬间）
- 对于聊天消息，极低概率丢失可接受

性能考量:
- 10000用户同时发消息，不等待确认更快
- 如用RequireAll，高峰期可能阻塞
```

---

## 六、StartOffset详解

### 偏移量模式

```go
StartOffset: kafka.LastOffset  // 当前配置
```

| 模式 | 说明 | 应用场景 |
|------|------|---------|
| `FirstOffset` | 从最早消息开始 | 需要历史消息 |
| `LastOffset` | 从最新消息开始 | 只处理新消息 ★ |

### 本项目为什么用LastOffset？

```
Server启动时:
- 如果用FirstOffset: 会处理所有历史消息（重复）
- 用LastOffset: 只处理启动后的新消息

离线消息处理:
- 用户登录时调用getMessageList获取历史消息
- Kafka只处理实时消息
```

---

## 七、Channel模式 vs Kafka模式

### 模式对比

```
messageMode配置:
- "channel": 小规模使用，内存Channel传递
- "kafka": 大规模使用，Kafka消息队列

切换方式:
只需修改配置文件，代码自动适配
```

### Channel模式 - Server结构体

**文件位置:** `internal/service/chat/server.go`

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

// Server Channel模式服务器结构体
type Server struct {
    Clients  map[string]*Client  // 在线用户列表，key为用户UUID
    mutex    *sync.Mutex         // 并发锁，保护Clients map
    Transmit chan []byte         // 消息转发通道
    Login    chan *Client        // 登录通道
    Logout   chan *Client        // 退出登录通道
}

// ChatServer 全局Server实例
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

// Close 关闭Server
func (s *Server) Close() {
    close(s.Login)
    close(s.Logout)
    close(s.Transmit)
}

// SendClientToLogin 将Client发送到登录通道
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
```

### Channel模式消息流转

```
Client.Read() 
    ↓ 写入
ChatServer.Transmit (chan []byte)
    ↓ Server.Start() 从channel读取
case data := <-s.Transmit
    ↓ 处理消息
存数据库 → 转发给在线用户 → 更新Redis

问题: Channel容量有限（constants.CHANNEL_SIZE），高峰期阻塞
```

### Kafka模式 - KafkaServer结构体

**文件位置:** `internal/service/chat/kafka_server.go`

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
    "gochat/internal/service/kafka"
    myredis "gochat/internal/service/redis"
    "gochat/pkg/constants"
    "gochat/pkg/enum/message/message_status_enum"
    "gochat/pkg/enum/message/message_type_enum"
    "gochat/pkg/util/random"
    "gochat/pkg/zlog"
    "log"
    "os"
    "sync"
    "time"
)

// KafkaServer Kafka模式服务器结构体
type KafkaServer struct {
    Clients map[string]*Client  // 在线用户列表，key为用户UUID
    mutex   *sync.Mutex         // 并发锁，保护Clients map
    Login   chan *Client        // 登录通道
    Logout  chan *Client        // 退出登录通道
}

// KafkaChatServer 全局KafkaServer实例
var KafkaChatServer *KafkaServer

var kafkaQuit = make(chan os.Signal, 1)

func init() {
    if KafkaChatServer == nil {
        KafkaChatServer = &KafkaServer{
            Clients: make(map[string]*Client),
            mutex:   &sync.Mutex{},
            Login:   make(chan *Client),
            Logout:  make(chan *Client),
        }
    }
}

// Close 关闭KafkaServer
func (k *KafkaServer) Close() {
    close(k.Login)
    close(k.Logout)
}

// SendClientToLogin 将Client发送到登录通道
func (k *KafkaServer) SendClientToLogin(client *Client) {
    k.mutex.Lock()
    k.Login <- client
    k.mutex.Unlock()
}

// SendClientToLogout 将Client发送到登出通道
func (k *KafkaServer) SendClientToLogout(client *Client) {
    k.mutex.Lock()
    k.Logout <- client
    k.mutex.Unlock()
}

// RemoveClient 从在线列表移除Client
func (k *KafkaServer) RemoveClient(uuid string) {
    k.mutex.Lock()
    delete(k.Clients, uuid)
    k.mutex.Unlock()
}
```

### Kafka模式消息流转

```
Client.Read() 
    ↓ 写入Kafka
Kafka Topic (chat_message)
    ↓ KafkaServer.Start() 消费
kafka.KafkaService.ChatReader.ReadMessage(ctx)
    ↓ 处理消息
存数据库 → 转发给在线用户 → 更新Redis

优势: Kafka Topic容量无限，不阻塞，支持分布式部署
```

### Server结构体对比

| 项目 | Channel Server | Kafka Server |
|------|---------------|-------------|
| 文件 | `internal/service/chat/server.go` | `internal/service/chat/kafka_server.go` |
| Transmit通道 | `Transmit chan []byte` | ❌ 无（Kafka代替） |
| Login通道 | `Login chan *Client` | `Login chan *Client` |
| Logout通道 | `Logout chan *Client` | `Logout chan *Client` |
| 消息处理 | `case data := <-s.Transmit` | `kafka.KafkaService.ChatReader.ReadMessage()` |

---

## 八、Kafka与WebSocket结合

### 完整流程图

```
                    Kafka Topic: chat_message
                    ┌──────────────────────────┐
                    │                          │
┌─────────┐        │ ┌──────────────────────┐ │        ┌─────────┐
│ Client1 │─WebSocket─────→│ Message 1          │ │──WebSocket─────→│ Server  │
│ (发消息)│        │ │ (JSON)              │ │        │ (消费)  │
└─────────┘        │ └──────────────────────┘ │        └─────────┘
                   │                          │              │
┌─────────┐        │ ┌──────────────────────┐ │              │
│ Client2 │───────→│ │ Message 2          │ │──────────────┤
│ (发消息)│        │ │                    │ │              │
└─────────┘        │ └──────────────────────┘ │              │
                   │                          │        ┌─────┴─────┐
                   └──────────────────────────┘        │           │
                                                       │ 转发消息  │
                                                       │           │
                                              ┌────────┴───────────┐
                                              │                    │
                                     ┌────────┴────────┐ ┌────────┴────────┐
                                     │ Client3 (接收)  │ │ Client4 (接收)  │
                                     └─────────────────┘ └─────────────────┘
```

### ChatMessageRequest - WebSocket消息结构体

**文件位置:** `internal/dto/request/chat_message_request.go`

```go
package request

// ChatMessageRequest WebSocket消息请求结构体
// 前端通过WebSocket发送JSON消息，后端反序列化到此结构体
type ChatMessageRequest struct {
    SessionId  string `json:"session_id"`   // 会话ID
    Type       int8   `json:"type"`          // 消息类型（0=文本, 1=语音, 2=文件, 3=通话）
    Content    string `json:"content"`       // 文本内容
    Url        string `json:"url"`           // 文件URL
    SendId     string `json:"send_id"`       // 发送者UUID
    SendName   string `json:"send_name"`     // 发送者昵称
    SendAvatar string `json:"send_avatar"`   // 发送者头像
    ReceiveId  string `json:"receive_id"`    // 接收者UUID（U开头=用户, G开头=群）
    FileSize   string `json:"file_size"`     // 文件大小
    FileType   string `json:"file_type"`     // 文件类型
    FileName   string `json:"file_name"`     // 文件名
    AVdata     string `json:"av_data"`       // 音视频通话数据
}
```

### Client.Read() - 根据messageMode写入Kafka或Channel

```go
// internal/service/chat/client.go

import (
    "context"
    "encoding/json"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "github.com/segmentio/kafka-go"
    "gochat/internal/config"
    "gochat/internal/dao"
    "gochat/internal/dto/request"
    "gochat/internal/model"
    myKafka "gochat/internal/service/kafka"
    "gochat/pkg/constants"
    "gochat/pkg/enum/message/message_status_enum"
    "gochat/pkg/zlog"
    "log"
    "net/http"
    "strconv"
)

// Client结构体
type Client struct {
    Conn     *websocket.Conn
    Uuid     string
    SendTo   chan []byte       // 给server端（Channel模式用）
    SendBack chan *MessageBack // 给前端
}

type MessageBack struct {
    Message []byte
    Uuid    string
}

var messageMode = config.GetConfig().KafkaConfig.MessageMode

var ctx = context.Background()

var upgrader = websocket.Upgrader{
    ReadBufferSize:  2048,
    WriteBufferSize: 2048,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// 读取websocket消息，根据mode选择写入Kafka还是Channel
func (c *Client) Read() {
    zlog.Info("ws read goroutine start")
    for {
        _, jsonMessage, err := c.Conn.ReadMessage()
        if err != nil {
            zlog.Error(err.Error())
            return
        }
        
        var message = request.ChatMessageRequest{}
        if err := json.Unmarshal(jsonMessage, &message); err != nil {
            zlog.Error(err.Error())
        }
        log.Println("接受到消息为: ", jsonMessage)
        
        if messageMode == "channel" {
            // Channel模式：写入Server的Transmit通道
            for len(ChatServer.Transmit) < constants.CHANNEL_SIZE && len(c.SendTo) > 0 {
                sendToMessage := <-c.SendTo
                ChatServer.SendMessageToTransmit(sendToMessage)
            }
            if len(ChatServer.Transmit) < constants.CHANNEL_SIZE {
                ChatServer.SendMessageToTransmit(jsonMessage)
            } else if len(c.SendTo) < constants.CHANNEL_SIZE {
                c.SendTo <- jsonMessage
            } else {
                // Channel满了，提示用户
                c.Conn.WriteMessage(websocket.TextMessage, []byte("由于目前同一时间过多用户发送消息，消息发送失败，请稍后重试"))
            }
        } else {
            // Kafka模式：直接写入Kafka Topic
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
```

### KafkaServer.Start() - 消费Kafka消息

```go
// internal/service/chat/kafka_server.go

type KafkaServer struct {
    Clients map[string]*Client
    mutex   *sync.Mutex
    Login   chan *Client  // 登录通道
    Logout  chan *Client  // 退出登录通道
}

var KafkaChatServer *KafkaServer

func (k *KafkaServer) Start() {
    // ★启动Kafka消费协程：读取chat_message Topic
    go func() {
        for {
            kafkaMessage, err := kafka.KafkaService.ChatReader.ReadMessage(ctx)
            if err != nil {
                zlog.Error(err.Error())
            }
            
            data := kafkaMessage.Value
            var chatMessageReq request.ChatMessageRequest
            if err := json.Unmarshal(data, &chatMessageReq); err != nil {
                zlog.Error(err.Error())
            }
            
            // 根据消息类型处理
            if chatMessageReq.Type == message_type_enum.Text {
                // 1. 存Message到数据库
                message := model.Message{...}
                dao.GormDB.Create(&message)
                
                // 2. 判断是私聊还是群聊（receive_id[0] == 'U' 或 'G'）
                // 3. 转发给在线用户（通过SendBack通道）
                // 4. 更新Redis缓存
            } else if chatMessageReq.Type == message_type_enum.File {
                // 处理文件消息...
            } else if chatMessageReq.Type == message_type_enum.AudioOrVideo {
                // 处理音视频通话消息...
            }
        }
    }()
    
    // ★处理登录/登出
    for {
        select {
        case client := <-k.Login:
            k.mutex.Lock()
            k.Clients[client.Uuid] = client
            k.mutex.Unlock()
            
        case client := <-k.Logout:
            k.mutex.Lock()
            delete(k.Clients, client.Uuid)
            k.mutex.Unlock()
        }
    }
}
```

---

## 九、main.go 中Kafka的初始化与关闭

### main.go完整代码

```go
// cmd/gochat/main.go
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
    
    // ★根据messageMode决定是否初始化Kafka
    if kafkaConfig.MessageMode == "kafka" {
        kafka.KafkaService.KafkaInit()
    }
    
    // ★根据messageMode选择启动Channel或Kafka Server
    if kafkaConfig.MessageMode == "channel" {
        go chat.ChatServer.Start()
    } else {
        go chat.KafkaChatServer.Start()
    }
    
    // 启动HTTP服务器（TLS）
    go func() {
        if err := https_server.GE.RunTLS(fmt.Sprintf("%s:%d", host, port), "证书路径", "密钥路径"); err != nil {
            zlog.Fatal("server running fault")
            return
        }
    }()
    
    // ★等待信号，优雅关闭
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    // ★关闭Kafka服务
    if kafkaConfig.MessageMode == "kafka" {
        kafka.KafkaService.KafkaClose()
    }
    
    chat.ChatServer.Close()
    zlog.Info("关闭服务器...")
    
    // 删除所有Redis键
    if err := myredis.DeleteAllRedisKeys(); err != nil {
        zlog.Error(err.Error())
    }
    
    zlog.Info("服务器已关闭")
}
```

---

## 十、创建Kafka Topic

### 安装Kafka

```bash
# Windows: 下载Kafka二进制包
# https://kafka.apache.org/downloads

# 解压后进入目录
cd kafka_2.13-3.6.0
```

### 启动Zookeeper

```bash
# Kafka依赖Zookeeper
bin/windows/zookeeper-server-start.bat config/zookeeper.properties
```

### 启动Kafka

```bash
bin/windows/kafka-server-start.bat config/server.properties
```

### 创建Topic

```bash
# 创建chat_message Topic
bin/windows/kafka-topics.bat --create \
    --topic chat_message \
    --bootstrap-server localhost:9092 \
    --partitions 1 \
    --replication-factor 1
```

注意: CreateTopic函数只能执行1次，当已创建该Topic后，再次创建会报错。本教程采用手动创建方式。

---

## 十一、安装依赖

```bash
# Kafka Go客户端
go get github.com/segmentio/kafka-go

# WebSocket库
go get github.com/gorilla/websocket
```

---

## 十二、常量与枚举定义

### constants.go - 常量定义

**文件位置:** `pkg/constants/constants.go`

```go
package constants

const (
    CHANNEL_SIZE  = 100            // 通道大小
    SYSTEM_ERROR  = "系统错误，请联系工作人员" // 系统错误
    FILE_MAX_SIZE = 50000          // 文件最大大小
    REDIS_TIMEOUT = 1              // redis timeout（分钟）
)
```

### message_type_enum.go - 消息类型枚举

**文件位置:** `pkg/enum/message/message_type_enum/message_type_enum.go`

```go
package message_type_enum

const (
    Text = iota    // 0 - 文本消息
    Voice          // 1 - 语音消息
    File           // 2 - 文件消息
    AudioOrVideo   // 3 - 音视频通话
)
```

### message_status_enum.go - 消息状态枚举

**文件位置:** `pkg/enum/message/message_status_enum/message_status_enum.go`

```go
package message_status_enum

const (
    Unsent = iota  // 0 - 未发送
    Sent           // 1 - 已发送
)
```

### normalizePath - 头像路径标准化函数

**文件位置:** `internal/service/chat/server.go`

```go
import (
    "log"
    "strings"
    "gochat/pkg/zlog"
)

// normalizePath 标准化头像路径
// 将 https://127.0.0.1:8000/static/xxx 转为 /static/xxx
// 防止IP前缀引入导致前端无法正确加载静态资源
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
```

---

## 十三、Respond结构体定义

### GetMessageListRespond - 私聊消息响应

**文件位置:** `internal/dto/respond/get_message_list_respond.go`

```go
package respond

// GetMessageListRespond 私聊消息响应结构体
type GetMessageListRespond struct {
    SendId     string `json:"send_id"`     // 发送者UUID
    SendName   string `json:"send_name"`   // 发送者昵称
    SendAvatar string `json:"send_avatar"` // 发送者头像
    ReceiveId  string `json:"receive_id"`  // 接收者UUID
    Type       int8   `json:"type"`        // 消息类型
    Content    string `json:"content"`     // 文本内容
    Url        string `json:"url"`         // 文件URL
    FileType   string `json:"file_type"`   // 文件类型
    FileName   string `json:"file_name"`   // 文件名
    FileSize   string `json:"file_size"`   // 文件大小
    CreatedAt  string `json:"created_at"`  // 创建时间
}
```

### GetGroupMessageListRespond - 群聊消息响应

**文件位置:** `internal/dto/respond/get_group_messagelist_respond.go`

```go
package respond

// GetGroupMessageListRespond 群聊消息响应结构体
type GetGroupMessageListRespond struct {
    SendId     string `json:"send_id"`     // 发送者UUID
    SendName   string `json:"send_name"`   // 发送者昵称
    SendAvatar string `json:"send_avatar"` // 发送者头像
    ReceiveId  string `json:"receive_id"`  // 群组UUID
    Type       int8   `json:"type"`        // 消息类型
    Content    string `json:"content"`     // 文本内容
    Url        string `json:"url"`         // 文件URL
    FileType   string `json:"file_type"`   // 文件类型
    FileName   string `json:"file_name"`   // 文件名
    FileSize   string `json:"file_size"`   // 文件大小
    CreatedAt  string `json:"created_at"`  // 创建时间
}
```

### AVMessageRespond - 音视频消息响应

**文件位置:** `internal/dto/respond/av_message_respond.go`

```go
package respond

// AVMessageRespond 音视频通话消息响应结构体
type AVMessageRespond struct {
    SendId     string `json:"send_id"`     // 发送者UUID
    SendName   string `json:"send_name"`   // 发送者昵称
    SendAvatar string `json:"send_avatar"` // 发送者头像
    ReceiveId  string `json:"receive_id"`  // 接收者UUID
    Type       int8   `json:"type"`        // 消息类型（3=音视频通话）
    Content    string `json:"content"`     // 文本内容
    Url        string `json:"url"`         // 文件URL
    FileType   string `json:"file_type"`   // 文件类型
    FileName   string `json:"file_name"`   // 文件名
    FileSize   string `json:"file_size"`   // 文件大小
    CreatedAt  string `json:"created_at"`  // 创建时间
    AVdata     string `json:"av_data"`     // 音视频通话数据（JSON字符串）
}
```

### AVData - 音视频通话数据请求

**文件位置:** `internal/dto/request/av_data_request.go`

```go
package request

// AVData 音视频通话数据结构体
// 用于解析 ChatMessageRequest.AVdata 字段
type AVData struct {
    MessageId string `json:"messageId"` // 消息ID（"PROXY" 表示代理消息）
    Type      string `json:"type"`      // 通话类型：start_call, receive_call, reject_call
}
```

---

## 十四、创建文件步骤

### 步骤1: 安装并启动Kafka

按照上面的步骤安装和启动Kafka服务

### 步骤2: 创建Topic

创建 `chat_message`、`login`、`logout` Topic

### 步骤3: 更新配置文件

在 `config_local.toml` 添加Kafka配置

### 步骤4: 更新config.go

添加KafkaConfig结构体（含LoginTopic、LogoutTopic字段）

### 步骤5: 创建Kafka服务

创建 `internal/service/kafka/kafka_service.go`

### 步骤6: 更新main.go

添加Kafka初始化和关闭逻辑

---

## 十五、下一步

Kafka消息队列理解后，继续学习：
- **20-WebSocket高并发.md** - WebSocket Server和Client实现
- **21-聊天室管理.md** - 聊天室功能实现
