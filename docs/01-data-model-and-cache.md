# GoChat 数据模型与缓存设计 — 面试详解

## 一、整体架构

```
┌──────────┐     ┌───────────┐     ┌──────────┐
│ Controller│ ──► │  Service   │ ──► │  DAO     │
│ (api/v1/) │     │(service/  │     │(dao/)    │
│           │     │ gorm/)    │     │          │
└──────────┘     └─────┬─────┘     └────┬─────┘
                       │                 │
              ┌────────┴────────┐        │
              ▼                 ▼        ▼
          ┌────────┐      ┌────────┐ ┌────────┐
          │ Redis  │      │ MySQL  │ │ MySQL  │
          │ 缓存层  │      │ 持久层  │ │ 连接池  │
          └────────┘      └────────┘ └────────┘

查询流程：
  1. 先查 Redis 缓存
  2. 缓存命中 → 直接返回（不查 MySQL）
  3. 缓存未命中 → 查 MySQL → 写入 Redis → 返回

写入流程：
  1. 写 MySQL（持久化）
  2. 删除/更新对应 Redis 缓存
  3. 下次查询时重新从 MySQL 加载到 Redis
```

## 二、数据库设计（6 张表）

### 2.1 表总览

| 表名 | 模型 | 前缀 | 软删除 | 说明 |
|------|------|------|--------|------|
| `user_info` | UserInfo | U | ✅ | 用户表 |
| `user_contact` | UserContact | — | ✅ | 联系人关系表 |
| `contact_apply` | ContactApply | A | ✅ | 联系人申请表 |
| `group_info` | GroupInfo | G | ✅ | 群组表 |
| `session` | Session | S | ✅ | 会话表 |
| `message` | Message | M | ❌ | 消息表 |

> **Q: 为什么 message 表没有软删除？**
> A: 消息是聊天记录的核心数据，不应该被"软删除"后还占空间。消息一旦发送就是事实，不允许撤回删除（当前没有撤回功能）。其他表用软删除是因为用户/群组/联系人可能需要恢复。

### 2.2 UUID 生成规则

所有实体 ID 都是 **20 字符字符串**，由三部分拼接：

```
前缀(1位) + 日期(8位) + 随机数(11位) = 20位
  U/G/S/A/M  20260607   38475629184

示例：U2026060738475629184
```

| 前缀 | 含义 | 示例 |
|------|------|------|
| `U` | 用户 | U2026042459746164431 |
| `G` | 群组 | G2026042567123847654 |
| `S` | 会话 | S2026051048510829861 |
| `A` | 申请 | A2026060112345678901 |
| `M` | 消息 | M2026060738475629184 |

生成代码（`pkg/util/random/random_int.go`）：

```go
func GetNowAndLenRandomString(n int) (string, error) {
    randomInt, _ := GetRandomInt(n)  // 生成 n 位随机数
    return time.Now().Format("20060102") + strconv.Itoa(randomInt), nil
}

// 调用处拼接前缀
uuid: fmt.Sprintf("U%s", value)  // U + 日期8位 + 随机11位 = 20位
```

> **Q: 为什么不用自增 ID？为什么不用 UUID？**
> A: 不用自增 ID 是因为分布式场景下自增会冲突，且暴露业务数据量。不用标准 UUID 是因为太长（36 位），且无业务语义。自定义 UUID 方案：前缀带类型信息，拿到 ID 就知道是用户还是群组；日期部分支持按时间排序；随机部分保证唯一性。

### 2.3 用户表（user_info）

```go
type UserInfo struct {
    Id            int64          // 自增主键
    Uuid          string         // 用户唯一ID，U前缀，char(20)，唯一索引
    Nickname      string         // 昵称，varchar(20)
    Telephone     string         // 电话，char(11)，普通索引
    Email         string         // 邮箱，char(30)
    Avatar        string         // 头像URL，char(255)
    Gender        int8           // 性别：0=男，1=女
    Signature     string         // 个性签名，varchar(100)
    Password      string         // 密码，char(18)
    Birthday      string         // 生日，char(8)
    CreatedAt     time.Time      // 注册时间，索引
    DeletedAt     gorm.DeletedAt // 软删除时间，索引
    LastOnlineAt  sql.NullTime   // 上次登录时间
    LastOfflineAt sql.NullTime   // 最近离线时间
    IsAdmin       int8           // 是否管理员：0=否，1=是
    Status        int8           // 状态：0=正常，1=禁用，索引
}
```

| 索引 | 字段 | 用途 |
|------|------|------|
| 唯一索引 | `uuid` | 登录/查询时快速定位用户 |
| 普通索引 | `telephone` | 按手机号查询 |
| 普通索引 | `status` | 管理员批量查询禁用用户 |
| 普通索引 | `created_at` | 按注册时间排序 |
| 软删除索引 | `deleted_at` | GORM 软删除过滤 |

> **Q: 为什么 `LastOnlineAt` 用 `sql.NullTime` 而不是 `time.Time`？**
> A: 新注册用户从未登录过，`LastOnlineAt` 应该是 NULL 而不是零值（`0001-01-01`）。`time.Time` 的零值写入 MySQL 会变成 `0001-01-01 00:00:00`，语义不对。`sql.NullTime` 可以区分"没有值"和"有值"。

### 2.4 联系人关系表（user_contact）

```go
type UserContact struct {
    Id          int64          // 自增主键
    UserId      string         // 用户ID，char(20)，索引
    ContactId   string         // 联系人ID，char(20)，索引
    ContactType int8           // 类型：0=用户，1=群聊
    Status      int8           // 状态：0=正常，1=拉黑，2=被拉黑，3=删除
    CreatedAt   time.Time      // 创建时间
    UpdateAt    time.Time      // 更新时间（拉黑/删除时更新）
    DeletedAt   gorm.DeletedAt // 软删除
}
```

**状态机设计**：

```
          申请通过              拉黑              解除拉黑
  申请中 ───────► 正常(0) ───────► 拉黑(1) ───────► 正常(0)
                    │                              │
                    │ 被对方拉黑                     │ 删除好友
                    ▼                              ▼
                被拉黑(2)                       删除(3)
```

> **Q: 为什么拉黑和被拉黑是两个不同状态？**
> A: A 拉黑 B，对 A 来说状态是"拉黑(1)"，对 B 来说状态是"被拉黑(2)"。同一条记录的 Status 只能表示一方的视角，所以需要在双方的记录中分别标记。A 拉黑 B 时：A 的记录 status=1，B 的记录 status=2。

### 2.5 群组表（group_info）

```go
type GroupInfo struct {
    Id        int64           // 自增主键
    Uuid      string          // 群组ID，G前缀，唯一索引
    Name      string          // 群名称，varchar(20)
    Notice    string          // 群公告，varchar(500)
    Members   json.RawMessage // 群成员列表，JSON 格式存储
    MemberCnt int             // 群人数，默认1
    OwnerId   string          // 群主UUID，char(20)
    AddMode   int8            // 加群方式：0=直接，1=审核
    Avatar    string          // 群头像
    Status    int8            // 状态：0=正常，1=禁用，2=解散
    CreatedAt time.Time       // 创建时间，索引
    UpdatedAt time.Time       // 更新时间
    DeletedAt gorm.DeletedAt  // 软删除，索引
}
```

**Members 字段的 JSON 存储**：

```json
["U2026042459746164431", "U2026042869644063371", "U2026050112345678901"]
```

> **Q: 为什么群成员用 JSON 存储，而不是关联表？**
> A: 简单场景下的取舍。关联表（group_member）需要额外维护一张表和 JOIN 查询，但支持大群（万人群）和复杂查询。JSON 存储简单直接，GORM 原生支持 `json.RawMessage`，查群消息时反序列化即可。缺点是成员多时 JSON 体积大、无法用 SQL 查询特定成员。当前面向中小群聊，JSON 够用。

### 2.6 会话表（session）

```go
type Session struct {
    Id            int64          // 自增主键
    Uuid          string         // 会话ID，S前缀，唯一索引
    SendId        string         // 创建者ID，char(20)，索引
    ReceiveId     string         // 接收者ID，char(20)，索引
    ReceiveName   string         // 接收者名称
    Avatar        string         // 接收者头像
    LastMessage   string         // 最新消息（会话列表预览用）
    LastMessageAt sql.NullTime   // 最新消息时间
    CreatedAt     time.Time      // 创建时间，索引
    DeletedAt     gorm.DeletedAt // 软删除，索引
}
```

**会话 = 聊天窗口**。用户 A 和用户 B 聊天，A 会有一条 session（receiveId=B），B 也会有一条 session（receiveId=A）。各自管理自己的会话列表。

> **Q: 为什么不做"一个会话双方共享"？**
> A: 因为双方的会话状态可能不同（A 删除了会话，B 没有），各自一条记录更灵活。类似微信的设计——删除聊天窗口只是删自己的，对方的还在。

### 2.7 消息表（message）

```go
type Message struct {
    Id         int64        // 自增主键
    Uuid       string       // 消息ID，M前缀，唯一索引
    SessionId  string       // 所属会话ID，索引
    Type       int8         // 类型：0=文本，1=语音，2=文件，3=音视频通话
    Content    string       // 文本内容，TEXT
    Url        string       // 文件/语音 URL
    SendId     string       // 发送者ID，索引
    SendName   string       // 发送者昵称
    SendAvatar string       // 发送者头像
    ReceiveId  string       // 接收者ID，索引
    FileType   string       // 文件类型
    FileName   string       // 文件名
    FileSize   string       // 文件大小
    Status     int8         // 状态：0=未发送，1=已发送
    CreatedAt  time.Time    // 创建时间
    SendAt     sql.NullTime // 实际发送时间
    AVdata     string       // 音视频通话数据（WebRTC 信令 JSON）
}
```

**消息类型与字段对应**：

| Type | Content | Url | AVdata | 说明 |
|------|---------|-----|--------|------|
| 0 文本 | ✅ 文本内容 | 空 | 空 | 普通聊天消息 |
| 1 语音 | 空 | ✅ 语音文件URL | 空 | 语音消息 |
| 2 文件 | 空 | ✅ 文件URL | 空 | 文件传输 |
| 3 通话 | 空 | 空 | ✅ WebRTC信令 | 音视频通话 |

**Status 的作用**：

```
消息创建 → Status=0（Unsent，已入库但未送达）
        → Client.Write() 成功写回 WebSocket → Status=1（Sent，已送达）
```

> **Q: 为什么消息要存 SendName 和 SendAvatar，不直接关联 user_info 表？**
> A: 冗余存储，避免 JOIN 查询。消息是高频读取的数据，每次显示消息都要知道发送者信息，如果 JOIN user_info 表，查询开销大。冗余存储的代价是用户改名后历史消息还是旧名字，但聊天记录本身就应该保留发送时的状态，类似微信。

### 2.8 联系人申请表（contact_apply）

```go
type ContactApply struct {
    Id          int64          // 自增主键
    Uuid        string         // 申请ID，A前缀，唯一索引
    UserId      string         // 申请人ID，索引
    ContactId   string         // 被申请人ID，索引
    ContactType int8           // 类型：0=用户，1=群聊
    Status      int8           // 状态：0=申请中，1=通过，2=拒绝，3=拉黑
    Message     string         // 申请信息，varchar(100)
    LastApplyAt time.Time      // 最后申请时间
    DeletedAt   gorm.DeletedAt // 软删除，索引
}
```

## 三、GORM 使用要点

### 3.1 连接池配置（`dao/gorm.go`）

```go
sqlDB.SetMaxIdleConns(10)       // 空闲连接数
sqlDB.SetMaxOpenConns(100)      // 最大连接数
sqlDB.SetConnMaxLifetime(time.Hour)  // 连接最大存活时间 1 小时
```

| 参数 | 值 | 作用 |
|------|-----|------|
| `MaxIdleConns` | 10 | 保持 10 个空闲连接，减少新建连接开销 |
| `MaxOpenConns` | 100 | 最多 100 个连接，防止连接数打满 MySQL |
| `ConnMaxLifetime` | 1h | 连接最多活 1 小时，防止 MySQL 主动断开长期空闲连接导致报错 |

### 3.2 AutoMigrate 自动建表

```go
GormDB.AutoMigrate(
    &model.UserInfo{},
    &model.GroupInfo{},
    &model.UserContact{},
    &model.ContactApply{},
    &model.Session{},
    &model.Message{},
)
```

启动时自动检测表结构，不存在则创建，缺少字段则添加。**不会删除列或修改列类型**，安全可重复执行。

### 3.3 软删除机制

使用 `gorm.DeletedAt` 字段：

```go
// 查询时自动过滤已删除记录
dao.GormDB.Where("uuid=?", uuid).Find(&user)
// 实际 SQL: SELECT * FROM user_info WHERE uuid=? AND deleted_at IS NULL

// 软删除：设置 deleted_at 为当前时间
dao.GormDB.Delete(&user)
// 实际 SQL: UPDATE user_info SET deleted_at=NOW() WHERE uuid=?
```

> **Q: 软删除的好处？**
> A: 数据可恢复。用户注销账号后如果后悔，管理员可以恢复数据。硬删除是不可逆的。GORM 的软删除是 `DeletedAt` 字段，查询时自动加 `WHERE deleted_at IS NULL` 过滤，不需要手动写。

## 四、Redis 缓存设计

### 4.1 缓存 Key 命名规则

| Key 模式 | 示例 | TTL | 说明 |
|----------|------|-----|------|
| `user_info_{uuid}` | `user_info_U2026042459746164431` | 1 分钟 | 用户信息缓存 |
| `message_list_{sendId}_{receiveId}` | `message_list_U1_U2` | 1 分钟 | 私聊消息列表缓存 |
| `group_messagelist_{groupId}` | `group_messagelist_G1` | 1 分钟 | 群聊消息列表缓存 |
| `session_list_{uuid}` | `session_list_U1` | 1 分钟 | 用户会话列表缓存 |
| `group_session_list_{uuid}` | `group_session_list_U1` | 1 分钟 | 群聊会话列表缓存 |
| `contact_user_list_*` | `contact_user_list_U1` | 1 分钟 | 联系人列表缓存 |
| `email_code_{email}` | `email_code_xxx@qq.com` | 5 分钟 | 邮箱验证码 |

### 4.2 缓存读写模式 — Cache-Aside（旁路缓存）

所有 Service 层查询都遵循同一模式：

```
读取：
  1. redis.GetKey(key)
  2. 命中 → 反序列化 → 返回
  3. 未命中(redis.Nil) → 查 MySQL → 序列化 → redis.SetKeyEx(key, json, TTL) → 返回

写入：
  1. 写 MySQL
  2. 删除对应 Redis 缓存（DelKeysWithPattern）
  3. 下次查询时重新从 MySQL 加载
```

以 `GetUserInfo` 为例：

```go
func (u *userInfoService) GetUserInfo(uuid string) (...) {
    // 1. 先查 Redis
    rspString, err := myredis.GetKeyNilIsErr("user_info_" + uuid)
    if err != nil {
        if errors.Is(err, redis.Nil) {
            // 2. 缓存未命中，查 MySQL
            var user model.UserInfo
            dao.GormDB.Where("uuid=?", uuid).Find(&user)
            rsp := respond.GetUserInfoRespond{...}

            // 3. 写入 Redis，TTL 1 分钟
            rspBytes, _ := json.Marshal(rsp)
            myredis.SetKeyEx("user_info_"+uuid, string(rspBytes), 
                time.Minute * constants.REDIS_TIMEOUT)
            return rsp
        }
    }
    // 4. 缓存命中，反序列化返回
    var rsp respond.GetUserInfoRespond
    json.Unmarshal([]byte(rspString), &rsp)
    return rsp
}
```

以 `UpdateUserInfo` 为例（缓存失效）：

```go
func (u *userInfoService) UpdateUserInfo(updateReq ...) (...) {
    // 1. 更新 MySQL
    dao.GormDB.Save(&user)

    // 2. 删除 Redis 缓存
    myredis.DelKeysWithPattern("user_info_" + updateReq.Uuid)

    return "修改用户信息成功"
}
```

> **Q: 为什么写操作是删缓存而不是更新缓存？**
> A: 删缓存更简单且安全。更新缓存的问题：如果并发写，可能出现旧值覆盖新值。删缓存让下次查询时重新加载，保证数据最终一致。代价是多一次 Cache Miss，但写操作频率远低于读，可以接受。

### 4.3 消息列表缓存的双写

聊天消息比较特殊，除了查询时的 Cache-Aside，WebSocket 实时消息也会更新缓存：

```
场景1：用户打开聊天窗口（首次查询）
  → Redis 未命中 → 查 MySQL 全量 → 写入 Redis

场景2：用户在当前聊天窗口收到新消息（WebSocket 推送）
  → Server.Start() 处理消息时：
     1. 存 MySQL
     2. WebSocket 推送给前端
     3. 追加到 Redis 缓存（append 方式，不用重新查 MySQL）

场景3：用户切换聊天窗口后切回来
  → Redis 命中 → 直接返回（包含场景2追加的消息）
```

```go
// Server.Start() 中消息推送后追加 Redis 缓存
rspString, err := myredis.GetKeyNilIsErr("message_list_" + sendId + "_" + receiveId)
if err == nil {
    var rsp []respond.GetMessageListRespond
    json.Unmarshal([]byte(rspString), &rsp)
    rsp = append(rsp, messageRsp)         // 追加新消息
    rspByte, _ := json.Marshal(rsp)
    myredis.SetKeyEx("message_list_"+..., string(rspByte), TTL)  // 写回 Redis
}
```

> **Q: 这种 append 方式有什么问题？**
> A: 如果 Redis key 过期了（TTL 到了），但消息还在 WebSocket 通道中传输，可能出现 Redis 里没缓存但消息已推送的情况，下次查询走 MySQL 全量加载，不会丢消息，只是这次 append 会 miss。整体是最终一致性，不影响功能。

### 4.4 缓存删除策略

| 操作 | 删除的缓存 Key | 原因 |
|------|---------------|------|
| 更新用户信息 | `user_info_{uuid}` | 用户信息变更，缓存失效 |
| 创建会话 | `session_list_{ownerId}` | 会话列表新增，缓存失效 |
| 删除会话 | `session_list_{ownerId}` + 后缀匹配 `sessionId` | 会话列表变更 + 具体会话缓存失效 |
| 新增联系人 | `contact_user_list_*` | 联系人列表变更 |

### 4.5 Redis Scan 替代 Keys

项目使用 `Scan` 增量迭代查找 key，而不是 `Keys` 命令：

```go
// ✅ 用 Scan（项目实现）
keys, nextCursor, _ := redisClient.Scan(ctx, cursor, pattern, 100).Result()

// ❌ 用 Keys（生产环境禁止）
keys, _ := redisClient.Keys(ctx, pattern).Result()
```

> **Q: 为什么禁止 Keys 命令？**
> A: `Keys` 会遍历整个 Redis 实例的所有 key，是 O(N) 操作，会阻塞 Redis，导致其他请求超时。`Scan` 是增量迭代，每次只返回少量结果，不阻塞 Redis，适合生产环境。

### 4.6 优雅关闭时清理缓存

```go
// main.go 中的优雅关闭
func main() {
    <-quit

    // 删除所有 Redis 键
    myredis.DeleteAllRedisKeys()

    zlog.Info("服务器已关闭")
}
```

服务关闭时清除所有缓存，防止下次启动时读到过期数据。

## 五、面试速记口诀

1. **6 张表**：用户(U)、联系人、联系人申请(A)、群组(G)、会话(S)、消息(M)
2. **UUID 规则**：前缀(1位) + 日期(8位) + 随机(11位) = 20位，前缀带类型语义
3. **软删除**：5 张表用 `gorm.DeletedAt`，消息表不用（聊天记录不可删）
4. **缓存模式**：Cache-Aside 旁路缓存，读时回写，写时删除
5. **缓存 TTL**：1 分钟（`REDIS_TIMEOUT`），邮箱验证码 5 分钟
6. **消息双写**：查询时从 MySQL 回写 Redis，WebSocket 推送时 append Redis
7. **Scan 替代 Keys**：增量迭代不阻塞 Redis，生产环境必须用 Scan
8. **冗余存储**：消息表存 SendName/SendAvatar，避免 JOIN，保留历史状态
