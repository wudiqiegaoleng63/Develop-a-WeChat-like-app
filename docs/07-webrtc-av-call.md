# GoChat 音视频通话与 WebRTC 设计 — 面试详解

## 一、WebRTC 是什么

WebRTC（Web Real-Time Communication）是浏览器内置的**点对点实时通信技术**，可以让两个浏览器直接传输音视频数据，不需要经过服务器中转。

### 1.1 普通HTTP通信 vs WebRTC

```
普通HTTP通信（所有数据经过服务器）：
  浏览器A ──请求──► 服务器 ──响应──► 浏览器B
                    ↑
              服务器是中转站
              延迟高、服务器压力大

WebRTC通信（浏览器直接连接）：
  浏览器A ◄════════════════════► 浏览器B
            P2P 直连
            延迟低、服务器压力小
            但需要一个"信令服务器"帮忙建立连接
```

### 1.2 WebRTC 的三个核心概念

| 概念 | 作用 | 类比 |
|------|------|------|
| **SDP**（Session Description Protocol） | 描述自己的媒体能力（编码格式、分辨率等） | 交换名片，告诉对方"我支持什么" |
| **ICE Candidate** | 描述自己的网络地址（公网IP、局域网IP等） | 告诉对方"我的地址是什么" |
| **RTCPeerConnection** | 管理音视频传输的连接对象 | 通话管道本身 |

### 1.3 WebRTC 建立连接的流程

```
用户A                              信令服务器（WebSocket）                    用户B
  │                                     │                                     │
  │  1. 创建 RTCPeerConnection          │                                     │
  │  2. 获取本地音视频流                   │                                     │
  │  3. createOffer() → 生成 SDP Offer   │                                     │
  │                                     │                                     │
  │  4. 发送 SDP Offer ─────────────────►│ ──── 通过WebSocket转发 ────►        │
  │                                     │                                     │
  │                                     │               5. 收到 Offer           │
  │                                     │               6. 创建 RTCPeerConnection│
  │                                     │               7. setRemoteDescription  │
  │                                     │               8. createAnswer()        │
  │                                     │               → 生成 SDP Answer       │
  │                                     │                                     │
  │         ◄─── 通过WebSocket转发 ◄────│ ◄──── 发送 SDP Answer ─────────────│
  │  9. setRemoteDescription            │                                     │
  │                                     │                                     │
  │  10. 交换 ICE Candidate ───────────►│ ──────────────────────────────────►│
  │  ◄─────────────────────────────────│ ◄─── 交换 ICE Candidate ───────────│
  │                                     │                                     │
  │  11. P2P 连接建立，音视频直传 ◄══════│══════════════════════════════════►│
  │                                     │                                     │
  │           之后的数据不经过服务器         │                                     │
```

> **Q: 为什么需要信令服务器？浏览器不是直连吗？**
> A: 两个浏览器在建立直连之前，必须先交换 SDP 和 ICE Candidate，但它们还不知道对方的地址，没法直接通信。所以需要一个双方都能访问的中间人（信令服务器）来转交这些信息。**信令只在建立连接时用，连接建立后数据是 P2P 直传，不再经过服务器。** 本项目用 WebSocket 做信令服务器。

> **Q: 什么是 ICE Candidate？为什么有多个？**
> A: 浏览器会尝试多种方式连接对方：局域网 IP、公网 IP、TURN 中继服务器等，每种方式就是一个 ICE Candidate。浏览器会逐个尝试，选第一个能通的建立连接。这个过程叫 ICE（Interactive Connectivity Establishment）。

## 二、本项目音视频通话架构

```
┌─── 用户A浏览器 ────────────────────────────────────────────┐
│                                                             │
│  AVCallModal.tsx                  useAVStore.ts             │
│  ┌─────────────────┐           ┌──────────────────┐        │
│  │ RTCPeerConnection│           │ callStatus       │        │
│  │ localVideo       │           │ ableToStartCall │        │
│  │ remoteVideo      │◄────────►│ ableToReceive    │        │
│  │                  │           │ remoteSdp        │        │
│  │ getUserMedia()   │           │ handleAVMessage  │        │
│  │ createOffer()    │           │ sendAVMessage    │        │
│  │ createAnswer()   │           └──────────────────┘        │
│  └─────────────────┘                    │                   │
│                                          │                   │
│  wsService.send()  wsService.onMessage() │                   │
└──────────┬──────────────────────────────┬───────────────────┘
           │ WebSocket                     │
           ▼                               │
┌──────────────────────────────────────────┴──────────────────┐
│                      Go 后端                                 │
│                                                              │
│  Client(A).Read() ──► Transmit ──► Server.Start()           │
│                                              │               │
│                                    type == AudioOrVideo      │
│                                    ReceiveId[0] == 'U'       │
│                                              │               │
│                              ┌─── 存MySQL(仅状态变更) ──┐    │
│                              │  start_call / receive_call│    │
│                              │  reject_call              │    │
│                              └──────────────────────────┘    │
│                                              │               │
│                                    Client(B).SendBack ◄─────┘
│                                              │
│                                              ▼
│                                       Client(B).Write()
│                                              │
└──────────────────────────────────────────────┤
                                               ▼
┌──────────────────────────────────────────────────────────────┐
│                      用户B浏览器                               │
│                                                              │
│  useAVStore.handleAVMessage() → 解析 av_data → 更新状态       │
│  AVCallModal 监听状态变化 → 执行对应 WebRTC 操作               │
│                                                              │
│  音视频数据流（P2P直连，不经过服务器）：                         │
│  用户A ◄══ RTCPeerConnection ══► 用户B                        │
└──────────────────────────────────────────────────────────────┘
```

## 三、通话状态机

```
                    ┌─────────────────────────────────────┐
                    │              idle                    │
                    │           (初始状态)                  │
                    └──────┬──────────────┬────────────────┘
                           │              │
                    发起通话│              │收到start_call
                           ▼              ▼
                    ┌──────────┐   ┌───────────────────┐
                    │ calling  │   │ ableToReceive=true │
                    │(等待接听) │   │ ableToStart=false │
                    └──┬───┬───┘   └──┬─────────┬──────┘
                       │   │            │         │
              对方接听│   │对方拒绝     接听│       │拒绝
                       │   │            │         │
                       │   ▼            ▼         │
                       │ rejected    connected     │
                       │ (对方已拒绝) (通话中)      │
                       │                │         │
                       │         挂断/对方挂断│     │
                       │                │         │
                       ▼                ▼         ▼
                              resetAllState → idle
```

## 四、四种通话操作的详细流程

### 4.1 发起通话（start_call）

```
用户A点击"发起"按钮
    │
    ├── 1. createRtcPeerConnection()  → 创建 RTCPeerConnection
    ├── 2. getLocalMediaStream()       → 获取本地摄像头+麦克风
    ├── 3. attachTracks(stream)        → 把本地音视频轨道加入连接
    ├── 4. sendAVMessage({             → 通过WebSocket发送信令
    │       messageId: 'PROXY',
    │       type: 'start_call'
    │     })
    └── 5. setCallStatus('calling')   → 状态变为"等待接听"

后端处理：
    ├── 存MySQL（type=3, av_data={"messageId":"PROXY","type":"start_call"}）
    └── 转发给用户B（不回显给A，否则出现两个start_call）

用户B收到：
    ├── handleAVMessage() → ableToReceive=true, ableToStart=false
    └── showToast("收到来自A的通话请求")
```

### 4.2 接听通话（receive_call）

```
用户B点击"接听"按钮
    │
    ├── 1. createRtcPeerConnection()
    ├── 2. getLocalMediaStream()
    ├── 3. attachTracks(stream)
    └── 4. sendAVMessage({
            messageId: 'PROXY',
            type: 'receive_call'
          })

用户A收到receive_call：
    ├── handleAVMessage() → callStatus = 'create_offer'
    └── AVCallModal监听到状态变化 → createOffer()
```

### 4.3 SDP 和 ICE 交换（核心）

```
A.createOffer()
    │
    ├── 生成 SDP Offer（A的媒体能力描述）
    ├── setLocalDescription(offer)   → 设为自己的本地描述
    └── sendAVMessage({
            messageId: 'PROXY',
            type: 'sdp',
            messageData: { sdp: offer }
          })                         → 通过WebSocket发给B

B收到Offer SDP：
    ├── setRemoteDescription(offer) → 设为远程描述
    ├── createAnswer()              → 生成 SDP Answer
    ├── setLocalDescription(answer)  → 设为自己的本地描述
    └── sendAVMessage({
            messageId: 'PROXY',
            type: 'sdp',
            messageData: { sdp: answer }
          })                         → 通过WebSocket发给A

A收到Answer SDP：
    └── setRemoteDescription(answer) → 设为远程描述

双方同时：
    onicecandidate事件触发 → 发现新的ICE Candidate
    └── sendAVMessage({
            messageId: 'PROXY',
            type: 'candidate',
            messageData: { candidate: event.candidate }
          })                         → 通过WebSocket发给对方

    收到对方的ICE Candidate：
    └── addIceCandidate(candidate)   → 加入候选地址

ICE协商完成 → P2P连接建立 → 音视频数据直传
```

> **Q: 为什么 SDP 要分 Offer 和 Answer？**
> A: Offer 是发起方提出的媒体方案（"我支持H.264视频、Opus音频"），Answer 是接收方的回应（"我同意用H.264和Opus"）。双方必须协商出共同支持的编码格式，才能建立连接。

### 4.4 拒绝通话（reject_call）

```
用户B点击"拒绝"按钮
    │
    ├── sendAVMessage({
    │       messageId: 'PROXY',
    │       type: 'reject_call'
    │     })
    └── resetAllState()

用户A收到reject_call：
    ├── callStatus = 'rejected'
    ├── showToast("对方拒绝通话")
    └── cleanup() → 关闭连接、释放摄像头
```

### 4.5 挂断通话（hangup）

```
用户A点击"挂断"按钮
    │
    ├── sendAVMessage({
    │       messageId: 'PEER_LEAVE'
    │     })
    ├── cleanup()
    │   ├── localStream.getTracks().forEach(t => t.stop())  → 关闭摄像头麦克风
    │   ├── peerConnection.close()                           → 关闭WebRTC连接
    │   └── 清空视频元素
    └── resetAllState()

用户B收到PEER_LEAVE：
    ├── callStatus = 'peer_hangup'
    ├── showToast("对方已挂断")
    └── cleanup() + resetAllState()
```

## 五、信令透传 — 后端做了什么

后端对音视频通话消息的处理和文本消息不同：

| 维度 | 文本消息 | 音视频消息 |
|------|---------|-----------|
| **存MySQL** | 每条都存 | 仅存状态变更（start_call/receive_call/reject_call） |
| **发送者回显** | 回显 | **不回显**（否则出现两个start_call） |
| **数据处理** | 反序列化→构建响应 | 透传，不解析av_data内容 |

```go
// Server.Start() 中音视频消息处理
if chatMessageReq.Type == message_type_enum.AudioOrVideo {
    var avData request.AVData
    json.Unmarshal([]byte(chatMessageReq.AVdata), &avData)

    // 只有状态变更才存库
    if avData.MessageId == "PROXY" &&
       (avData.Type == "start_call" || avData.Type == "receive_call" || avData.Type == "reject_call") {
        dao.GormDB.Create(&message)
    }
    // SDP、ICE Candidate 等临时信令不存库

    // 只转发给接收者，不给发送者回显
    if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
        receiveClient.SendBack <- messageBack
    }
    // 注意：没有 sendClient.SendBack ← 不回显！
}
```

> **Q: 为什么 SDP 和 ICE Candidate 不存库？**
> A: 这些是临时信令，只在建立连接时有用，连接建立后就没意义了。存了浪费数据库空间。只有 start_call、receive_call、reject_call 是业务事件（"谁在什么时候给谁打过电话"），需要持久化。

> **Q: 为什么不给发送者回显？**
> A: 文本消息需要回显是因为前后端的数据结构不同，前端消息列表存的是响应结构。但音视频消息的发送者不需要再收到自己发出的 start_call——前端点击按钮时已经本地更新了状态，如果后端再回显，就会触发两次状态变更，出现两个通话窗口。

## 六、前端状态管理（useAVStore）

### 6.1 状态定义

```typescript
interface AVState {
    callStatus: CallStatus      // 通话状态：idle/calling/connected/...
    ableToReceiveOrReject: boolean  // 能否接听/拒绝（收到start_call时=true）
    ableToStartCall: boolean    // 能否发起通话（默认true，收到start_call时=false）
    remoteSdp: ... | null      // 远程SDP或ICE Candidate
}
```

### 6.2 handleAVMessage — 接收信令

```typescript
handleAVMessage: (avDataStr, sendName) => {
    const avData = JSON.parse(avDataStr)

    if (avData.messageId === 'PROXY') {
        if (avData.type === 'start_call') {
            // 收到来电 → 允许接听/拒绝，禁止发起
            set({ ableToReceiveOrReject: true, ableToStartCall: false })
            showToast(`收到来自${sendName}的通话请求`)
        } else if (avData.type === 'receive_call') {
            // 对方接听了 → 发起SDP协商
            set({ callStatus: 'create_offer' })
        } else if (avData.type === 'sdp') {
            if (sdp.type === 'offer')  set({ callStatus: 'handle_offer_sdp', remoteSdp: sdp })
            if (sdp.type === 'answer') set({ callStatus: 'set_answer_sdp', remoteSdp: sdp })
        } else if (avData.type === 'candidate') {
            set({ callStatus: 'add_ice_candidate', remoteSdp: candidate })
        }
    } else if (avData.messageId === 'PEER_LEAVE') {
        set({ callStatus: 'peer_hangup' })
    }
}
```

### 6.3 sendAVMessage — 发送信令

```typescript
sendAVMessage: (sessionId, userInfo, contactId, avPayload) => {
    const request: ChatMessageRequest = {
        session_id: sessionId,
        type: 3,                          // AudioOrVideo
        av_data: JSON.stringify(avPayload), // 信令数据
        // ... 其他字段
    }
    wsService.send(request)               // 通过WebSocket发送
}
```

### 6.4 AVCallModal — 状态驱动UI

```typescript
// 监听 callStatus 变化，执行对应 WebRTC 操作
useEffect(() => {
    if (callStatus === 'create_offer') {
        createOffer()
        setCallStatus('calling')
    } else if (callStatus === 'handle_offer_sdp' && remoteSdp) {
        peerConnection.setRemoteDescription(remoteSdp)
            .then(() => createAnswer())
    } else if (callStatus === 'set_answer_sdp' && remoteSdp) {
        peerConnection.setRemoteDescription(remoteSdp)
    } else if (callStatus === 'add_ice_candidate' && remoteSdp) {
        peerConnection.addIceCandidate(remoteSdp)
    } else if (callStatus === 'rejected') {
        showToast('对方拒绝通话')
        cleanup()
    } else if (callStatus === 'peer_hangup') {
        showToast('对方已挂断')
        cleanup()
    }
}, [callStatus])
```

## 七、完整通话时序图

```
用户A                           WebSocket服务器                    用户B
  │                                  │                              │
  │ ──── start_call ────────────────►│ ──── 透传 ────────────────►│ 收到来电提示
  │                                  │                              │
  │                                  │                  ◄── receive_call ──│ 点击接听
  │ ◄── receive_call ───────────────│                              │
  │                                  │                              │
  │ createOffer()                    │                              │
  │ ──── SDP Offer ────────────────►│ ──── 透传 ────────────────►│
  │                                  │                              │
  │                                  │      setRemoteDescription    │
  │                                  │      createAnswer()          │
  │ ◄── SDP Answer ────────────────│ ◄── SDP Answer ──────────────│
  │                                  │                              │
  │ setRemoteDescription             │                              │
  │                                  │                              │
  │ ──── ICE Candidate ────────────►│ ──── 透传 ────────────────►│
  │ ◄── ICE Candidate ─────────────│ ◄── ICE Candidate ───────────│
  │                                  │                              │
  │          ◄═══ P2P连接建立，音视频直传 ═══►                      │
  │                                  │                              │
  │          （之后数据不经过服务器）    │                              │
  │                                  │                              │
  │ ──── PEER_LEAVE ───────────────►│ ──── 透传 ────────────────►│
  │    cleanup()                     │                     cleanup()│
```

## 八、面试速记口诀

1. **WebRTC**：浏览器内置 P2P 实时通信，音视频数据浏览器直传，不经过服务器
2. **信令服务器**：本项目用 WebSocket 做信令，只转交 SDP 和 ICE Candidate，不参与实际音视频传输
3. **建立连接**：交换 SDP（Offer/Answer）→ 交换 ICE Candidate → P2P 连接建立
4. **后端透传**：后端不解析 av_data 内容，只做转发；仅 start_call/receive_call/reject_call 存库
5. **不回显**：音视频消息不给发送者回显，前端点击按钮时已本地更新状态
6. **状态驱动**：useAVStore 管理 callStatus，AVCallModal 监听状态变化执行对应 WebRTC 操作
7. **资源释放**：挂断时关闭 RTCPeerConnection + 停止摄像头/麦克风 track + 清空视频元素
