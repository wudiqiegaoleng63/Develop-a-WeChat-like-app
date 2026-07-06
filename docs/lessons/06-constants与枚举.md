---
name: 06-constants与枚举
description: 常量定义和枚举类型详解
type: project
---

# 教学文档 06: 常量定义与枚举类型

### 常量枚举是基础设施层

常量和枚举贯穿整个项目，在所有层都可以使用。按开发顺序：

```
Model → DAO → Service → Controller → 路由注册

常量枚举可在任何阶段引入，主要用于：
- 错误提示常量（SYSTEM_ERROR）
- 状态枚举（用户状态、联系人状态等）
- 配置常量（通道大小、超时时间等）
```

本章节介绍项目中使用的常量(constants)和枚举类型(enum)，它们贯穿整个项目的各个模块。

---

## 一、常量定义 (constants.go)

**源码位置**: `pkg/constants/constants.go`

```go
package constants

const (
    CHANNEL_SIZE  = 100            // 通道大小
    SYSTEM_ERROR  = "系统错误，请联系工作人员" // 系统错误
    FILE_MAX_SIZE = 50000          // 文件最大大小
    REDIS_TIMEOUT = 1              // redis timeout
)
```

### 常量详解

| 常量名 | 值 | 用途 |
|--------|-----|------|
| `CHANNEL_SIZE` | 100 | WebSocket通道容量，控制消息缓冲 |
| `SYSTEM_ERROR` | "系统错误，请联系工作人员" | 统一系统错误提示，不暴露内部细节 |
| `FILE_MAX_SIZE` | 50000 | 文件上传大小限制(字节) |
| `REDIS_TIMEOUT` | 1 | Redis操作超时时间(秒) |

### 使用场景

**SYSTEM_ERROR 使用示例**:
```go
// Controller层统一错误返回
func JsonBack(c *gin.Context, message string, ret int, data interface{}) {
    if ret == -1 {  // 系统错误
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": message,  // 通常用 constants.SYSTEM_ERROR
        })
    }
}

// Service层返回系统错误
if err != nil {
    return constants.SYSTEM_ERROR, nil, -1
}
```

**Why:** 使用统一错误信息避免暴露系统内部细节，提升安全性和用户体验。

**How to apply:** 所有数据库错误、Redis错误、第三方API错误等非业务错误都应使用 `constants.SYSTEM_ERROR`。

---

## 二、枚举类型概览

**源码位置**: `pkg/enum/` 目录

枚举使用 Go 的 `iota` 关键字定义，从0开始自动递增。

### 枚举目录结构

```
pkg/enum/
├── contact/
│   ├── contact_status_enum/    # 联系人状态
│   └── contact_type_enum/      # 联系人类型
├── contact_apply/
│   └── contact_apply_status_enum/  # 申请状态
├── group_info/
│   ├── add_mode_enum/          # 加群方式
│   └── group_status_enum/      # 群组状态
├── message/
│   ├── message_status_enum/    # 消息状态
│   └── message_type_enum/      # 消息类型
└── user_info/
    └── user_status_enum/       # 用户状态
```

---

## 三、联系人相关枚举

### 3.1 联系人状态 (contact_status_enum)

**源码位置**: `pkg/enum/contact/contact_status_enum/contact_status_enum.go`

```go
package contact_status_enum

const (
    NORMAL = iota        // 0 - 正常
    BE_BLACK             // 1 - 被拉黑
    BLACK                // 2 - 拉黑对方
    BE_DELETE            // 3 - 被删除
    DELETE               // 4 - 删除好友
    SILENCE              // 5 - 免打扰(未实现)
    QUIT_GROUP           // 6 - 退出群聊
    KICK_OUT_GROUP       // 7 - 被踢出群聊
)
```

**状态流转图**:
```
用户添加好友 → NORMAL(0)
     │
     ├─→ 拉黑对方 → BLACK(2)
     │        │
     │        └─→ 取消拉黑 → NORMAL(0)
     │
     ├─→ 被对方拉黑 → BE_BLACK(1)
     │
     ├─→ 删除好友 → DELETE(4)
     │        │
     │        └─→ 对方视角 → BE_DELETE(3)
     │
群聊进群 → NORMAL(0)
     │
     ├─→ 主动退群 → QUIT_GROUP(6)
     │
     └─→ 被踢出 → KICK_OUT_GROUP(7)
```

**Why:** 区分用户主动操作和被动状态，便于双向关系管理。

**How to apply:** 
- 查询联系人列表时过滤 `NORMAL` 状态
- 拉黑/删除时需要更新双方状态

### 3.2 联系人类型 (contact_type_enum)

**源码位置**: `pkg/enum/contact/contact_type_enum/contact_type_enum.go`

```go
package contact_type_enum

const (
    USER = iota    // 0 - 用户联系人
    GROUP          // 1 - 群组联系人
)
```

**使用场景**:
```go
// 判断联系人类型
if contact.ContactType == contact_type_enum.USER {
    // 单聊逻辑
} else if contact.ContactType == contact_type_enum.GROUP {
    // 群聊逻辑
}

// 判断接收者类型(Uuid前缀)
if receiveId[0] == 'U' {  // 用户
} else if receiveId[0] == 'G' {  // 群组
}
```

---

## 四、联系人申请状态 (contact_apply_status_enum)

**源码位置**: `pkg/enum/contact_apply/contact_apply_status_enum/contact_apply_status_enum.go`

```go
package contact_apply_status_enum

const (
    PENDING = iota    // 0 - 申请中
    AGREE             // 1 - 已同意
    REFUSE            // 2 - 已拒绝
    BLACK             // 3 - 已拉黑
)
```

**申请流程图**:
```
用户A申请添加用户B → PENDING(0)
         │
         ├─→ 用户B同意 → AGREE(1) → 创建双向联系人
         │
         ├─→ 用户B拒绝 → REFUSE(2) → 可再次申请
         │
         └─→ 用户B拉黑 → BLACK(3) → 禁止再次申请
```

**Why:** 支持多次申请机制，拒绝后可重新申请，拉黑后禁止申请。

**How to apply:**
- 查询新联系人列表: `WHERE status = PENDING`
- 通过申请后创建双向 `UserContact` 记录

---

## 五、群组相关枚举

### 5.1 加群方式 (add_mode_enum)

**源码位置**: `pkg/enum/group_info/add_mode_enum/add_mode_enum.go`

```go
package add_mode_enum

const (
    DIRECT = iota    // 0 - 直接加群
    AUDIT            // 1 - 需审核
)
```

**加群流程对比**:
```
DIRECT(0): 用户点击加群 → 直接进群 → 创建UserContact

AUDIT(1): 用户点击加群 → 创建申请记录 → 等待群主审核 → 通过后进群
```

**使用场景**:
```go
// 检查加群方式
func CheckGroupAddMode(groupId string) (string, int8, int) {
    var group model.GroupInfo
    dao.GormDB.First(&group, "uuid = ?", groupId)
    return "加群方式获取成功", group.AddMode, 0
}

// 根据加群方式处理
if addMode == add_mode_enum.DIRECT {
    EnterGroupDirectly(groupId, userId)
} else if addMode == add_mode_enum.AUDIT {
    ApplyContact(userId, groupId, contact_type_enum.GROUP, "申请加群")
}
```

### 5.2 群组状态 (group_status_enum)

**源码位置**: `pkg/enum/group_info/group_status_enum/group_status_enum.go`

```go
package group_status_enum

const (
    NORMAL = iota     // 0 - 正常
    DISABLE           // 1 - 禁用
    DISSOLVE          // 2 - 解散
)
```

**状态说明**:
- `NORMAL`: 群组正常运作，成员可发言
- `DISABLE`: 管理员禁用群组，禁止发言(软删除)
- `DISSOLVE`: 群主解散群组，彻底删除

---

## 六、消息相关枚举

### 6.1 消息状态 (message_status_enum)

**源码位置**: `pkg/enum/message/message_status_enum/message_status_enum.go`

```go
package message_status_enum

const (
    Unsent = iota    // 0 - 未发送
    Sent             // 1 - 已发送
)
```

**Why:** 区分消息发送状态，支持离线消息重发。

**How to apply:** WebSocket发送成功后更新状态为 `Sent`。

### 6.2 消息类型 (message_type_enum)

**源码位置**: `pkg/enum/message/message_type_enum/message_type_enum.go`

```go
package message_type_enum

const (
    Text = iota          // 0 - 文本消息
    Voice                // 1 - 语音消息
    File                 // 2 - 文件消息
    AudioOrVideo         // 3 - 音视频通话
)
```

**不同类型处理逻辑**:
```go
switch message.Type {
case message_type_enum.Text:
    // 存储Content字段
case message_type_enum.File:
    // 存储Url, FileType, FileName, FileSize字段
case message_type_enum.AudioOrVideo:
    // 解析AVdata字段(WebRTC信令)
}
```

---

## 七、用户状态枚举 (user_status_enum)

**源码位置**: `pkg/enum/user_info/user_status_enum/user_status_enum.go`

```go
package user_status_enum

const (
    NORMAL = iota    // 0 - 正常
    DISABLE          // 1 - 禁用
)
```

**管理员功能使用**:
```go
// 禁用用户
func DisableUsers(uuids []string) (string, int) {
    dao.GormDB.Model(&model.UserInfo{}).
        Where("uuid IN ?", uuids).
        Update("status", user_status_enum.DISABLE)
    return "已禁用用户", 0
}

// 启用用户
func AbleUsers(uuids []string) (string, int) {
    dao.GormDB.Model(&model.UserInfo{}).
        Where("uuid IN ?", uuids).
        Update("status", user_status_enum.NORMAL)
    return "已启用用户", 0
}
```

---

## 八、枚举使用最佳实践

### 8.1 导入枚举包

```go
import (
    contact_status_enum "gochat/pkg/enum/contact/contact_status_enum"
    contact_type_enum "gochat/pkg/enum/contact/contact_type_enum"
    add_mode_enum "gochat/pkg/enum/group_info/add_mode_enum"
    group_status_enum "gochat/pkg/enum/group_info/group_status_enum"
    message_type_enum "gochat/pkg/enum/message/message_type_enum"
    user_status_enum "gochat/pkg/enum/user_info/user_status_enum"
)
```

### 8.2 数据库查询使用枚举

```go
// 查询正常状态的联系人
dao.GormDB.Where("user_id = ? AND status = ?", userId, contact_status_enum.NORMAL)

// 查询申请中的申请记录
dao.GormDB.Where("contact_id = ? AND status = ?", userId, contact_apply_status_enum.PENDING)

// 查询正常状态的群组
dao.GormDB.Where("status = ?", group_status_enum.NORMAL)
```

### 8.3 枚举与数据库映射

| Go枚举值 | 数据库存储值 | 含义 |
|---------|-------------|------|
| `NORMAL` (iota=0) | 0 | 正常状态 |
| `DISABLE` (iota=1) | 1 | 禁用状态 |
| `DIRECT` (iota=0) | 0 | 直接加群 |
| `AUDIT` (iota=1) | 1 | 需审核 |

**Why:** Go的 `iota` 从0开始，与数据库INT类型完美匹配，无需转换。

---

## 九、总结

### 枚举设计原则

1. **使用 `iota`**: 自动递增，避免手动维护数值
2. **包路径清晰**: `pkg/enum/{模块}/{具体枚举}` 便于定位
3. **命名规范**: 枚举名 + `_enum` 后缀
4. **状态完整**: 覆盖所有可能状态，包括被动状态(被拉黑、被删除)

### 枚举与业务逻辑

| 模块 | 使用枚举 | 关键逻辑 |
|------|---------|---------|
| 用户管理 | `user_status_enum` | 禁用/启用用户 |
| 群组管理 | `add_mode_enum`, `group_status_enum` | 加群方式、群组状态 |
| 联系人 | `contact_status_enum`, `contact_type_enum` | 双向关系、用户/群聊 |
| 申请流程 | `contact_apply_status_enum` | 申请状态流转 |
| 消息 | `message_type_enum`, `message_status_enum` | 消息类型、发送状态 |