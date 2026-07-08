# GoChat 双消息路由模式设计 — 面试详解

## 一、为什么需要两种模式

| 维度 | Channel 模式 | Kafka 模式 |
|------|-------------|-----------|
| 部署方式 | 单机 | 分布式多实例 |
| 适用场景 | 开发/小规模 | 生产/大规模 |
| 外部依赖 | 无 | Kafka 集群 |
| 核心区别 | 消息在进程内传递 | 消息通过 Kafka 中间件传递 |

**一句话概括**：Channel 模式简单直接，Kafka 模式解耦可扩展。通过配置 `kafkaConfig.messageMode` 一键切换，业务逻辑无需改动。

## 二、模式选择入口（`main.go`）

```go
func main() {
    conf := config.GetConfig()
    kafkaConfig := conf.KafkaConfig

    // 1. Kafka 模式需要初始化 Kafka 连接
    if kafkaConfig.MessageMode == "kafka" {
        kafka.KafkaService.KafkaInit()
    }

    // 2. 根据模式启动对应的 Server
    if kafkaConfig.MessageMode == "channel" {
        go chat.ChatServer.Start()
    } else {
        go chat.KafkaChatServer.Start()
    }

    // 3. 优雅关闭
    if kafkaConfig.MessageMode == "kafka" {
        kafka.KafkaService.KafkaClose()
    }
    chat.ChatServer.Close()
}
```

启动时根据配置决定走哪条路径，两种模式的 Server 共享同一个 `Client` 结构体和 Read/Write 协程逻辑。

## 三、Channel 模式设计

### 3.1 架构图

```
┌────────────────────────── 单个 Go 进程 ──────────────────────────┐
│                                                                  │
│  ┌────────┐     ┌────────┐                                      │
│  │Client A│     │Client B│     Clients: map[uuid]*Client        │
│  │ Read() │     │ Read() │                                      │
│  │ Write()│     │ Write()│                                      │
│  └───┬────┘     └───┬────┘                                      │
│      │               │                                           │
│      │  发消息        │  发消息                                    │
│      ▼               ▼                                           │
│  ┌──────────────────────────┐                                    │
│  │   Transmit chan []byte   │  ← 所有 Client 共享这一个通道       │
│  │      (容量 100)          │                                    │
│  └────────────┬─────────────┘                                    │
│               │                                                  │
│               ▼                                                  │
│  ┌──────────────────────────┐                                    │
│  │     Server.Start()       │  ← 单协程 select 消费             │
│  │                          │                                    │
│  │  select {                │                                    │
│  │    case <-Login:         │  注册连接                          │
│  │    case <-Logout:        │  移除连接                          │
│  │    case <-Transmit:      │  消息路由                          │
│  │  }                       │                                    │
│  └────────────┬─────────────┘                                    │
│               │                                                  │
│      ┌────────┴────────┐                                        │
│      ▼                 ▼                                        │
│  Client A.SendBack  Client B.SendBack                           │
│      │                 │                                        │
│      ▼                 ▼                                        │
│  Client A.Write()   Client B.Write()                            │
│      │                 │                                        │
│      ▼                 ▼                                        │
│   前端A浏览器        前端B浏览器                                  │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 3.2 Server 结构体

```go
type Server struct {
    Clients  map[string]*Client  // 在线用户连接表
    mutex    *sync.Mutex         // 并发安全锁
    Transmit chan []byte          // 消息转发通道（缓冲100）
    Login    chan *Client         // 登录通道（缓冲100）
    Logout   chan *Client         // 登出通道（缓冲100）
}
```

三个 channel 的职责：

| Channel | 生产者 | 消费者 | 数据类型 | 作用 |
|---------|--------|--------|---------|------|
| `Transmit` | Client.Read() | Server.Start() | `[]byte`（JSON消息） | 承载所有聊天消息 |
| `Login` | NewClientInit() | Server.Start() | `*Client` | 新连接注册 |
| `Logout` | ClientLogout() | Server.Start() | `*Client` | 连接断开移除 |

### 3.3 Server.Start() 核心逻辑

```go
func (s *Server) Start() {
    for {
        select {
        case client := <-s.Login:
            // 注册到在线用户表
            s.Clients[client.Uuid] = client
            client.Conn.WriteMessage(websocket.TextMessage, []byte("欢迎来到GoChat聊天服务器"))

        case client := <-s.Logout:
            // 从在线用户表移除
            delete(s.Clients, client.Uuid)
            client.Conn.WriteMessage(websocket.TextMessage, []byte("已退出登录"))

        case data := <-s.Transmit:
            // 消息路由核心
            // 1. 反序列化 → ChatMessageRequest
            // 2. 存 MySQL（持久化）
            // 3. 按 ReceiveId[0] 路由：
            //    'U' → 单聊，查 Clients[receiveId] 直接转发
            //    'G' → 群聊，查群成员遍历转发
            // 4. 更新 Redis 缓存
        }
    }
}
```

**关键点**：`select` 三路复用，单协程串行处理，天然无并发竞争问题。

> **Q: 为什么用 select 而不是多个 goroutine 分别处理？**
> A: 三个 channel 共享 `Clients` map，如果多个 goroutine 并发读写 map 会 panic。用 select 单协程消费，避免加锁开销，也避免竞态条件。Login/Logout 操作本身就很少，和 Transmit 放一起不会互相阻塞。

### 3.4 Client.Read() 如何写入 Transmit

```go
func (c *Client) Read() {
    for {
        _, jsonMessage, err := c.Conn.ReadMessage()
        // ...

        if messageMode == "channel" {
            // 三级背压策略
            // 1. 先排空 SendTo 缓冲区中的旧消息
            for len(ChatServer.Transmit) < CHANNEL_SIZE && len(c.SendTo) > 0 {
                sendToMessage := <-c.SendTo
                ChatServer.SendMessageToTransmit(sendToMessage)
            }
            // 2. Transmit 没满 → 直接写入
            if len(ChatServer.Transmit) < CHANNEL_SIZE {
                ChatServer.SendMessageToTransmit(jsonMessage)
            // 3. Transmit 满了 → 暂存 SendTo 缓冲区
            } else if len(c.SendTo) < CHANNEL_SIZE {
                c.SendTo <- jsonMessage
            // 4. 都满了 → 丢弃并提示
            } else {
                c.Conn.WriteMessage(websocket.TextMessage, []byte("消息发送失败，请稍后重试"))
            }
        }
    }
}
```

### 3.5 Channel 模式的局限性

| 局限 | 原因 |
|------|------|
| **无法水平扩展** | Clients map 在进程内存中，其他实例看不到 |
| **单点故障** | 进程挂了，所有连接断开，消息丢失（未持久化的部分） |
| **无削峰** | channel 满了只能丢弃 |

这些正是引入 Kafka 模式要解决的问题。

## 四、Kafka 模式设计

### 4.1 架构图

```
┌─── 实例1 ───┐                    ┌─── 实例2 ───┐
│             │                    │             │
│ Client A    │                    │ Client B    │
│ Read()      │                    │ Read()      │
│ Write()     │                    │ Write()     │
│             │                    │             │
│ KafkaServer │                    │ KafkaServer │
│ Clients map │                    │ Clients map │
│ (只有A)     │                    │ (只有B)     │
└──────┬──────┘                    └──────┬──────┘
       │ 写消息                           │ 写消息
       ▼                                  ▼
┌──────────────────────────────────────────────────┐
│                   Kafka Cluster                   │
│                                                   │
│  Topic: chat                                      │
│  ┌─────────┬─────────┬─────────┐                 │
│  │Partition0│Partition1│Partition2│               │
│  │ msg1    │ msg3    │ msg5    │                 │
│  │ msg2    │ msg4    │ msg6    │                 │
│  └─────────┴─────────┴─────────┘                 │
└──────────────┬───────────────────┬───────────────┘
               │ 消费消息           │ 消费消息
               ▼                    ▼
       ┌─── 实例1 ───┐      ┌─── 实例2 ───┐
       │ KafkaServer │      │ KafkaServer │
       │   Start()   │      │   Start()   │
       │             │      │             │
       │ 消费到消息：  │      │ 消费到消息：  │
       │ receiveId=B │      │ receiveId=B │
       │ 查Clients:  │      │ 查Clients:  │
       │ B不在线(跳过)│      │ B在线→转发   │
       └─────────────┘      └─────────────┘
```

### 4.2 KafkaServer 结构体

```go
type KafkaServer struct {
    Clients map[string]*Client  // 本实例的在线用户连接表
    mutex   *sync.Mutex         // 并发安全锁
    Login   chan *Client        // 登录通道
    Logout  chan *Client        // 登出通道
}
```

**与 Channel 模式 Server 的区别**：没有 `Transmit` 通道！消息不经过进程内 channel，而是走 Kafka。

### 4.3 Kafka 初始化（`kafka_service.go`）

```go
type kafkaService struct {
    ChatWriter *kafka.Writer   // 生产者：Client.Read() 写消息
    ChatReader *kafka.Reader   // 消费者：KafkaServer.Start() 读消息
    KafkaConn  *kafka.Conn     // 管理连接
}
```

```go
func (k *kafkaService) KafkaInit() {
    kafkaConfig := config.KafkaConfig

    // 生产者：往 chat Topic 写消息
    k.ChatWriter = &kafka.Writer{
        Addr:  kafka.TCP(kafkaConfig.HostPort),
        Topic: kafkaConfig.ChatTopic,       // "chat"
        Balancer: &kafka.Hash{},            // 按 Key 哈希选分区
    }

    // 消费者：从 chat Topic 读消息
    k.ChatReader = kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{kafkaConfig.HostPort},
        Topic:    kafkaConfig.ChatTopic,     // "chat"
        GroupID:  "chat",                    // 消费者组
        StartOffset: kafka.LastOffset,       // 从最新消息开始消费
    })
}
```

| 组件 | 作用 |
|------|------|
| `ChatWriter` | Client.Read() 读到消息后写入 Kafka Topic |
| `ChatReader` | KafkaServer.Start() 持续消费 Topic 中的消息 |
| `GroupID: "chat"` | 同一消费者组内的多个实例分摊消费，保证每条消息只被一个实例处理 |
| `StartOffset: LastOffset` | 启动后只消费新消息，不消费历史 |

### 4.4 KafkaServer.Start() 核心逻辑

```go
func (k *KafkaServer) Start() {
    // 协程1：持续消费 Kafka Topic
    go func() {
        for {
            kafkaMessage, err := kafka.KafkaService.ChatReader.ReadMessage(ctx)
            data := kafkaMessage.Value

            var chatMessageReq request.ChatMessageRequest
            json.Unmarshal(data, &chatMessageReq)

            // 消息路由逻辑（与 Channel 模式完全相同）
            // 1. 存 MySQL
            // 2. 按 ReceiveId[0] 路由
            //    'U' → 单聊：查 k.Clients[receiveId]
            //    'G' → 群聊：查群成员遍历
            // 3. 更新 Redis 缓存
        }
    }()

    // 主协程：处理登录登出
    for {
        select {
        case client := <-k.Login:
            k.Clients[client.Uuid] = client
        case client := <-k.Logout:
            delete(k.Clients, client.Uuid)
        }
    }
}
```

**与 Channel 模式 Server.Start() 的对比**：

| 维度 | Channel 模式 | Kafka 模式 |
|------|-------------|-----------|
| 消息来源 | `select case <-Transmit` | `kafka.Reader.ReadMessage()` |
| 登录登出 | 同一个 select 处理 | 单独的 select 处理（主协程） |
| 消息消费 | 串行，select 分支 | 串行，独立协程消费 |
| 消息路由逻辑 | **完全相同** | **完全相同** |

### 4.5 Client.Read() 在 Kafka 模式下的行为

```go
func (c *Client) Read() {
    for {
        _, jsonMessage, err := c.Conn.ReadMessage()
        // ...

        if messageMode == "channel" {
            // Channel 模式：写入 Transmit channel
        } else {
            // Kafka 模式：写入 Kafka Topic
            myKafka.KafkaService.ChatWriter.WriteMessages(ctx, kafka.Message{
                Key:   []byte(strconv.Itoa(config.KafkaConfig.Partition)),
                Value: jsonMessage,
            })
        }
    }
}
```

| 模式 | Client.Read() 写入目标 |
|------|----------------------|
| Channel | `ChatServer.Transmit` channel |
| Kafka | `KafkaService.ChatWriter` → Kafka Topic |

### 4.6 消息流转对比

**Channel 模式**：

```
A发消息 → Client(A).Read() → Transmit channel → Server.Start() 消费 → Client(B).Write() → B收到
                           ↑                                                    ↑
                           └────── 同一进程内，内存传递 ──────────────────────────┘
```

**Kafka 模式**：

```
A发消息 → Client(A).Read() → Kafka Writer → Kafka Topic → Kafka Reader → KafkaServer.Start() → Client(B).Write() → B收到
                              ↑              ↑            ↑               ↑
                              └── 实例1 ──┘  └── Kafka ──┘  └── 实例2 ──┘
```

## 五、两种模式的共同设计

### 5.1 共享 Client 和消息路由

两种模式共享的内容：

| 共享组件 | 文件 | 说明 |
|----------|------|------|
| `Client` 结构体 | `client.go` | Conn/Uuid/SendTo/SendBack 完全一样 |
| `Client.Read()` | `client.go` | 读取消息后根据 mode 分发 |
| `Client.Write()` | `client.go` | 从 SendBack 读消息写回 WebSocket，完全一样 |
| `NewClientInit()` | `client.go` | 创建 Client，根据 mode 发送到对应 Server 的 Login 通道 |
| 消息路由逻辑 | `server.go` / `kafka_server.go` | 单聊/群聊/音视频 路由代码几乎相同 |

### 5.2 防伪造身份

```go
// Client.Read() 中
message.SendId = c.Uuid  // 强制使用认证后的 UUID
```

无论哪种模式，都**用 JWT 认证后的 UUID 覆盖消息中的 SendId**，防止客户端伪造身份。即使前端传了别人的 SendId，也会被覆盖为连接认证时的真实用户。

### 5.3 消息持久化顺序

两种模式都是**先存 MySQL，再转发 WebSocket**：

```
收到消息 → 1. 存 MySQL（status=Unsent）
         → 2. WebSocket 转发给接收方
         → 3. Client.Write() 成功后更新 status=Sent
```

这样即使转发时服务崩溃，消息也不会丢失，接收方上线后能从数据库拉取。

## 六、Kafka 模式的关键面试问题

> **Q: 两个实例都从同一个 Topic 消费，消息会不会被重复处理？**
> A: 不会。两个实例的 ChatReader 属于同一个消费者组（`GroupID: "chat"`），Kafka 保证同一条消息只被组内一个消费者处理。如果实例 1 消费了 partition 0 的消息，实例 2 就不会收到。

> **Q: 用户 A 在实例 1，用户 B 在实例 2，A 给 B 发消息，B 怎么收到？**
> A: A 的 Client.Read() 把消息写到 Kafka Topic。B 所在的实例 2 的 KafkaServer.Start() 从 Topic 消费到这条消息，查本地 Clients 表发现 B 在线，通过 B.SendBack 写回 WebSocket 连接。实例 1 消费到这条消息时，查本地 Clients 发现 B 不在线，跳过。

> **Q: 如果两个实例都消费到了同一条消息，B 会不会收到两次？**
> A: 不会。同消费者组内 Kafka 保证每条消息只被一个消费者处理。即使用不同消费者组导致重复消费，也只有一个实例的 Clients 表里有 B 的连接，另一个实例查不到 B，会跳过。

> **Q: Kafka 挂了怎么办？**
> A: 当前实现没有做降级。ChatWriter.WriteMessages 失败只会打日志，消息丢失。生产环境可以做降级：Kafka 不可用时自动切换到 Channel 模式，或者把消息暂存本地队列等 Kafka 恢复后重发。

> **Q: 为什么 Channel 模式的 Server.Start() 用单协程 select，Kafka 模式却把消费放在独立协程？**
> A: Channel 模式下 Login/Logout/Transmit 三个通道通过 select 统一调度，因为都在同一个进程内，事件频率可控。Kafka 模式下消费是阻塞式 IO（ReadMessage 会一直等待），如果放在主协程的 select 里，会阻塞 Login/Logout 的处理。所以把 Kafka 消费放在独立协程，Login/Logout 放主协程，互不影响。

> **Q: Kafka 模式怎么实现削峰？**
> A: 突发大量消息时，ChatWriter 快速写入 Kafka Topic，消息堆积在 Kafka 中。KafkaServer 的 ChatReader 按自己的速度消费，不会压垮业务处理逻辑。消费者可以根据消费延迟指标动态扩容实例，加快消费速度。

## 七、面试速记口诀

1. **切换方式**：配置 `messageMode = "channel"/"kafka"` 一键切换，业务逻辑完全不变
2. **Channel 模式**：Transmit channel 串联 Read → Start → Write，单进程内内存传递，简单高效
3. **Kafka 模式**：Read → Kafka Writer → Topic → Kafka Reader → Start → Write，消息解耦，多实例部署
4. **共享设计**：Client/Read/Write/消息路由逻辑完全共用，只有消息分发目标不同
5. **Kafka 削峰**：消息堆积在 Topic，消费者按自己速度消费，不压垮业务
6. **Kafka 消费组**：同 GroupID 保证每条消息只被一个实例处理，不会重复
7. **安全防伪**：Read() 中强制用 JWT 认证的 UUID 覆盖 SendId，无论哪种模式都防伪造
