# React前端功能开发提示词

本文档包含三个功能模块的开发提示词，可直接复制给AI使用。

---

## 提示词一：管理员页面

```
请为KamaChat项目开发管理员页面（React + TypeScript + Ant Design 5.x）。

## 项目背景

这是一个仿微信聊天应用，后端是Go + Gin框架，前端使用React 18 + TypeScript + Zustand + Ant Design 5.x。
项目路径：react_web/src/

## 路由配置

在 App.tsx 中添加路由：
- /manager -> ManagerPage（需要管理员权限，检查 useAuthStore.userInfo.is_admin === 1）

路由守卫逻辑：非管理员用户重定向到 /chat

## 页面结构

整体布局：Header + Sidebar + Main Content

### Header区域
- 高度70px，背景色 rgb(252, 210, 210)，顶部圆角30px
- 左侧：管理员图标 + "Admin" 标题
- 右侧：一个"返回聊天"按钮，点击跳转 /chat

### 左侧边栏（200px宽）
使用 Ant Design 的 Menu 组件，包含两个子菜单：

子菜单1："用户管理"（用户图标）
- 菜单项1：启用/禁用用户
- 菜单项2：删除用户
- 菜单项3：设置管理员

子菜单2："群组管理"（群组图标）
- 菜单项1：启用/禁用群组
- 菜单项2：删除/解散群组

点击菜单项时，右侧主内容区显示对应的管理面板（同一时间只显示一个）。

### 右侧主内容区

根据选中的菜单项显示不同的管理面板。五个面板共享相同的交互模式：
- 使用 Ant Design 的 Table 组件，支持行选择（checkbox）
- 页面加载时调用API获取数据
- 底部有操作按钮，执行批量操作后刷新表格

## 五个管理面板详情

### 面板A：启用/禁用用户

数据加载：调用 `getUserInfoList(owner_id)` API（已存在于 react_web/src/api/user.ts）

表格列：
| 列名 | 字段 | 显示方式 |
|------|------|---------|
| UUID | uuid | 文本 |
| 昵称 | nickname | 文本 |
| 邮箱 | email | 文本 |
| 管理员 | is_admin | Tag：是(绿色)/否(默认) |
| 状态 | status | Tag：正常(绿色)/禁用(红色) |

底部两个按钮：
- "禁用"按钮：获取选中行的uuid列表，调用 `disableUsers({ uuid_list, is_admin: 0 })`，成功后刷新表格
- "启用"按钮：获取选中行的uuid列表，调用 `ableUsers({ uuid_list, is_admin: 0 })`，成功后刷新表格

### 面板B：删除用户

数据加载：同上 `getUserInfoList(owner_id)`

表格列：UUID、昵称、邮箱、管理员状态、用户状态

底部一个按钮：
- "删除"按钮：获取选中行的uuid列表，调用 `deleteUsers({ uuid_list, is_admin: 0 })`，成功后刷新表格
- 删除前弹出确认对话框

### 面板C：设置管理员

数据加载：同上 `getUserInfoList(owner_id)`

表格列：UUID、昵称、邮箱、管理员状态

底部两个按钮：
- "设为管理员"按钮：调用 `setAdmin({ uuid_list, is_admin: 1 })`
- "取消管理员"按钮：调用 `setAdmin({ uuid_list, is_admin: 0 })`

### 面板D：启用/禁用群组

数据加载：调用 `getGroupInfoList()` API（已存在于 react_web/src/api/group.ts）

表格列：
| 列名 | 字段 | 显示方式 |
|------|------|---------|
| UUID | uuid | 文本 |
| 群名称 | name | 文本 |
| 群主ID | owner_id | 文本 |
| 状态 | status | Tag：正常(绿色)/禁用(红色) |

底部两个按钮：
- "禁用"按钮：调用 `setGroupsStatus({ uuid_list, status: 1 })`
- "启用"按钮：调用 `setGroupsStatus({ uuid_list, status: 0 })`

### 面板E：删除/解散群组

数据加载：同上 `getGroupInfoList()`

表格列：UUID、群名称、群主ID、状态

底部一个按钮：
- "删除"按钮：调用 `deleteGroups(uuid_list)`，删除前确认

## 已有的API函数（直接import使用）

```typescript
// react_web/src/api/user.ts
export async function getUserInfoList(owner_id: string): Promise<ApiResponse<UserInfo[]>>
export async function ableUsers(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>>
export async function disableUsers(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>>
export async function deleteUsers(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>>
export async function setAdmin(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>>

// react_web/src/api/group.ts
export async function getGroupInfoList(): Promise<ApiResponse<GroupInfo[]>>
export async function deleteGroups(uuid_list: string[]): Promise<ApiResponse<null>>
export async function setGroupsStatus(data: { uuid_list: string[]; status: number }): Promise<ApiResponse<null>>
```

## 类型定义

```typescript
// react_web/src/types/user.ts
export interface UserInfo {
  uuid: string
  nickname: string
  email: string
  avatar: string
  status: number       // 0=正常, 1=禁用
  is_admin: number     // 0=普通, 1=管理员
}

// react_web/src/types/group.ts
export interface GroupInfo {
  group_id: string
  group_name: string
  group_avatar: string
  owner_id: string
  member_count: number
}
```

## 状态管理

使用 Zustand（已安装）。当前用户信息从 `useAuthStore` 获取：
```typescript
import { useAuthStore } from '../stores/useAuthStore'
const userInfo = useAuthStore(state => state.userInfo)
// userInfo.uuid 用于 owner_id 参数
```

## 提示消息

使用已有的 showToast 工具函数：
```typescript
import { showToast } from '../utils/toast'
showToast('操作成功', 'success')
showToast('操作失败', 'error')
```

## 样式要求

- 使用 Ant Design 组件（Table, Button, Menu, Tag, Modal, Layout）
- 整体风格与项目一致：主色调 #07C160（微信绿）
- 表格支持分页（如果数据量大）
- 按钮使用 Ant Design 的 danger 属性表示危险操作

## 文件结构

```
react_web/src/
  pages/
    ManagerPage.tsx          # 管理员主页面
  components/
    admin/
      DisableUserPanel.tsx   # 启用/禁用用户面板
      DeleteUserPanel.tsx    # 删除用户面板
      SetAdminPanel.tsx      # 设置管理员面板
      DisableGroupPanel.tsx  # 启用/禁用群组面板
      DeleteGroupPanel.tsx   # 删除群组面板
```

请实现以上所有功能，确保TypeScript类型正确，API调用正确，UI交互流畅。
```

---

## 提示词二：音视频通话页面

```
请为KamaChat项目实现音视频通话功能（React + TypeScript + WebRTC）。

## 项目背景

这是一个仿微信聊天应用，后端是Go + Gin框架，前端使用React 18 + TypeScript + Zustand。
音视频通话使用WebRTC实现，信令通过已有的WebSocket连接传输。

## 技术架构

### 信令传输
复用已有的WebSocket连接（react_web/src/services/websocket.ts），使用 type: 3（AV消息类型）。
信令数据放在 ChatMessage 的 av_data 字段中（JSON字符串）。

### 需要修改的类型定义

在 react_web/src/types/message.ts 中，ChatMessage 接口需要添加 av_data 字段：
```typescript
export interface ChatMessage {
  // ... 现有字段 ...
  av_data?: string  // 音视频信令数据（JSON字符串）
}
```

在 ChatMessageRequest 中也需要添加：
```typescript
export interface ChatMessageRequest {
  // ... 现有字段 ...
  av_data?: string
}
```

### 信令协议

所有AV消息的 type = 3，av_data 是JSON字符串，结构如下：

```typescript
interface AVData {
  messageId: string  // "PROXY" | "PEER_LEAVE" | "CURRENT_PEERS" | "PEER_JOIN"
  type?: string      // "start_call" | "receive_call" | "reject_call" | "sdp" | "candidate"
  messageData?: {
    sdp?: RTCSessionDescriptionInit
    candidate?: RTCIceCandidateInit
  }
}
```

| messageId | type | 方向 | 用途 |
|-----------|------|------|------|
| PROXY | start_call | 发起方→接收方 | 通知有来电 |
| PROXY | receive_call | 接收方→发起方 | 接听通话 |
| PROXY | reject_call | 接收方→发起方 | 拒绝通话 |
| PROXY | sdp | 双向 | 交换SDP offer/answer |
| PROXY | candidate | 双向 | 交换ICE候选者 |
| PEER_LEAVE | - | 任一方 | 挂断通话 |

### WebSocket消息格式

发送AV消息时，通过 wsService.send() 发送：
```typescript
const request = {
  session_id: activeSessionId,
  type: 3,  // MessageType.AV
  content: "",
  url: "",
  send_id: userInfo.uuid,
  send_name: userInfo.nickname,
  send_avatar: userInfo.avatar,
  receive_id: contactInfo.contact_id,
  file_size: "",
  file_name: "",
  file_type: "",
  av_data: JSON.stringify(avData)
}
wsService.send(request)
```

## UI设计

### 触发方式
在 ChatHeader.tsx 的电话按钮点击时，打开音视频通话弹窗。

### 通话弹窗（Modal）
使用 Ant Design 的 Modal 组件，全屏或大尺寸（800x600），包含：

**头部**：标题"语音/视频通话" + 关闭按钮

**主体**：两个video元素并排显示
- 左侧：本地视频（muted，330x320px）
- 右侧：远程视频（330x320px，未连接时显示占位）
- 通话状态文字（等待中/通话中/已断开）

**底部**：5个操作按钮
1. "发起通话"按钮 - startCall(true)
2. "接听通话"按钮 - startCall(false)
3. "拒绝通话"按钮 - rejectCall()
4. "挂断"按钮 - sendEndCall()
5. "退出聊天室"按钮 - closeAVModal()

## 通话流程

### 流程A：发起通话
1. 用户点击"发起通话"
2. 检查：如果已在通话中，提示"通话中"
3. 创建 RTCPeerConnection（ICE配置为空对象，使用默认STUN）
4. 调用 navigator.mediaDevices.getUserMedia({ video: true, audio: true }) 获取本地流
5. 将本地流绑定到本地video元素（muted=true）
6. 将本地流的所有track添加到RTCPeerConnection
7. 发送WebSocket消息：av_data = { messageId: "PROXY", type: "start_call" }
8. 等待对方接听

### 流程B：接听通话
1. 收到 start_call 信令 → 显示通知"收到来自XXX的通话请求"
2. 用户点击"接听通话"
3. 检查：如果没有收到start_call，提示"对方未发起通话"
4. 同流程A的步骤3-6（创建PeerConnection、获取本地流）
5. 发送：av_data = { messageId: "PROXY", type: "receive_call" }
6. 发起方收到 receive_call → 创建SDP offer
7. 双方交换SDP和ICE候选者
8. ontrack事件触发时，将远程流绑定到远程video元素

### 流程C：拒绝通话
1. 用户点击"拒绝通话"
2. 发送：av_data = { messageId: "PROXY", type: "reject_call" }
3. 发起方收到 reject_call → 清理资源，提示"对方已拒绝"

### 流程D：挂断
1. 用户点击"挂断"
2. 隐藏video元素
3. 停止本地流的所有track
4. 关闭RTCPeerConnection
5. 重置所有状态
6. 发送：av_data = { messageId: "PEER_LEAVE" }
7. 对方收到 PEER_LEAVE → 同样清理资源

### 流程E：退出聊天室
1. 检查：如果还在通话中（本地或远程流存在），提示"请先结束通话"
2. 否则关闭弹窗

## WebSocket消息处理

修改 react_web/src/hooks/useWebSocket.ts，添加AV消息处理：

```typescript
// 在 onmessage 回调中
if (message.type === 3) {
  // 处理AV信令
  const avData = JSON.parse(message.av_data)
  // 调用音视频处理逻辑
} else {
  // 处理普通消息（现有逻辑）
  addIncomingMessage(message, userInfo.uuid)
}
```

## 文件结构

```
react_web/src/
  components/
    chat/
      AVCallModal.tsx       # 音视频通话弹窗组件
  hooks/
    useWebRTC.ts            # WebRTC逻辑Hook（封装RTCPeerConnection、媒体流管理）
  services/
    websocket.ts            # 可能需要添加AV消息专用的发送方法
```

## useWebRTC Hook 应该封装

```typescript
function useWebRTC() {
  // 状态
  const [isModalVisible, setIsModalVisible] = useState(false)
  const [callStatus, setCallStatus] = useState<'idle' | 'calling' | 'connected' | 'rejected'>('idle')
  const [ableToStartCall, setAbleToStartCall] = useState(true)
  const [ableToReceiveCall, setAbleToReceiveCall] = useState(false)
  
  // Refs
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const localVideoRef = useRef<HTMLVideoElement>(null)
  const remoteVideoRef = useRef<HTMLVideoElement>(null)
  
  // 方法
  const startCall: (isInitiator: boolean) => Promise<void>
  const rejectCall: () => void
  const sendEndCall: () => void
  const receiveEndCall: () => void
  const handleOfferSdp: (sdp: RTCSessionDescriptionInit) => Promise<void>
  const handleAnswerSdp: (sdp: RTCSessionDescriptionInit) => Promise<void>
  const handleCandidate: (candidate: RTCIceCandidateInit) => Promise<void>
  const openModal: () => void
  const closeModal: () => void
  
  // 处理接收到的AV消息
  const handleAVMessage: (msg: ChatMessage) => void
  
  return {
    isModalVisible, callStatus,
    localVideoRef, remoteVideoRef,
    startCall, rejectCall, sendEndCall,
    openModal, closeModal, handleAVMessage
  }
}
```

## 注意事项

1. RTCPeerConnection 的 ICE 配置使用空对象 {}（默认STUN服务器）
2. 本地video元素必须设置 muted=true 防止回声
3. 挂断时要正确释放所有资源（stop tracks、close peer connection）
4. onicecandidate 事件中发送ICE候选者给对方
5. ontrack 事件中将远程流绑定到远程video元素
6. 处理浏览器兼容性（getUserMedia可能需要在HTTPS环境下使用）

请实现完整的音视频通话功能，确保信令流程正确，资源管理完善，用户体验流畅。
```

---

## 提示词三：查看信息弹窗

```
请为KamaChat项目实现联系人/群组信息查看弹窗（React + TypeScript + Ant Design 5.x）。

## 项目背景

这是一个仿微信聊天应用。聊天头部右侧有"更多"按钮（三个点），点击后弹出下拉菜单。
点击菜单中的"个人信息"或"群聊信息"时，需要弹出对应的信息弹窗。

## 已有的ChatHeader组件

文件路径：react_web/src/components/chat/ChatHeader.tsx

当前下拉菜单中，"个人信息"和"群聊信息"的onClick是占位的 showToast。
需要将它们替换为打开对应弹窗的逻辑。

## 需要实现的弹窗

### 弹窗A：个人信息弹窗

触发条件：联系人ID以"U"开头时，点击下拉菜单的"个人信息"

使用 Ant Design 的 Modal 组件，宽度500px。

**显示内容**（数据来自 contactInfo，已在ChatHeader中可用）：
| 字段 | 显示方式 |
|------|---------|
| 头像 | 100x100px 圆形图片，居中显示 |
| ID | contact_id |
| 昵称 | contact_name |
| 性别 | contact_gender：0=男，1=女（显示文字） |
| 手机 | contact_phone |
| 邮箱 | contact_email |
| 生日 | contact_birthday |
| 个性签名 | contact_signature（多行显示区域，高度70px） |

布局建议：使用 Ant Design 的 Descriptions 组件（vertical布局，带边框），或自定义布局。
头像居中放在顶部，下面用 Descriptions 展示其他信息。

### 弹窗B：群聊信息弹窗

触发条件：联系人ID以"G"开头时，点击下拉菜单的"群聊信息"

使用 Ant Design 的 Modal 组件，宽度500px。

**显示内容**：
| 字段 | 显示方式 |
|------|---------|
| 群头像 | 100x100px 圆形图片，居中显示 |
| 群ID | contact_id |
| 群名称 | contact_name |
| 成员数量 | contact_member_cnt |
| 群主ID | contact_owner_id |
| 加群方式 | contact_add_mode：0=直接加入，1=需审核（显示文字） |
| 群公告 | contact_notice（多行显示区域，高度70px） |

## ContactInfo 类型定义

```typescript
// react_web/src/types/user.ts
export interface ContactInfo {
  contact_id: string
  contact_name: string
  contact_avatar: string
  contact_phone: string
  contact_email: string
  contact_gender: number        // 0=男, 1=女
  contact_signature: string
  contact_birthday: string
  contact_notice: string
  contact_members: string       // JSON字符串
  contact_member_cnt: number
  contact_owner_id: string
  contact_add_mode: number      // 0=直接, 1=审核
}
```

## 需要修改的文件

### 1. 新建组件：react_web/src/components/chat/ContactInfoModal.tsx

```tsx
interface ContactInfoModalProps {
  visible: boolean
  onClose: () => void
  contactInfo: ContactInfo
  isGroup: boolean  // contact_id.startsWith('G')
}
```

根据 isGroup 显示不同的内容布局。

### 2. 修改：react_web/src/components/chat/ChatHeader.tsx

添加状态控制弹窗显示：
```tsx
const [infoModalVisible, setInfoModalVisible] = useState(false)
```

修改下拉菜单的"个人信息"和"群聊信息"项：
```tsx
// 原来：
{ key: 'info', label: '个人信息', onClick: () => showToast('个人信息功能开发中', 'info') }

// 改为：
{ key: 'info', label: '个人信息', onClick: () => setInfoModalVisible(true) }
```

在组件JSX底部添加弹窗：
```tsx
<ContactInfoModal
  visible={infoModalVisible}
  onClose={() => setInfoModalVisible(false)}
  contactInfo={contactInfo}
  isGroup={isGroup}
/>
```

## 样式要求

- 弹窗使用 Ant Design Modal，圆角边框
- 头像居中显示，圆形裁剪
- Descriptions 组件使用 bordered 属性
- 群公告和个性签名区域有滚动条（内容过长时）
- 整体风格简洁，与微信风格一致
- 关闭按钮使用右上角的X

## 文件结构

```
react_web/src/
  components/
    chat/
      ChatHeader.tsx          # 修改：添加弹窗状态和调用
      ContactInfoModal.tsx    # 新建：信息查看弹窗
```

请实现以上功能，确保类型正确，UI美观，交互流畅。
```
