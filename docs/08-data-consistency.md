# GoChat 数据一致性设计 — 面试详解

## 一、项目中涉及一致性的场景

```
┌────────┐     ┌────────┐     ┌──────────┐
│ MySQL  │     │ Redis  │     │ WebSocket│
│ 持久层  │     │ 缓存层  │     │ 实时推送  │
└───┬────┘     └───┬────┘     └────┬─────┘
    │              │               │
    │   三者之间数据需要保持一致       │
    │              │               │
    ▼              ▼               ▼
  用户信息       用户信息缓存      在线用户连接表
  消息记录       消息列表缓存      实时消息推送
  会话记录       会话列表缓存      会话变更通知
```

核心问题：**MySQL 是权威数据源，Redis 是缓存加速，WebSocket 是实时通道。三者可能出现不一致。**

## 二、MySQL ↔ Redis 一致性

### 2.1 查询时：Cache-Aside 旁路缓存

```go
// 以 GetUserInfo 为例
rspString, err := redis.GetKeyNilIsErr("user_info_" + uuid)
if err != nil {
    if errors.Is(err, redis.Nil) {
        // 缓存未命中 → 查 MySQL → 写 Redis
        var user model.UserInfo
        dao.GormDB.Where("uuid=?", uuid).Find(&user)
        rsp := respond.GetUserInfoRespond{...}
        rspBytes, _ := json.Marshal(rsp)
        myredis.SetKeyEx("user_info_"+uuid, string(rspBytes), 1*time.Minute)
        return rsp
    }
}
// 缓存命中 → 直接返回（可能和MySQL有1分钟延迟）
var rsp respond.GetUserInfoRespond
json.Unmarshal([]byte(rspString), &rsp)
return rsp
```

**一致性保证**：最终一致。TTL = 1 分钟，最多 1 分钟后缓存过期重新从 MySQL 加载。

### 2.2 写入时：先更新 MySQL，再删缓存

```go
// 以 UpdateUserInfo 为例
func (u *userInfoService) UpdateUserInfo(updateReq ...) (...) {
    // 1. 先更新 MySQL（权威数据源）
    dao.GormDB.Save(&user)

    // 2. 再删除 Redis 缓存
    myredis.DelKeysWithPattern("user_info_" + updateReq.Uuid)

    return "修改用户信息成功"
}
```

**为什么是删缓存，不是更新缓存？**

| 方案 | 问题 |
|------|------|
| 更新缓存 | 并发写时可能旧值覆盖新值（A写→B写→A更新缓存→B更新缓存，顺序不对） |
| 删缓存 ✅ | 下次查询时从 MySQL 重新加载，保证和数据库一致 |

### 2.3 可能的不一致窗口

```
时间线：
  t1: 用户A查询 → 缓存miss → 查MySQL(值X) → 还没写入Redis
  t2: 管理员更新MySQL(值Y) → 删除缓存(缓存本来就没有，删了个寂寞)
  t3: t1的写入Redis执行 → 缓存值=X（旧值！）
  t4: 缓存TTL到期 → 重新从MySQL加载 → 值=Y ✅ 最终一致
```

> **Q: 怎么解决这个不一致窗口？**
> A: 当前项目没有处理。常见方案：延迟双删（删缓存 → 更新MySQL → 延迟再删一次缓存），或者用消息队列保证删除和更新的顺序。1 分钟 TTL 的存在，最坏情况 1 分钟后自动恢复，对聊天系统可接受。

## 三、MySQL ↔ WebSocket 实时消息一致性

### 3.1 先存库，再推送

```go
// Server.Start() 处理消息
case data := <-s.Transmit:
    // 1. 先存MySQL（持久化，权威数据源）
    message := model.Message{...}
    dao.GormDB.Create(&message)

    // 2. 再通过WebSocket推送
    receiveClient.SendBack <- messageBack
    sendClient.SendBack <- messageBack
```

**为什么先存再推？**

| 顺序 | 风险 |
|------|------|
| 先推再存 | 推送成功但存库失败 → 消息丢失，对方看到了但刷新后消失 |
| **先存再推** ✅ | 存库成功但推送失败 → 消息不丢，对方下次登录从MySQL拉取 |

**一致性保证**：消息不会丢。推送失败只是实时性受影响，数据最终可查。

### 3.2 消息状态：Unsent → Sent

```go
// 消息创建时
Status: message_status_enum.Unsent,   // 0=未送达

// Client.Write() 成功后更新状态
func (c *Client) Write() {
    for messageBack := range c.SendBack {
        c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
        // WebSocket写入成功 → 更新MySQL状态为已发送
        dao.GormDB.Model(&model.Message{}).
            Where("uuid=?", messageBack.Uuid).
            Update("status", message_status_enum.Sent)  // 1=已送达
    }
}
```

```
消息生命周期：
  创建 → Status=0(Unsent) → 存MySQL → WebSocket推送
                                        │
                                  推送成功 → Status=1(Sent)
                                  推送失败 → Status保持0(Unsent)
                                        │
                                  用户下次登录查MySQL → 能看到所有消息
                                  （无论Sent还是Unsent，只是状态不同）
```

> **Q: Unsent 和 Sent 的区别是什么？**
> A: 当前实现中只是一个状态标记，没有实质业务差异——用户无论哪种状态都能查到消息。如果要做得更完善，Unsent 可以表示"对方还没收到"，Sent 表示"对方已收到"，类似微信的发送成功/已读回执。

## 四、Redis ↔ WebSocket 实时消息一致性

### 4.1 消息推送时追加 Redis 缓存

```go
// Server.Start() 中推送消息后追加缓存
rspString, err := myredis.GetKeyNilIsErr("message_list_" + sendId + "_" + receiveId)
if err == nil {
    var rsp []respond.GetMessageListRespond
    json.Unmarshal([]byte(rspString), &rsp)
    rsp = append(rsp, messageRsp)        // 追加新消息
    rspByte, _ := json.Marshal(rsp)
    myredis.SetKeyEx("message_list_"+..., string(rspByte), TTL)
}
```

### 4.2 可能的不一致场景

| 场景 | Redis 状态 | MySQL 状态 | 影响 |
|------|-----------|-----------|------|
| Redis key 过期了，推送时 append miss | 空 | 有消息 | 无影响，下次查询走 Cache-Aside 从 MySQL 加载 |
| 推送成功但 Redis 写入失败 | 旧数据 | 有新消息 | 无影响，TTL 到期后从 MySQL 加载 |
| 消息存 MySQL 失败（极端情况） | 没有这条消息 | 没有这条消息 | 前端看到了推送但刷新后消失（当前没有回滚机制） |

> **Q: 第三种情况怎么处理？**
> A: 当前没有处理。严格来说应该用事务：MySQL 存库和 Redis 更新放在同一个流程里，MySQL 失败就不推送。但当前代码是存库成功后直接推送，没有事务保护。对聊天系统来说，偶尔丢一条消息比系统不可用更容易接受。

## 五、软删除一致性

### 5.1 GORM 自动过滤

```go
// 所有带 DeletedAt 字段的模型，查询时自动加 WHERE deleted_at IS NULL
dao.GormDB.Where("uuid=?", uuid).Find(&user)
// 实际 SQL: SELECT * FROM user_info WHERE uuid=? AND deleted_at IS NULL

// 软删除
dao.GormDB.Delete(&user)
// 实际 SQL: UPDATE user_info SET deleted_at=NOW() WHERE uuid=?
```

### 5.2 软删除后清理缓存

```go
// DeleteSession 中
func (s *sessionService) DeleteSession(ownerId, sessionId string) (string, int) {
    // 1. 软删除（MySQL）
    session.DeletedAt.Valid = true
    session.DeletedAt.Time = time.Now()
    dao.GormDB.Save(&session)

    // 2. 清理相关Redis缓存
    myredis.DelKeysWithSuffix(sessionId)
    myredis.DelKeysWithPattern("session_list_" + ownerId)
    myredis.DelKeysWithPattern("group_session_list_" + ownerId)
}
```

**保证**：MySQL 软删除后，对应 Redis 缓存立即失效，下次查询不会返回已删除的数据。

## 六、防越权一致性

### 6.1 SendId 强制覆盖

```go
// Client.Read() 中
message.SendId = c.Uuid  // 用JWT认证后的UUID覆盖前端传的SendId
```

防止客户端伪造身份发消息。即使前端篡改 `send_id`，后端会用 JWT 中的真实 UUID 替换。

### 6.2 CheckOwner 校验

```go
// Controller 层
func CheckOwner(c *gin.Context, ownerId string) bool {
    tokenUuid := GetTokenUuid(c)    // 从JWT取uuid
    if tokenUuid != ownerId {
        c.JSON(200, gin.H{"code": 403, "message": "无权操作他人数据"})
        return false
    }
    return true
}
```

防止用户操作他人数据（如修改别人信息、删除别人的会话）。

## 七、优雅关闭时的一致性

```go
func main() {
    <-quit  // 收到终止信号

    // 1. 关闭Kafka
    if kafkaConfig.MessageMode == "kafka" {
        kafka.KafkaService.KafkaClose()
    }

    // 2. 关闭ChatServer
    chat.ChatServer.Close()

    // 3. 删除所有Redis缓存
    myredis.DeleteAllRedisKeys()
    // 防止下次启动时读到过期数据
    // 下次启动后首次查询会走Cache-Aside从MySQL重新加载
}
```

> **Q: 为什么关闭时要清所有 Redis？**
> A: 运行期间 Redis 缓存可能和 MySQL 有短暂不一致（如缓存删了还没回写）。关闭时清空缓存，下次启动后所有查询都走 MySQL 加载，保证启动后数据一定一致。代价是启动后第一波查询会慢一点（全部 Cache Miss）。

## 八、一致性总结

| 数据对 | 一致性策略 | 一致性级别 | 最大延迟 |
|--------|-----------|-----------|---------|
| MySQL → Redis（查询） | Cache-Aside | 最终一致 | 1分钟（TTL） |
| MySQL → Redis（写入） | 先写MySQL再删缓存 | 最终一致 | 下次查询时 |
| MySQL → WebSocket | 先存库再推送 | 最终一致 | 无（推送失败消息不丢） |
| WebSocket → Redis | 推送时 append 缓存 | 最佳努力 | TTL到期后恢复 |
| 软删除 → 缓存 | 删库后删缓存 | 强一致 | 无（同步删除） |
| SendId 身份 | JWT 覆盖 | 强一致 | 无（同步覆盖） |

> **Q: 为什么不用强一致性？**
> A: 聊天系统对一致性的要求是"消息不丢"，而不是"毫秒级一致"。1 分钟内用户刷新页面能看到正确数据就够了。强一致性需要分布式锁或事务，代价是性能大幅下降，对聊天系统来说不值得。

## 九、面试速记口诀

1. **MySQL 权威**：MySQL 是唯一权威数据源，Redis 只是加速层，WebSocket 只是实时通道
2. **先存再推**：消息先存 MySQL 再 WebSocket 推送，推送失败消息不丢，上线后从 MySQL 拉取
3. **写后删缓存**：写操作先更新 MySQL 再删 Redis 缓存，不用更新缓存（避免并发覆盖）
4. **最终一致**：Redis TTL 1 分钟，最坏 1 分钟后自动恢复一致
5. **状态追踪**：消息 Status 从 Unsent→Sent 标记是否送达，但两种状态都可见
6. **防越权**：SendId 强制 JWT 覆盖 + CheckOwner 校验，保证身份和权限一致
7. **优雅关闭**：关服时清空所有 Redis 缓存，下次启动从 MySQL 全量加载，保证启动后一致
