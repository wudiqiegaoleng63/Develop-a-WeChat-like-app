# GoChat WebSocket 聊天服务设计 — 面试详解

## 一、整体架构

```
┌─────────────┐     WebSocket      ┌─────────────────────────────────────┐
│   前端浏览器  │ ◄───────────────► │          Gin + Gorilla WebSocket     │
│  wsService   │   ws://host/       │                                     │
│  (单例)      │   wsLogin?token=xx │  ┌───────────┐                      │
└──────┬───────┘                    │  │  Upgrader  │ HTTP→WebSocket升级   │
       │                            │  └─────┬─────┘                      │
       │ 发消息(json)               │        │                            │
       │ 收消息(json)               │  ┌─────▼──────┐                     │
       │                            │  │   Client    │ 每个用户一个实例      │
       │                            │  │ ┌────────┐ │                     │
       │                            │  │ │Read协程│ │ 读取消息→写入通道     │
       │                            │  │ └────────┘ │                     │
       │                            │  │ ┌────────┐ │                     │
       │                            │  │ │Write协程│ │ 从通道读取→写回前端   │
       │                            │  │ └────────┘ │                     │
       │                            │  └──────┬─────┘                     │
       │                            │         │                           │
       │                            │  ┌──────▼──────────────────────┐    │
       │                            │  │       Server (核心)          │    │
       │                            │  │  Clients: map[uuid]*Client  │    │
       │                            │  │  Login chan *Client         │    │
       │                            │  │  Logout chan *Client        │    │
       │                            │  │  Transmit chan []byte       │    │
       │                            │  │                             │    │
       │                            │  │  select 三路复用:            │    │
       │                            │  │  case Login → 注册连接       │    │
       │                            │  │  case Logout → 移除连接      │    │
       │                            │  │  case Transmit → 消息路由    │    │
       │                            │  └─────────────────────────────┘    │
       │                            │         │                           │
       │                            │    配置决定模式                      │
       │                            │    ┌────┴────┐                     │
       │                            │  Channel模式  Kafka模式             │
       │                            │  (本进程内)   (消息中间件)            │
       │                            │    ┌────┴────┐                     │
       │                            │  KafkaServer  kafka.Writer/Reader  │
       └────────────────────────────┴───────────────────────────────────┘
```

## 二、核心数据结构

### 2.1 Server（`server.go`）

```go
type Server struct {
    Clients  map[string]*Client  // 在线用户连接表：uuid → Client
    mutex    *sync.Mutex         // 保护 Clients map 的读写锁
    Transmit chan []byte          // 消息转发通道
    Login    chan *Client         // 登录通道
    Logout   chan *Client         // 登出通道
}
```

| 字段 | 作用 | 面试怎么说 |
|------|------|-----------|
| `Clients` | 在线用户表，key 是用户 UUID | 用 map 实现 O(1) 查找在线用户，判断消息接收方是否在线 |
| `mutex` | 互斥锁 | map 不是并发安全的，多个 goroutine 同时读写会 panic，必须加锁 |
| `Transmit` | 消息转发 channel，缓冲区 100 | 生产者-消费者模式，Client.Read 写入，Server.Start 消费处理 |
| `Login/Logout` | 登录登出 channel | 同样是生产者-消费者，Client 初始化时发登录，退出时发登出 |

### 2.2 Client（`client.go`）

```go
type Client struct {
    Conn     *websocket.Conn      // WebSocket 连接
    Uuid     string               // 用户唯一标识
    SendTo   chan []byte          // 客户端→Server 的缓冲通道
    SendBack chan *MessageBack    // Server→客户端 的缓冲通道
}
```

每个用户建立 WebSocket 连接后，创建一个 Client 实例，启动 **Read 和 Write 两个协程**：

```
前端 ──json──► Client.Read() ──► Transmit/Login/Logout ──► Server.Start()
                                                                    │
                                                              消息路由处理
                                                                    │
前端 ◄──json── Client.Write() ◄── SendBack ◄──────────────────────┘
```

> **Q: 为什么一个 Client 要两个协程？**
> A: Read 协程负责从 WebSocket 连接读消息，Write 协程负责往连接写消息。读写分离，互不阻塞。如果用一个协程，读的时候就没法写，写的时候就没法读，消息延迟高。

## 三、连接建立流程

```
1. 前端调用 wsService.connect(token, url)
   → new WebSocket("ws://host/user/wsLogin?token=xxx")

2. Gin 路由匹配 GET /user/wsLogin，经过 JWTAuth 中间件
   → 从 Query 参数解析 token，验证 JWT，提取 uuid

3. WsLogin handler（ws_controller.go）
   → 从 gin.Context 取 uuid

4. NewClientInit（client.go）
   → upgrader.Upgrade() 把 HTTP 连接升级为 WebSocket
   → 创建 Client 实例
   → 把 Client 发送到 Login 通道

5. Server.Start() 的 select 收到 Login
   → s.Clients[client.Uuid] = client  注册到在线用户表
   → 往连接写 "欢迎来到GoChat聊天服务器"

6. 启动两个协程
   → go client.Read()   读前端消息
   → go client.Write()  写消息给前端
```

> **Q: HTTP 怎么变成 WebSocket 的？**
> A: WebSocket 握手是 HTTP 协议发起的，客户端发 `Upgrade: websocket` 请求头，服务端用 `upgrader.Upgrade()` 把这个 HTTP 连接"升级"为 WebSocket 长连接，升级后就不是 HTTP 了，是全双工的 TCP 通道。

## 四、消息路由 — 面试核心

### 4.1 单聊 vs 群聊的区分

消息路由靠 `ReceiveId` 的**首字母**判断：

```go
if message.ReceiveId[0] == 'U' {
    // 单聊：直接查 Clients[receiveId]，在线就转发
} else if message.ReceiveId[0] == 'G' {
    // 群聊：查 GroupInfo.Members，遍历所有成员转发
}
```

| ReceiveId 前缀 | 含义 | 路由方式 |
|----------------|------|---------|
| `U`（如 U20260424...） | 用户 | 直接查在线用户表，点对点转发 |
| `G`（如 G20260425...） | 群组 | 查群成员列表，遍历在线成员逐个转发 |

> **Q: 为什么用首字母而不是枚举字段区分？**
> A: ID 本身就带了类型信息，不需要额外字段。U/G 前缀是 UUID 生成规则的一部分（U=user, G=group, S=session, M=message），任何地方拿到 ID 就能判断类型，省去了类型字段和校验。

### 4.2 单聊消息流

```
用户A发消息给用户B
    │
    ▼
Client(A).Read() 读取消息
    │
    ▼
写入 Transmit 通道
    │
    ▼
Server.Start() 从 Transmit 取出
    │
    ├── 1. 存 MySQL（message 表）
    ├── 2. 判断 ReceiveId[0] == 'U' → 单聊
    ├── 3. 查 Clients[B的uuid]
    │       ├── 在线 → receiveClient.SendBack ← messageBack
    │       └── 离线 → 跳过（消息已在 MySQL，B上线时查库获取）
    ├── 4. 发送者回显：sendClient.SendBack ← messageBack
    └── 5. 更新 Redis 缓存（message_list_SendId_ReceiveId）
```

> **Q: 为什么发送者也要回显？前端不能直接显示吗？**
> A: 因为前端发送的请求结构（ChatMessageRequest）和服务端返回的响应结构（GetMessageListRespond）不同，前端的消息列表存的是响应结构。如果前端直接把请求体塞进列表，字段对不上。所以由服务端统一回显，保证数据结构一致。

### 4.3 群聊消息流

```
用户A在群G发消息
    │
    ▼
Server.Start() 处理
    │
    ├── 1. 存 MySQL
    ├── 2. 判断 ReceiveId[0] == 'G' → 群聊
    ├── 3. 查 GroupInfo.Members → [A, B, C, D]
    ├── 4. 遍历成员：
    │       ├── B 在线 → B.SendBack ← messageBack
    │       ├── C 在线 → C.SendBack ← messageBack
    │       ├── D 离线 → 跳过
    │       └── A（发送者）→ A.SendBack ← messageBack（回显）
    └── 5. 更新 Redis 缓存（group_messagelist_GroupId）
```

> **Q: 群消息要查群成员列表，性能怎么样？**
> A: 当前是每次群消息都查一次 MySQL 的 group_info 表，成员列表存在 JSON 字段里。如果群多、消息频繁，可以做优化：群成员列表缓存到 Redis，群成员变更时更新缓存。当前实现对于中小规模够用。

### 4.4 音视频通话消息流

```go
if chatMessageReq.Type == message_type_enum.AudioOrVideo {
    // 只有特定信令才存库（start_call、receive_call、reject_call）
    if avData.MessageId == "PROXY" && (是通话状态变更) {
        dao.GormDB.Create(&message)  // 持久化通话记录
    }
    // 只转发给接收者，不给发送者回显（否则出现两个 start_call）
    receiveClient.SendBack <- messageBack
    // sendClient.SendBack ← 不回显！
}
```

| 与文本消息的区别 | 原因 |
|----------------|------|
| 不给发送者回显 | 前端发起通话后本地已经显示了，如果服务端再回显就会出现两条 |
| 只有部分信令存库 | ICE candidate 等临时信令不需要持久化，只有状态变更（发起/接听/拒绝）才存 |

## 五、双模式：Channel vs Kafka

### 5.1 Channel 模式（默认）

```
Client.Read() → Transmit channel → Server.Start() 消费处理
```

- 消息在**同一进程内**通过 Go channel 传递
- 单机部署，简单高效，无外部依赖
- 缺点：无法水平扩展，单机挂了全部断连

### 5.2 Kafka 模式

```
Client.Read() → kafka.Writer → Kafka Topic → kafka.Reader → KafkaServer.Start() 消费处理
```

- 消息通过 Kafka 解耦收发
- **多实例部署**：多个服务实例共享一个 Kafka Topic，任意实例的 Client 发消息，所有实例都能消费
- **流量削峰**：高峰期消息堆积在 Kafka，消费者按自己的速度消费，不会压垮服务

### 5.3 两种模式对比

| 维度 | Channel 模式 | Kafka 模式 |
|------|-------------|-----------|
| 部署 | 单机 | 分布式 |
| 依赖 | 无 | Kafka 集群 |
| 延迟 | 极低（内存通道） | 略高（网络 IO + Kafka 存储） |
| 扩展性 | 无法水平扩展 | 多实例部署 |
| 削峰 | 无（channel 满了就丢弃） | 有（Kafka 堆积缓冲） |
| Clients 表 | 每个实例自己维护 | 每个实例自己维护（本地） |

> **Q: Kafka 模式下，用户 A 在实例 1 发消息，用户 B 在实例 2，怎么收到？**
> A: 实例 1 的 Client.Read() 把消息写到 Kafka Topic，实例 2 的 KafkaServer.Start() 从 Topic 消费到消息，查本地 Clients 表找到用户 B 的 Client，通过 SendBack 写回 WebSocket 连接。Kafka 是消息的"中转站"，每个实例只维护自己连接的用户。

> **Q: Kafka 模式下用户连接表怎么同步？**
> A: 没有同步。每个实例只维护连接到自己的用户。消息从 Kafka 消费后，只查本地 Clients 表，用户不在线就跳过（消息已存 MySQL，用户上线时通过 REST API 拉取历史）。

## 六、背压与流量控制

Channel 模式下 Client.Read() 做了背压处理：

```go
// 先排空 SendTo 缓冲区中的旧消息
for len(ChatServer.Transmit) < constants.CHANNEL_SIZE && len(c.SendTo) > 0 {
    sendToMessage := <-c.SendTo
    ChatServer.SendMessageToTransmit(sendToMessage)
}

// 尝试写入 Transmit
if len(ChatServer.Transmit) < constants.CHANNEL_SIZE {
    ChatServer.SendMessageToTransmit(jsonMessage)  // 主通道没满，直接发
} else if len(c.SendTo) < constants.CHANNEL_SIZE {
    c.SendTo <- jsonMessage  // 主通道满了，暂存到客户端缓冲区
} else {
    c.Conn.WriteMessage(websocket.TextMessage, 
        []byte("由于目前同一时间过多用户发送消息，消息发送失败，请稍后重试"))
    // 两个缓冲区都满了，返回错误提示
}
```

三级缓冲策略：

```
Transmit 通道（容量100） → SendTo 通道（容量100） → 丢弃并提示
       1级缓冲                  2级缓冲                3级兜底
```

> **Q: 为什么不直接用无缓冲 channel？**
> A: 无缓冲 channel 要求发送方和接收方同时就绪，否则阻塞。如果 Server.Start() 处理慢，Client.Read() 就会被阻塞，无法继续读新消息。有缓冲 channel + 背压策略可以在短时间内吸收突发流量。

## 七、前端 WebSocket 服务（`websocket.ts`）

### 7.1 单例模式

```typescript
export const wsService = new WebSocketService()  // 全局唯一实例
```

整个应用只维护一个 WebSocket 连接，多个组件通过 `onMessage` 注册回调。

### 7.2 自动重连 + 指数退避

```typescript
private attemptReconnect(): void {
    // 1. 先检查 token 是否过期，过期直接跳登录页
    if (this.isTokenExpired()) {
        window.location.href = '/login'
        return
    }
    // 2. 最多重连 10 次
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return

    // 3. 指数退避：1s, 2s, 4s, 8s, 16s, 30s, 30s...
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    setTimeout(() => this.doConnect(), delay)
}
```

| 机制 | 实现 | 作用 |
|------|------|------|
| **指数退避** | `1000 * 2^n`，上限 30s | 避免断线后疯狂重连，给服务端恢复时间 |
| **最大重连次数** | 10 次 | 防止无限重试，可能是服务端已关闭 |
| **Token 过期检测** | 解码 JWT payload 检查 exp | token 过期了重连也没用，直接跳登录页 |
| **主动断开标记** | `intentionalClose` | 用户主动登出时不触发重连 |

> **Q: 为什么用指数退避而不是固定间隔？**
> A: 固定间隔在服务端压力大时会加重雪崩——所有客户端同一时刻重连。指数退避让重连请求分散开，给服务端缓冲时间，是分布式系统的通用做法。

### 7.3 消息分发

```typescript
this.ws.onmessage = (event) => {
    // 跳过非 JSON 消息（如 "欢迎来到GoChat聊天服务器"）
    if (!event.data.startsWith('{')) return

    const message = JSON.parse(event.data) as ChatMessage
    this.messageHandlers.forEach(handler => handler(message))
}
```

使用**观察者模式**：组件调用 `wsService.onMessage(handler)` 注册回调，收到消息后遍历所有 handler 分发。

## 八、离线消息方案

```
用户B离线时：
  A发消息 → Server处理 → 存MySQL → 查Clients[B]不存在 → 跳过WebSocket推送
                                                ↑
                                        消息已在MySQL中

用户B上线时：
  前端登录 → 调用 POST /session/getUserSessionList（REST API）
          → 调用 POST /message/getMessageList（REST API）
          → 从MySQL读取所有未读消息 → 展示
          → 之后的消息通过 WebSocket 实时推送
```

| 阶段 | 数据来源 | 方式 |
|------|---------|------|
| 登录时加载历史 | MySQL | REST API 一次性拉取 |
| 首次切换聊天对象 | MySQL | REST API 拉取 |
| 再次切换聊天对象 | Redis | 缓存命中直接返回 |
| 在线实时消息 | WebSocket | Server 转发推送 |

## 九、面试速记口诀

1. **连接建立**：前端带 JWT 的 Query 参数连 WebSocket → upgrader.Upgrade 升级 HTTP → 创建 Client → 注册到 Server.Clients
2. **双协程模型**：每个 Client 起 Read（读消息写通道）和 Write（从通道读写回连接）两个协程，读写分离互不阻塞
3. **消息路由**：ReceiveId 首字母 U=单聊点对点转发，G=群聊遍历成员转发，不在线就跳过（消息已存 MySQL）
4. **双模式**：Channel 模式本进程内转发（单机），Kafka 模式通过 Topic 解耦收发（分布式 + 削峰）
5. **背压控制**：Transmit → SendTo → 丢弃，三级缓冲防突发流量
6. **前端重连**：指数退避 + 最大 10 次 + token 过期检测 + 主动断开不重连
7. **离线消息**：在线走 WebSocket 实时推，离线存 MySQL，上线后 REST API 拉取历史
