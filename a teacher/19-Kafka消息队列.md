# 教学文档 19: Kafka消息队列（★重点）

### 前置条件：基础架构已完成

Kafka是消息队列扩展，需要先完成基础开发顺序：

```
Model → DAO → Service → Controller → 路由注册 → WebSocket → Kafka

①-⑤: 基础架构（01-09文档）已完成
⑥ WebSocket: 实时通信基础（19文档）
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

// ============================================================
// 导入依赖包
// ============================================================
import (
    "context"
    "time"
    
    // ★Kafka Go客户端库
    "github.com/segmentio/kafka-go"
    
    // ★项目内部依赖
    myconfig "kama_chat_server/internal/config"
    "kama_chat_server/pkg/zlog"
)

// ============================================================
// 全局上下文
// ============================================================
var ctx = context.Background()

// ============================================================
// Kafka服务结构体
// ============================================================
type kafkaService struct {
    ChatWriter *kafka.Writer  // 生产者：写入消息
    ChatReader *kafka.Reader  // 消费者：读取消息
    KafkaConn  *kafka.Conn    // Kafka连接（用于创建Topic）
}

// ============================================================
// 全局Kafka服务实例
// ============================================================
var KafkaService = new(kafkaService)

// ============================================================
// KafkaInit - 初始化Kafka服务
// ============================================================
// ★在main.go中根据messageMode配置决定是否调用
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

// ============================================================
// KafkaClose - 关闭Kafka服务
// ============================================================
func (k *kafkaService) KafkaClose() {
    if err := k.ChatWriter.Close(); err != nil {
        zlog.Error(err.Error())
    }
    if err := k.ChatReader.Close(); err != nil {
        zlog.Error(err.Error())
    }
}

// ============================================================
// CreateTopic - 创建Topic
// ============================================================
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
    myconfig "kama_chat_server/internal/config"  // ★用别名避免与config包名冲突
    "kama_chat_server/pkg/zlog"
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

### Channel模式代码

```go
// Channel模式（小规模）
type Server struct {
    Transmit chan []byte  // 内存Channel
}

// 消息流转:
Client → Transmit Channel → Server → 转发

// 问题:
Channel容量有限（如1000），高峰期阻塞
```

### Kafka模式代码

```go
// Kafka模式（大规模）
// 消息流转:
Client → Kafka Topic → Server消费 → 转发

// 优势:
Kafka Topic容量无限，不阻塞
```

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

### Client.Read() - 根据messageMode写入Kafka或Channel

```go
// internal/service/chat/client.go

// Client结构体
type Client struct {
    Conn     *websocket.Conn
    Uuid     string
    SendTo   chan []byte       // 给server端（Channel模式用）
    SendBack chan *MessageBack // 给前端
}

// 读取websocket消息，根据mode选择写入Kafka还是Channel
func (c *Client) Read() {
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
            KafkaService.ChatWriter.WriteMessages(ctx, kafka.Message{
                Key:   []byte(strconv.Itoa(config.GetConfig().KafkaConfig.Partition)),
                Value: jsonMessage,
            })
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
// cmd/kama-chat-server/main.go
package main

import (
    "fmt"
    "kama_chat_server/internal/config"
    "kama_chat_server/internal/https_server"
    "kama_chat_server/internal/service/chat"
    "kama_chat_server/internal/service/kafka"
    myredis "kama_chat_server/internal/service/redis"
    "kama_chat_server/pkg/zlog"
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

## 十、KafkaServer vs Channel Server 对比

### Server结构体对比

| 项目 | Channel Server | Kafka Server |
|------|---------------|-------------|
| 文件 | `internal/service/chat/server.go` | `internal/service/chat/kafka_server.go` |
| Transmit通道 | `Transmit chan []byte` | ❌ 无（Kafka代替） |
| Login通道 | `Login chan *Client` | `Login chan *Client` |
| Logout通道 | `Logout chan *Client` | `Logout chan *Client` |
| 消息处理 | `case data := <-s.Transmit` | `kafka.KafkaService.ChatReader.ReadMessage()` |

### 核心差异

```
Channel Server:
Client.Read() → Server.Transmit → Server处理 → Server转发

Kafka Server:
Client.Read() → Kafka Topic → KafkaServer.ReadMessage() → KafkaServer处理 → 转发
```

---

## 十一、创建Kafka Topic

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

## 十二、安装依赖

```bash
go get github.com/segmentio/kafka-go
```

---

## 十三、创建文件步骤

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

## 十四、下一步

Kafka消息队列理解后，继续学习：
- **20-WebSocket高并发.md** - WebSocket Server和Client实现
- **21-聊天室管理.md** - 聊天室功能实现
