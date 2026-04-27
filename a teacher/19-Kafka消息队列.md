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
| **Topic** | 消息主题，消息分类 | `chat_message` 聊天消息主题 |
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

## 三、Kafka服务文件

**文件位置:** `internal/service/kafka/kafka_service.go`

根据文档 `4.后端开发.md` 和 `7.项目优化的点.md`：

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
    "kama-chat-server/internal/config"
    "kama-chat-server/pkg/zlog"
)

// ============================================================
// Kafka服务结构体
// ============================================================
// ★包含生产者和消费者
type kafkaService struct {
    ChatWriter *kafka.Writer  // 生产者：写入消息
    ChatReader *kafka.Reader  // 消费者：读取消息
}

// ============================================================
// 全局Kafka服务实例
// ============================================================
var KafkaService *kafkaService
var ctx = context.Background()

// ============================================================
// InitKafka - 初始化Kafka服务
// ============================================================
// ★在main.go或https_server.init()中调用
func InitKafka() {
    // 1. 获取配置
    conf := config.GetConfig()
    
    // 2. 创建Writer（生产者）
    // ★kafka.Writer: 生产者配置
    KafkaService = &kafkaService{
        ChatWriter: &kafka.Writer{
            Addr:                   kafka.TCP(conf.KafkaConfig.HostPort),  // Kafka地址
            Topic:                  conf.KafkaConfig.ChatTopic,            // Topic名称
            Balancer:               &kafka.Hash{},                         // 分区策略
            WriteTimeout:           time.Duration(conf.KafkaConfig.Timeout) * time.Second,
            RequiredAcks:          kafka.RequireNone,                      // ★确认模式
            AllowAutoTopicCreation: false,                                  // 不自动创建Topic
        },
    }
    
    // 3. 创建Reader（消费者）
    // ★kafka.Reader: 消费者配置
    KafkaService.ChatReader = kafka.NewReader(kafka.ReaderConfig{
        Brokers:        []string{conf.KafkaConfig.HostPort},  // Kafka地址列表
        Topic:          conf.KafkaConfig.ChatTopic,           // Topic名称
        CommitInterval: time.Duration(conf.KafkaConfig.Timeout) * time.Second,
        GroupID:        "chat",                               // ★消费者组ID
        StartOffset:    kafka.LastOffset,                     // ★从最新消息开始
    })
    
    zlog.Info("Kafka服务初始化成功")
}

// ============================================================
// WriteMessage - 写入消息到Kafka
// ============================================================
// ★Client发送消息时调用
func WriteMessage(message []byte) error {
    // 1. 获取分区号
    conf := config.GetConfig()
    partition := conf.KafkaConfig.Partition
    
    // 2. 构建Kafka消息
    // ★kafka.Message: 消息结构
    kafkaMsg := kafka.Message{
        Key:   []byte(strconv.Itoa(partition)),  // Key用于分区路由
        Value: message,                          // Value是消息内容（JSON）
    }
    
    // 3. 写入Kafka
    // ★ChatWriter.WriteMessages: 批量写入
    err := KafkaService.ChatWriter.WriteMessages(ctx, kafkaMsg)
    if err != nil {
        zlog.Error("Kafka写入失败: " + err.Error())
        return err
    }
    
    return nil
}

// ============================================================
// ReadMessage - 从Kafka读取消息
// ============================================================
// ★Server启动时在协程中调用
func ReadMessage() (kafka.Message, error) {
    // ★ChatReader.ReadMessage: 读取一条消息（阻塞）
    msg, err := KafkaService.ChatReader.ReadMessage(ctx)
    if err != nil {
        zlog.Error("Kafka读取失败: " + err.Error())
        return kafka.Message{}, err
    }
    
    return msg, nil
}
```

---

## 四、RequiredAcks详解

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

## 五、StartOffset详解

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

## 六、Kafka配置添加

### 更新config_local.toml

```toml
# ----------------------
# [kafkaConfig] Kafka配置
# ----------------------
[kafkaConfig]
messageMode = "kafka"          # channel 或 kafka
hostPort = "127.0.0.1:9092"    # Kafka地址
chatTopic = "chat_message"     # Topic名称
partition = 0                  # 分区号
timeout = 1                    # 超时秒数
```

### 更新config.go

```go
// KafkaConfig - Kafka配置
type KafkaConfig struct {
    MessageMode string `toml:"messageMode"` // channel 或 kafka
    HostPort    string `toml:"hostPort"`    // Kafka地址
    ChatTopic   string `toml:"chatTopic"`   // Topic名称
    Partition   int    `toml:"partition"`   // 分区号
    Timeout     int    `toml:"timeout"`     // 超时秒数
}

// Config - 总配置结构体
type Config struct {
    MainConfig   MainConfig   `toml:"mainConfig"`
    MysqlConfig  MysqlConfig  `toml:"mysqlConfig"`
    RedisConfig  RedisConfig  `toml:"redisConfig"`
    KafkaConfig  KafkaConfig  `toml:"kafkaConfig"`  // ★新增
    LogConfig    LogConfig    `toml:"logConfig"`
}
```

---

## 七、Channel模式 vs Kafka模式

### 模式对比

文档原文（`7.项目优化的点.md`）：

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

### Client.Read() - 写入Kafka

```go
// WebSocket Client读取消息 → 写入Kafka
func (c *Client) Read() {
    for {
        // 1. 从WebSocket读取消息（阻塞）
        _, jsonMessage, err := c.Conn.ReadMessage()
        if err != nil {
            c.Conn.Close()
            return
        }
        
        // 2. ★写入Kafka
        kafka.KafkaService.WriteMessage(jsonMessage)
    }
}
```

### Server消费Kafka

```go
// Server消费Kafka消息
func (k *KafkaServer) Start() {
    // ★启动Kafka消费协程
    go func() {
        for {
            // 1. 从Kafka读取消息（阻塞）
            kafkaMessage, err := kafka.KafkaService.ReadMessage()
            if err != nil {
                continue
            }
            
            // 2. 处理消息（存储 + 转发）
            HandleMessage(kafkaMessage.Value)
        }
    }()
}
```

---

## 九、创建Kafka Topic

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

### ★CreateTopic函数说明

参考文档还提供了CreateTopic函数（行715-742），可通过代码创建Topic:

```go
// CreateTopic 创建topic
func (k *kafkaService) CreateTopic() {
    // 如果已经有topic了，就不创建了
    kafkaConfig := myconfig.GetConfig().KafkaConfig
    k.KafkaConn, err = kafka.Dial("tcp", kafkaConfig.HostPort)
    // ... 配置并创建Topic
}
```

注意: CreateTopic只能执行1次，当已创建该Topic后，再次创建会报错。
本教程采用手动创建方式。

---

## 十、安装依赖

```bash
go get github.com/segmentio/kafka-go
```

---

## 十一、创建文件步骤

### 步骤1: 安装并启动Kafka

按照上面的步骤安装和启动Kafka服务

### 步骤2: 创建Topic

创建 `chat_message` Topic

### 步骤3: 更新配置文件

在 `config_local.toml` 添加Kafka配置

### 步骤4: 更新config.go

添加KafkaConfig结构体

### 步骤5: 创建Kafka服务

创建 `internal/service/kafka/kafka_service.go`

---

## 十二、下一步

Kafka消息队列理解后，继续学习：
- **19-WebSocket高并发.md** - WebSocket Server和Client实现
- **20-聊天室管理.md** - 聊天室功能实现