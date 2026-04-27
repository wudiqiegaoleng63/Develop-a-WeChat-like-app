# 教学文档 20: WebSocket高并发（★重点）

### 前置条件：基础架构已完成

WebSocket是实时通信扩展，需要先完成基础开发顺序：

```
Model → DAO → Service → Controller → 路由注册 → WebSocket

①-⑤: 基础架构（01-09文档）已完成
⑥ WebSocket: ★当前文档★ 实时双向通信
⑦ Kafka: 高并发消息缓冲（18文档）
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

根据文档 `7.项目优化的点.md`：

```
┌────────┐  SendTo(chan)  ┌────────┐  Transmit(chan)  ┌────────┐
│ Client │ ────────────── │ Server │ ─────────────── │ 处理   │
└────────┘                └────────┘                  └────────┘

问题:
- Channel容量有限（如1000）
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

根据文档 `4.后端开发.md` 行435-470：

### 完整代码（带详细注释）

```go
package chat

// ============================================================
// 导入依赖包
// ============================================================
import (
    "encoding/json"
    
    // ★WebSocket库
    "github.com/gorilla/websocket"
    
    // ★项目内部依赖
    "kama-chat-server/internal/dto/request"
    "kama-chat-server/internal/service/kafka"
    "kama-chat-server/pkg/zlog"
)

// ============================================================
// MessageBack - 返回给前端的消息结构
// ============================================================
// ★Client.Write通过SendBack Channel发送这个结构
type MessageBack struct {
    Message []byte  // 序列化的消息（JSON）
    Uuid    string  // 消息UUID（用于消息回执）
}

// ============================================================
// Client - WebSocket客户端结构
// ============================================================
// ★每个WebSocket连接对应一个Client实例
type Client struct {
    Conn     *websocket.Conn      // WebSocket连接
    Uuid     string               // 用户唯一标识
    SendTo   chan []byte          // ★给Server端的Channel（Channel模式缓冲）
    SendBack chan *MessageBack    // 发送给前端的Channel
}

// ============================================================
// NewClient - 创建Client实例
// ============================================================
func NewClient(conn *websocket.Conn, uuid string) *Client {
    return &Client{
        Conn:     conn,
        Uuid:     uuid,
        // ★SendTo Channel: Client→Server的消息缓冲（Channel模式）
        SendTo:   make(chan []byte, constants.CHANNEL_SIZE),
        // ★SendBack Channel: Server→Client的消息缓冲
        // 超过100条未发送消息，会阻塞
        SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE),
    }
}

// ============================================================
// Read - 从WebSocket读取消息
// ============================================================
// ★在单独协程中运行
// 流程: WebSocket读取 → 根据配置选择Channel模式或Kafka模式
func (c *Client) Read() {
    defer func() {
        // ★连接关闭时，从Server注销
        Server.Logout <- c
        c.Conn.Close()
    }()
    
    for {
        // 1. 从WebSocket读取消息（阻塞）
        // ★c.Conn.ReadMessage: 读取WebSocket消息
        _, jsonMessage, err := c.Conn.ReadMessage()
        if err != nil {
            zlog.Error("WebSocket读取失败: " + err.Error())
            break
        }
        
        // 2. 解析消息获取SendId
        var message request.ChatMessageRequest
        json.Unmarshal(jsonMessage, &message)
        
        // ★安全检查：确保消息来自当前用户
        if message.SendId != c.Uuid {
            zlog.Warn("用户身份不匹配")
            continue
        }
        
        // 3. ★根据配置选择消息处理模式
        if messageMode == "channel" {
            // ★★★ Channel模式: 多级缓冲机制 ★★★
            
            // 步骤A: 如果Server.Transmit没满，先把SendTo中的消息转发
            for len(Server.Transmit) < constants.CHANNEL_SIZE && len(c.SendTo) > 0 {
                sendToMessage := <-c.SendTo
                Server.SendMessageToTransmit(sendToMessage)
            }
            
            // 步骤B: 如果Server没满，SendTo空了，直接给Server.Transmit
            if len(Server.Transmit) < constants.CHANNEL_SIZE {
                Server.SendMessageToTransmit(jsonMessage)
            } else if len(c.SendTo) < constants.CHANNEL_SIZE {
                // 步骤C: 如果Server满了，先塞到SendTo缓冲
                c.SendTo <- jsonMessage
            } else {
                // 步骤D: 都满了，发送失败提示
                c.Conn.WriteMessage(websocket.TextMessage, 
                    []byte("消息发送失败，请稍后重试"))
            }
        } else {
            // ★★★ Kafka模式: 直接写入Kafka ★★★
            err = kafka.WriteMessage(jsonMessage)
            if err != nil {
                zlog.Error("Kafka写入失败: " + err.Error())
                continue
            }
        }
    }
}

// ============================================================
// Write - 发送消息给前端
// ============================================================
// ★在单独协程中运行
// 流程: 从SendBack Channel读取 → 发送WebSocket
func (c *Client) Write() {
    defer func() {
        c.Conn.Close()
    }()
    
    for {
        select {
        // ★从SendBack Channel获取消息
        case messageBack, ok := <-c.SendBack:
            if !ok {
                // Channel关闭
                return
            }
            
            // ★发送WebSocket消息
            // ★c.Conn.WriteMessage: 发送WebSocket消息
            c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
        }
    }
}
```

---

## 四、Channel模式缓冲机制 ★重要

### 多级缓冲设计

根据源码 `client.go` 行62-79：

```
消息缓冲优先级：
1. Server.Transmit（最优先）
2. Client.SendTo（次优先）
3. 消息失败（最后）
```

### 缓冲流程图

```
WebSocket接收消息
       │
       ↓
┌──────────────────────────────────────┐
│ 步骤A: Server.Transmit未满?          │
│        ├─→ YES: 先清空Client.SendTo  │
│        │      (把缓冲的消息发送出去)  │
│        └─→ NO: 跳过                  │
└──────────────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────┐
│ 步骤B: Server.Transmit未满?          │
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
│        "消息发送失败，请稍后重试"      │
└──────────────────────────────────────┘
```

### Why这样设计？

**问题**: Channel容量有限，高峰期可能满

**解决**: 多级缓冲
- Client.SendTo作为第一级缓冲（每个用户独立）
- Server.Transmit作为第二级缓冲（全局共享）

**How to apply**:
- 小规模用户（<100）: Channel模式足够
- 大规模用户（>1000）: 使用Kafka模式

---

## 五、WebSocket Server结构

**文件位置:** `internal/service/chat/server.go`

根据文档 `4.后端开发.md` 行473-510：

### 完整代码（带详细注释）

```go
package chat

import (
    "encoding/json"
    "sync"
    
    "github.com/gorilla/websocket"
    
    "kama-chat-server/internal/dto/request"
    "kama-chat-server/internal/dao"
    "kama-chat-server/internal/model"
    "kama-chat-server/internal/service/kafka"
    "kama-chat-server/pkg/constants"
    "kama-chat-server/pkg/util/random"
    "kama-chat-server/pkg/zlog"
)

// ============================================================
// Server - WebSocket服务器结构
// ============================================================
// ★管理所有WebSocket连接
type Server struct {
    // ★Clients: 所有在线用户的map
    // Key: 用户Uuid, Value: Client实例
    // ★并发访问需要mutex保护
    Clients  map[string]*Client
    
    // ★mutex: 保护Clients map
    // 多协程同时访问Clients需要加锁
    mutex    *sync.Mutex
    
    // ★Transmit Channel: 消息转发通道（Channel模式核心）
    // 所有消息先进入Transmit，然后Server处理
    Transmit chan []byte
    
    // ★Login/Logout Channel: 用户登录/登出
    // 登录/登出频率低，可以用Channel
    Login    chan *Client
    Logout   chan *Client
}

// ============================================================
// 全局Server实例
// ============================================================
var Server *Server

// ============================================================
// init() - 初始化Server
// ============================================================
func init() {
    Server = &Server{
        Clients:  make(map[string]*Client),
        mutex:    &sync.Mutex{},
        // ★Transmit容量由constants.CHANNEL_SIZE决定（默认100）
        Transmit: make(chan []byte, constants.CHANNEL_SIZE),
        Login:    make(chan *Client, constants.CHANNEL_SIZE),
        Logout:   make(chan *Client, constants.CHANNEL_SIZE),
    }
}

// ============================================================
// Start - 启动Server
// ============================================================
// ★在main.go中调用
func (s *Server) Start() {
    defer func() {
        close(s.Transmit)
        close(s.Login)
        close(s.Logout)
    }()
    
    // ★启动Kafka消费协程（如果使用Kafka模式）
    if messageMode == "kafka" {
        go s.consumeKafka()
    }
    
    // 处理登录/登出/消息转发
    for {
        select {
        case client := <-s.Login:
            // ★用户登录：加入Clients map
            s.mutex.Lock()
            s.Clients[client.Uuid] = client
            s.mutex.Unlock()
            
            // 发送欢迎消息
            welcomeMsg := "欢迎来到kama聊天服务器"
            client.Conn.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))
            
        case client := <-s.Logout:
            // ★用户登出：从Clients map删除
            s.mutex.Lock()
            delete(s.Clients, client.Uuid)
            s.mutex.Unlock()
            
            // 关闭Client的Channel
            close(client.SendTo)
            close(client.SendBack)
            
        case data := <-s.Transmit:
            // ★★★ Channel模式: 从Transmit读取消息 ★★★
            // 处理消息（存储 + 转发）
            s.HandleMessage(data)
        }
    }
}

// ============================================================
// consumeKafka - 消费Kafka消息
// ============================================================
// ★单独协程运行，持续消费
func (s *Server) consumeKafka() {
    for {
        // 1. 从Kafka读取消息（阻塞）
        kafkaMessage, err := kafka.ReadMessage()
        if err != nil {
            continue
        }
        
        // 2. 处理消息（存储 + 转发）
        s.HandleMessage(kafkaMessage.Value)
    }
}

// ============================================================
// HandleMessage - 处理消息
// ============================================================
// ★核心函数：存储数据库 + 转发给接收者
func (s *Server) HandleMessage(data []byte) {
    // 1. 解析消息
    var message request.ChatMessageRequest
    json.Unmarshal(data, &message)
    
    // 2. ★存储到数据库
    dbMessage := model.Message{
        Uuid:        "M" + random.GetNowAndLenRandomString(11),
        SessionId:   message.SessionId,
        Type:        message.Type,
        Content:     message.Content,
        Url:         message.Url,
        SendId:      message.SendId,
        SendName:    message.SendName,
        SendAvatar:  message.SendAvatar,
        ReceiveId:   message.ReceiveId,
        FileType:    message.FileType,
        FileName:    message.FileName,
        FileSize:    message.FileSize,
        Status:      1,  // 已发送
        CreatedAt:   time.Now(),
    }
    
    dao.GormDB.Create(&dbMessage)
    
    // 3. ★转发消息（需要mutex保护）
    messageBack := &MessageBack{
        Message: data,
        Uuid:    dbMessage.Uuid,
    }
    
    s.mutex.Lock()
    
    // 判断是单聊还是群聊
    if len(message.ReceiveId) > 0 && message.ReceiveId[0] == 'G' {
        // ★群聊：转发给所有群成员
        s.forwardToGroup(message, messageBack)
    } else {
        // ★单聊：转发给接收者和发送者
        s.forwardToUser(message, messageBack)
    }
    
    s.mutex.Unlock()
}

// ============================================================
// forwardToUser - 单聊转发
// ============================================================
func (s *Server) forwardToUser(message request.ChatMessageRequest, messageBack *MessageBack) {
    // 1. ★转发给接收者（如果在线）
    if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
        receiveClient.SendBack <- messageBack
    }
    
    // 2. ★转发给发送者（消息回显）
    // ★用户发消息后，自己也需要收到（显示在聊天界面）
    if sendClient, ok := s.Clients[message.SendId]; ok {
        sendClient.SendBack <- messageBack
    }
}

// ============================================================
// forwardToGroup - 群聊转发
// ============================================================
func (s *Server) forwardToGroup(message request.ChatMessageRequest, messageBack *MessageBack) {
    // 1. 获取群成员列表
    var group model.GroupInfo
    dao.GormDB.Where("uuid = ?", message.ReceiveId).First(&group)
    
    var members []string
    json.Unmarshal(group.Members, &members)
    
    // 2. ★转发给所有群成员
    for _, member := range members {
        // 不转发给自己（已经在forwardToUser中处理）
        if member != message.SendId {
            if client, ok := s.Clients[member]; ok {
                client.SendBack <- messageBack
            }
        }
    }
    
    // 3. 转发给发送者（消息回显）
    if sendClient, ok := s.Clients[message.SendId]; ok {
        sendClient.SendBack <- messageBack
    }
}
```

---

## 六、WebSocket连接处理

### WsLogin函数

**文件位置:** `api/v1/ws_controller.go`

```go
package v1

import (
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    
    "kama-chat-server/internal/service/chat"
    "kama-chat-server/internal/https_server"
)

// ★WebSocket升级器
// ★gin.Context → websocket.Conn
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    // ★允许跨域
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// ============================================================
// WsLogin - WebSocket登录
// ============================================================
// ★必须是GET请求（WebSocket协议要求）
func WsLogin(c *gin.Context) {
    // 1. 获取用户Uuid（从查询参数）
    uuid := c.Query("uuid")
    if uuid == "" {
        https_server.JsonBack(c, "缺少uuid参数", -2, nil)
        return
    }
    
    // 2. ★升级HTTP连接为WebSocket
    // ★upgrader.Upgrade: HTTP → WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    
    // 3. 创建Client实例
    client := chat.NewClient(conn, uuid)
    
    // 4. ★注册到Server
    chat.Server.Login <- client
    
    // 5. ★启动Read和Write协程
    // ★两个协程分别处理读写
    go client.Read()
    go client.Write()
}
```

---

### WsLogout函数 - WebSocket登出

**文件位置:** `api/v1/ws_controller.go`

#### 1. 接口概述

| 项目 | 内容 |
|-----|------|
| 路由 | `/user/wsLogout` |
| HTTP方法 | POST |
| 功能描述 | WebSocket登出，关闭用户的WebSocket连接 |
| 使用场景 | 用户点击退出登录时，需要关闭WebSocket连接 |

#### 2. Request结构体定义

**文件位置:** `internal/dto/request/ws_logout_request.go`

```go
// ============================================================
// WsLogoutRequest - WebSocket登出请求
// ============================================================
// ★JSON请求体示例: {"owner_id": "U12345678901"}
type WsLogoutRequest struct {
    OwnerId string `json:"owner_id"` // 登出用户的uuid
}
```

#### 3. Respond结构体定义

此接口无data返回。

#### 4. Controller层完整代码

```go
// ============================================================
// WsLogout - WebSocket登出
// ============================================================
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

#### 5. Service层完整代码

**文件位置:** `internal/service/chat/client.go`

```go
// ============================================================
// ClientLogout - WebSocket客户端登出
// ============================================================
// ★当前端用户退出登录时，后端需要关闭对应的WebSocket连接
func ClientLogout(clientId string) (string, int) {
    // 1. 从Server的Clients map中获取客户端
    client := Server.Clients[clientId]
    
    // 2. 如果客户端存在，关闭连接
    if client != nil {
        // ★关闭WebSocket连接
        if err := client.Conn.Close(); err != nil {
            zlog.Error(err.Error())
            return constants.SYSTEM_ERROR, -1
        }
        // ★后续还需要：关闭channel、从Clients map中移除等操作
    }
    
    return "退出成功", 0
}
```

#### 6. WebSocket连接生命周期

```
前端登录 -> 后端创建Client -> 建立WebSocket连接 -> 消息通信
                                                    ↓
前端登出 -> 发送wsLogout请求 -> 后端关闭Conn -> 清理资源
```

---

## 七、sync.Mutex详解

### 为什么需要mutex？

```go
// ★Clients map被多协程访问：
// 1. consumeKafka协程: 转发消息时查询Clients
// 2. Login/Logout处理: 修改Clients
// 3. 多个Client.Read: 同时写入Kafka

// 不加锁会导致:
// - 数据竞争（race condition）
// - 程序崩溃或数据错乱

// 加锁保护:
s.mutex.Lock()           // 加锁
// ... 访问Clients
s.mutex.Unlock()         // 解锁
```

---

## 八、消息流转完整流程

### 单聊流程

```
1. ClientA发送消息
   ↓
2. ClientA.Read() 从WebSocket读取
   ↓
3. kafka.WriteMessage() 写入Kafka
   ↓
4. Server.consumeKafka() 从Kafka读取
   ↓
5. HandleMessage() 存储数据库
   ↓
6. forwardToUser() 转发
   ↓
7. ClientB.SendBack ← messageBack
   ↓
8. ClientB.Write() 发送WebSocket
   ↓
9. ClientB收到消息

同时:
   ClientA.SendBack ← messageBack（消息回显）
   ↓
   ClientA.Write() 发送WebSocket
   ↓
   ClientA收到自己的消息
```

### 群聊流程

```
1. ClientA发送消息到群组G
   ↓
2. 同上流程到HandleMessage()
   ↓
3. forwardToGroup() 查询群成员
   ↓
4. 遍历所有成员，转发给在线成员
   ↓
5. 所有在线群成员收到消息
```

---

## 九、离线消息处理

### 设计思路

```
用户在线:
- WebSocket实时推送

用户离线:
- 消息存储到数据库
- 用户登录时调用getMessageList获取
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

## 十、安装依赖

```bash
go get github.com/gorilla/websocket
```

---

## 十一、创建文件步骤

### 步骤1: 创建目录

```bash
mkdir internal/service/chat
```

### 步骤2: 创建文件

创建以下文件：
- `internal/service/chat/client.go`
- `internal/service/chat/server.go`
- `api/v1/ws_controller.go`

### 步骤3: 启动Server

在main.go中调用：

```go
func main() {
    // 启动WebSocket Server
    go chat.Server.Start()
    
    // 启动HTTP服务器
    https_server.RunServer()
}
```

---

## 十二、下一步

WebSocket实现完成后，继续学习：
- **20-聊天室管理.md** - 聊天室功能实现
- **21-分布式系统扩展.md** - 多Server部署和负载均衡