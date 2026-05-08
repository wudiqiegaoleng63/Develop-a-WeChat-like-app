# 教学文档 22: WebSocket实时聊天详解（超详细版）

## 写在前面

本文档适合：**完全不懂WebSocket的小白**

阅读时间：**约20分钟**

目标：**看完后能理解WebSocket是怎么实现实时聊天的**

---

## 第一章：什么是实时聊天？

### 1.1 生活中的例子

想象你在微信上给朋友发消息：

```
你：在吗？
      ↓ 瞬间到达
朋友：在的，怎么了？
      ↓ 瞬间到达
你：今晚一起吃饭？
```

这就是**实时聊天**：消息发出去，对方立刻就能收到。

### 1.2 和传统网页的区别

| 传统网页 | 实时聊天 |
|---------|---------|
| 你刷新页面，才能看到新内容 | 不用刷新，新内容自己弹出来 |
| 像看电视（被动接收） | 像打电话（双向沟通） |
| 只有你请求，服务器才响应 | 服务器可以主动给你发消息 |

---

## 第二章：为什么需要WebSocket？

### 2.1 传统HTTP的痛点

假设你要做一个聊天功能，用传统HTTP：

```
场景：用户A给用户B发消息

方式1：轮询（每隔几秒问一次服务器）
┌─────────────────────────────────────────────────────────────┐
│ 用户B的浏览器每隔2秒问服务器：                                 │
│                                                             │
│ B: 有人给我发消息吗？                                         │
│ 服务器：没有                                                  │
│ B: 有人给我发消息吗？                                         │
│ 服务器：没有                                                  │
│ B: 有人给我发消息吗？                                         │
│ 服务器：没有                                                  │
│ B: 有人给我发消息吗？                                         │
│ 服务器：有！用户A给你发了"你好"                                │
└─────────────────────────────────────────────────────────────┘

问题：
1. 浪费资源：大部分请求都是"没有消息"
2. 不实时：可能要等2秒才能收到消息
3. 服务器压力大：1000个用户 = 每秒500次请求
```

### 2.2 WebSocket来了

WebSocket像一个**打电话**：

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  用户A                服务器                用户B            │
│    │                   │                    │              │
│    │──── 建立连接 ────→│←──── 建立连接 ────│              │
│    │                   │                    │              │
│    │     (连接保持，不用断开)                 │              │
│    │                   │                    │              │
│    │──── "你好" ──────→│                    │              │
│    │                   │──── "你好" ───────→│              │
│    │                   │                    │              │
│    │                   │←─── "在吗？" ─────│              │
│    │←─── "在吗？" ────│                    │              │
│    │                   │                    │              │
└─────────────────────────────────────────────────────────────┘

优点：
1. 建立一次连接，一直保持
2. 服务器可以主动发消息给客户端
3. 消息瞬间到达，真正实时
```

---

## 第三章：WebSocket连接过程

### 3.1 连接建立的三个步骤

```
第1步：客户端发起HTTP请求（带上Upgrade头）

客户端 → 服务器
GET /user/wsLogin?client_id=U123 HTTP/1.1
Host: 127.0.0.1:8000
Upgrade: websocket          ← 告诉服务器：我要升级成WebSocket
Connection: Upgrade
Sec-WebSocket-Key: xxxxx    ← 随机字符串，用于验证

---

第2步：服务器同意升级

服务器 → 客户端
HTTP/1.1 101 Switching Protocols  ← 101表示"协议切换成功"
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: yyyyy       ← 用Key计算出的验证值

---

第3步：连接建立成功！

现在双方可以互相发消息了，不需要再发HTTP请求头
```

### 3.2 代码实现（后端）

```go
// 1. 定义WebSocket升级器
var upgrader = websocket.Upgrader{
    ReadBufferSize:  2048,
    WriteBufferSize: 2048,
    CheckOrigin: func(r *http.Request) bool {
        return true  // 允许跨域
    },
}

// 2. 处理WebSocket连接请求
func WsLogin(c *gin.Context) {
    // 获取用户ID
    clientId := c.Query("client_id")

    // 把HTTP连接升级成WebSocket连接
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }

    // 创建客户端对象
    client := &Client{
        Conn:     conn,      // WebSocket连接
        Uuid:     clientId,  // 用户ID
        SendBack: make(chan *MessageBack), // 消息通道
    }

    // 注册到服务器
    ChatServer.Register <- client

    // 启动读写协程
    go client.Read()   // 读取客户端消息
    go client.Write()  // 发送消息给客户端
}
```

### 3.3 代码实现（前端）

```javascript
// 创建WebSocket连接
const wsUrl = "ws://127.0.0.1:8000/user/wsLogin?client_id=U123";
const socket = new WebSocket(wsUrl);

// 连接成功
socket.onopen = function() {
    console.log("连接成功！");
};

// 收到消息
socket.onmessage = function(message) {
    console.log("收到消息：", message.data);
};

// 连接关闭
socket.onclose = function() {
    console.log("连接关闭");
};

// 发送消息
socket.send(JSON.stringify({
    type: 1,
    content: "你好",
    receive_id: "U456"
}));
```

---

## 第四章：聊天室模型（核心！）

### 4.1 现实世界的聊天室

想象一个真实的聊天室：

```
┌─────────────────────────────────────────────────────────────┐
│                        聊天室                               │
│                                                             │
│   用户A（在线）  用户B（在线）  用户C（在线）                  │
│       │            │            │                          │
│       └────────────┼────────────┘                          │
│                    │                                       │
│              服务器（管理员）                                │
│                                                             │
│   规则：                                                     │
│   1. 有人进来，管理员记录名字                                │
│   2. 有人说话，管理员告诉所有人                               │
│   3. 有人离开，管理员划掉名字                                │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 代码中的聊天室

```go
// 服务器 = 聊天室管理员
type Server struct {
    Clients  map[string]*Client  // 在线用户名单（名字 → 用户）
    Login    chan *Client        // 有人进来（入口）
    Logout   chan *Client        // 有人离开（出口）
    Transmit chan []byte         // 消息传送带
}

// 客户端 = 聊天室里的一个人
type Client struct {
    Conn     *websocket.Conn    // 和服务器通话的电话线
    Uuid     string             // 名字
    SendBack chan *MessageBack  // 收消息的信箱
}
```

### 4.3 服务器的核心工作

```go
func (s *Server) Start() {
    for {
        select {
        // 情况1：有人进来
        case client := <-s.Login:
            s.Clients[client.Uuid] = client  // 记录名字
            client.Conn.WriteMessage("欢迎来到聊天室")

        // 情况2：有人离开
        case client := <-s.Logout:
            delete(s.Clients, client.Uuid)   // 划掉名字
            client.Conn.WriteMessage("再见")

        // 情况3：有人发消息
        case message := <-s.Transmit:
            // 解析消息，看看发给谁
            // 找到目标用户，放他信箱里
        }
    }
}
```

---

## 第五章：消息发送完整流程

### 5.1 私聊流程（用户A发给用户B）

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  用户A                  服务器                 用户B         │
│    │                     │                     │            │
│    │  ① 输入"你好"       │                     │            │
│    │                     │                     │            │
│    │  ② 点击发送         │                     │            │
│    │                     │                     │            │
│    │───③ WebSocket ─────→│                     │            │
│    │   {"type":1,        │                     │            │
│    │    "content":"你好", │                     │            │
│    │    "receive_id":"U_B"}                   │            │
│    │                     │                     │            │
│    │                     │ ④ 解析消息          │            │
│    │                     │    receive_id = U_B │            │
│    │                     │                     │            │
│    │                     │ ⑤ 查找用户B         │            │
│    │                     │    Clients["U_B"]   │            │
│    │                     │    找到了！在线！    │            │
│    │                     │                     │            │
│    │                     │───⑥ 发送给用户B ────→│            │
│    │                     │   {"content":"你好"}│            │
│    │                     │                     │            │
│    │←──⑦ 回显给用户A ────│                     │            │
│    │   (证明发送成功)     │                     │            │
│    │                     │                     │            │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 群聊流程（用户A发到群组G）

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  用户A               服务器                用户B  用户C       │
│    │                  │                     │      │        │
│    │─── 发送消息 ────→│                     │      │        │
│    │  receive_id="G1" │                     │      │        │
│    │                  │                     │      │        │
│    │                  │ 查询群成员          │      │        │
│    │                  │ [B, C, D, E]        │      │        │
│    │                  │                     │      │        │
│    │                  │ 检查谁在线          │      │        │
│    │                  │ B在线，C在线        │      │        │
│    │                  │ D离线，E离线        │      │        │
│    │                  │                     │      │        │
│    │←── 回显 ─────────│                     │      │        │
│    │                  │─── 发送给B ────────→│      │        │
│    │                  │─── 发送给C ───────────────→│        │
│    │                  │                     │      │        │
│    │                  │ D和E离线，消息存数据库       │        │
│    │                  │ 等他们上线再发       │      │        │
│    │                  │                     │      │        │
└─────────────────────────────────────────────────────────────┘
```

---

## 第六章：关键代码详解

### 6.1 Client.Read() - 读取客户端消息

```go
// 这个函数一直在运行，监听客户端发来的消息
func (c *Client) Read() {
    for {
        // ① 阻塞等待，直到收到消息
        _, jsonMessage, err := c.Conn.ReadMessage()
        if err != nil {
            return  // 连接断了，退出
        }

        // ② 把消息放到服务器的传送带上
        ChatServer.Transmit <- jsonMessage
    }
}
```

**理解要点**：
- `Read()`像一个**客服专员**，专门听用户说话
- `ReadMessage()`会**阻塞**（停在那里等），直到用户发消息
- 收到消息后，放到`Transmit`传送带上，让服务器处理

### 6.2 Client.Write() - 发送消息给客户端

```go
// 这个函数一直在运行，等待发送消息
func (c *Client) Write() {
    for {
        // ① 阻塞等待，直到SendBack里有消息
        messageBack := <-c.SendBack

        // ② 通过WebSocket发送给客户端
        c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)

        // ③ 更新数据库：消息状态改为"已发送"
        dao.GormDB.Model(&Message{}).
            Where("uuid=?", messageBack.Uuid).
            Update("status", "已发送")
    }
}
```

**理解要点**：
- `Write()`像一个**邮递员**，负责送信
- `SendBack`是用户的**信箱**，有人往里放消息，他就送出去
- 发送成功后，更新数据库状态

### 6.3 Server.Start() - 服务器主循环

```go
// 服务器的主循环，处理三件事：登录、登出、消息转发
func (s *Server) Start() {
    for {
        select {
        // ① 有人登录
        case client := <-s.Login:
            s.Clients[client.Uuid] = client  // 加入在线列表
            client.Conn.WriteMessage("欢迎")

        // ② 有人登出
        case client := <-s.Logout:
            delete(s.Clients, client.Uuid)  // 从在线列表删除

        // ③ 有消息要处理
        case data := <-s.Transmit:
            // 解析消息
            var msg ChatMessageRequest
            json.Unmarshal(data, &msg)

            // 存数据库
            message := Message{...}
            dao.GormDB.Create(&message)

            // 判断私聊还是群聊
            if msg.ReceiveId[0] == 'U' {
                // 私聊：发给一个人
                if receiver, ok := s.Clients[msg.ReceiveId]; ok {
                    receiver.SendBack <- messageBack
                }
            } else {
                // 群聊：发给群成员
                // 查询群成员列表，遍历发送
            }
        }
    }
}
```

**理解要点**：
- `select`像**前台**，同时监听三个通道
- 哪个通道有东西，就处理哪个
- 这是Go语言的特性，可以同时等待多个事件

---

## 第七章：消息结构详解

### 7.1 客户端发送的消息格式

```json
{
    "session_id": "S20260428001",
    "type": 1,
    "content": "你好，今晚一起吃饭吗？",
    "url": "",
    "send_id": "U20260428001",
    "send_name": "张三",
    "send_avatar": "/static/avatar.png",
    "receive_id": "U20260428002",
    "file_size": "",
    "file_type": "",
    "file_name": ""
}
```

| 字段 | 说明 | 示例 |
|------|------|------|
| session_id | 会话ID | S20260428001 |
| type | 消息类型（1文本 2文件 3音视频） | 1 |
| content | 消息内容 | 你好 |
| send_id | 发送者ID | U开头=用户，G开头=群组 |
| receive_id | 接收者ID | U开头=私聊，G开头=群聊 |

### 7.2 服务器返回的消息格式

```json
{
    "send_id": "U20260428001",
    "send_name": "张三",
    "send_avatar": "/static/avatar.png",
    "receive_id": "U20260428002",
    "type": 1,
    "content": "你好，今晚一起吃饭吗？",
    "url": "",
    "file_size": "0B",
    "file_name": "",
    "file_type": "",
    "created_at": "2026-05-07 20:30:00"
}
```

### 7.3 消息类型

| type值 | 类型 | 说明 |
|--------|------|------|
| 1 | 文本 | 普通文字消息 |
| 2 | 文件 | 图片、文档等 |
| 3 | 音视频 | 通话信令 |

---

## 第八章：完整时序图

### 8.1 用户登录到收消息的完整流程

```
用户A                   前端                    后端                   数据库
  │                      │                      │                      │
  │── 输入账号密码 ──────→│                      │                      │
  │                      │── POST /login ──────→│                      │
  │                      │                      │── 查询用户 ─────────→│
  │                      │                      │←─ 返回用户信息 ───────│
  │                      │←─ 返回token ─────────│                      │
  │                      │                      │                      │
  │                      │── WebSocket连接 ────→│                      │
  │                      │  ws://.../wsLogin    │                      │
  │                      │                      │── 创建Client ────────│
  │                      │                      │── 加入Clients map ───│
  │                      │←─ 连接成功 ──────────│                      │
  │                      │                      │                      │
  │                      │                      │    (连接保持中...)    │
  │                      │                      │                      │
  │                      │←─ 收到消息 ──────────│                      │
  │                      │  (有人给你发消息)     │                      │
  │                      │                      │                      │
  │←─ 界面显示消息 ───────│                      │                      │
  │                      │                      │                      │
```

### 8.2 发送消息的完整流程

```
用户A                   前端                    后端                   数据库
  │                      │                      │                      │
  │── 输入"你好" ────────→│                      │                      │
  │── 点击发送 ──────────→│                      │                      │
  │                      │                      │                      │
  │                      │── WebSocket.send() ─→│                      │
  │                      │  {"content":"你好"}   │                      │
  │                      │                      │                      │
  │                      │                      │── Read()收到消息 ────│
  │                      │                      │── 放入Transmit ──────│
  │                      │                      │                      │
  │                      │                      │── Start()处理 ───────│
  │                      │                      │  1. 解析消息         │
  │                      │                      │  2. 存数据库 ────────→│
  │                      │                      │  3. 查找接收者        │
  │                      │                      │  4. 发送给接收者      │
  │                      │                      │                      │
  │                      │←─ 回显消息 ──────────│                      │
  │←─ 显示已发送 ─────────│                      │                      │
  │                      │                      │                      │
```

---

## 第九章：常见问题

### 9.1 为什么消息发不出去？

**检查清单**：

```
1. WebSocket连接建立了吗？
   - 打开浏览器控制台（F12）
   - 看 Network → WS 标签
   - 如果没有连接，说明连接失败

2. 路由对了吗？
   - 前端请求：ws://127.0.0.1:8000/user/wsLogin
   - 后端路由：GET /user/wsLogin
   - 必须完全一致！

3. 用户在线吗？
   - 后端：ChatServer.Clients["U123"]
   - 如果是nil，说明用户没连上

4. 看后端日志
   - 有没有报错？
   - Read()有没有收到消息？
   - Start()有没有处理？
```

### 9.2 为什么收不到消息？

**可能原因**：

```
1. 发送者的消息格式不对
   - receive_id写错了吗？
   - JSON格式正确吗？

2. 接收者不在Clients里
   - 没登录？
   - WebSocket断了？

3. SendBack通道满了
   - Write()协程挂了？
   - 网络断了？
```

### 9.3 如何调试WebSocket？

**浏览器端**：

```javascript
// 1. 打开控制台
// 2. 查看WebSocket对象
console.log(store.state.socket);

// 3. 手动发送消息测试
store.state.socket.send(JSON.stringify({
    type: 1,
    content: "测试消息",
    receive_id: "U123"
}));

// 4. 查看连接状态
console.log(store.state.socket.readyState);
// 0 = CONNECTING
// 1 = OPEN
// 2 = CLOSING
// 3 = CLOSED
```

**服务端**：

```go
// 在关键位置加日志
func (c *Client) Read() {
    zlog.Info("Read协程启动，用户：" + c.Uuid)
    for {
        _, jsonMessage, err := c.Conn.ReadMessage()
        if err != nil {
            zlog.Error("读取失败：" + err.Error())
            return
        }
        zlog.Info("收到消息：" + string(jsonMessage))
        ChatServer.Transmit <- jsonMessage
    }
}
```

---

## 第十章：总结

### 10.1 WebSocket的本质

```
WebSocket = 一个持久的双向通道

HTTP：
  客户端 → 服务器（一次性）
  问一次，答一次

WebSocket：
  客户端 ←→ 服务器（持续）
  双方随时可以发消息
```

### 10.2 聊天系统的核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│                     聊天系统架构                             │
│                                                             │
│  ┌─────────────┐    WebSocket    ┌─────────────────────┐   │
│  │   前端      │ ←─────────────→ │   Client            │   │
│  │             │                 │   - Read() 读消息   │   │
│  │  socket.send│                 │   - Write() 发消息  │   │
│  │  socket.onmsg│                 │   - Uuid 用户ID    │   │
│  └─────────────┘                 └──────────┬──────────┘   │
│                                             │              │
│                                             ↓              │
│                                   ┌─────────────────────┐  │
│                                   │   Server            │  │
│                                   │   - Clients 在线表  │  │
│                                   │   - Login 登录通道  │  │
│                                   │   - Logout 登出通道 │  │
│                                   │   - Transmit 消息通道│  │
│                                   └──────────┬──────────┘  │
│                                             │              │
│                                             ↓              │
│                                   ┌─────────────────────┐  │
│                                   │   数据库            │  │
│                                   │   - Message表       │  │
│                                   │   - 存历史消息      │  │
│                                   └─────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 10.3 核心流程总结

| 步骤 | 发生了什么 | 代码位置 |
|------|-----------|---------|
| 1. 登录 | 用户登录，创建WebSocket连接 | WsLogin() |
| 2. 注册 | Client加入Server的Clients列表 | Server.Start() case Login |
| 3. 发消息 | 前端socket.send() | Client.Read() |
| 4. 处理 | Server从Transmit读取，存数据库，转发 | Server.Start() case Transmit |
| 5. 收消息 | Server写入Client.SendBack | Client.Write() |
| 6. 显示 | 前端socket.onmessage触发 | 前端回调函数 |

---

## 第十一章：动手练习

### 练习1：查看WebSocket连接

1. 打开浏览器，按F12打开开发者工具
2. 切换到 Network 标签
3. 筛选 WS（WebSocket）
4. 登录系统，观察WebSocket连接
5. 点击连接，查看 Messages 标签，看消息内容

### 练习2：发送测试消息

```javascript
// 在浏览器控制台执行
const testMsg = {
    session_id: "S001",
    type: 1,
    content: "这是一条测试消息",
    send_id: "你的用户ID",
    send_name: "你的用户名",
    send_avatar: "",
    receive_id: "对方用户ID"
};
store.state.socket.send(JSON.stringify(testMsg));
```

### 练习3：添加日志观察

在后端代码中添加日志，观察消息流转：

```go
// 在Server.Start()的case Transmit里加
zlog.Info("收到消息：" + string(data))
zlog.Info("发送者：" + chatMessageReq.SendId)
zlog.Info("接收者：" + chatMessageReq.ReceiveId)

// 在Client.Write()里加
zlog.Info("发送消息给用户：" + c.Uuid + "，内容：" + string(messageBack.Message))
```

---

## 附录：名词解释

| 名词 | 解释 |
|------|------|
| WebSocket | 一种网络协议，支持双向通信 |
| Channel | Go语言的通道，用于协程间通信 |
| Goroutine | Go语言的轻量级线程 |
| 阻塞 | 程序停在那里等待，直到有数据 |
| 协程 | 轻量级线程，Go语言特色 |
| 全双工 | 双方可以同时发送和接收数据 |
| 心跳 | 定期发送的小消息，用于检测连接是否活着 |

---

**恭喜你看完了！现在你应该理解WebSocket实时聊天是怎么工作的了！**
