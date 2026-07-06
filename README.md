# GoChat

基于 Go + React + TypeScript 开发的全栈即时通讯应用，集成 CloudWeGo Eino AI 助手。

## 技术栈

### 后端

| 技术 | 版本 | 说明 |
|------|------|------|
| Go | 1.26.1+ | 后端语言 |
| Gin | 1.12.0 | Web 框架 |
| GORM | 1.31.1 | ORM 框架 |
| MySQL | 8.0+ | 数据库 |
| Redis | 6.0+ | 缓存服务 |
| Zap | 1.27.1 | 日志框架 |
| Gorilla WebSocket | 1.5.3 | WebSocket |
| golang-jwt/v5 | 5.3.1 | JWT 认证 |
| CloudWeGo Eino | 0.9.12 | AI Agent 框架 |
| segmentio/kafka-go | 0.4.51 | 消息队列（可选） |
| QQ SMTP | - | 邮箱验证码 |

### 前端

| 技术 | 版本 | 说明 |
|------|------|------|
| React | 18 | 前端框架 |
| TypeScript | 5.x | 类型安全 |
| Vite | 5.x | 构建工具 |
| Zustand | 5.x | 状态管理 |
| React Router | 6.x | 路由管理 |
| Ant Design | 5.x | UI 组件库 |
| Axios | 1.x | HTTP 请求 |

## 项目结构

```
gochat/
├── cmd/gochat/               # 入口文件
├── api/v1/                   # Controller 层
├── internal/
│   ├── agent/                # AI Agent 服务（Eino + Mock 双实现）
│   ├── config/               # 配置加载（TOML）
│   ├── dao/                  # 数据库连接
│   ├── dto/
│   │   ├── request/          # 请求结构体
│   │   └── respond/          # 响应结构体
│   ├── middleware/           # JWT 认证 + 管理员鉴权中间件
│   ├── model/                # 数据库模型（6 张表）
│   ├── service/
│   │   ├── gorm/             # Service 层（业务逻辑）
│   │   ├── redis/            # Redis 缓存服务
│   │   ├── email/            # 邮箱验证码服务
│   │   ├── chat/             # WebSocket 聊天服务
│   │   └── kafka/            # Kafka 消息服务（可选）
│   └── https_server/         # HTTP 服务器与路由注册
├── pkg/
│   ├── constants/            # 常量定义
│   ├── enum/                 # 枚举定义
│   ├── jwt/                  # JWT 生成与解析
│   ├── util/                 # 工具函数（随机数等）
│   └── zlog/                 # Zap 日志封装
├── docs/
│   ├── lessons/              # 教学文档（26 篇，架构→JWT 全链路）
│   └── build-log/            # 仿微信 Go 从零开发叙事文档
├── configs/                  # 配置文件
├── static/                   # 静态资源（头像/文件）
├── logs/                     # 日志文件
└── frontend/                 # React 前端项目
```

## 三层架构

```
Request → Controller → Service → DAO → Database
         ↑ JsonBack ←───────────────┘
```

开发顺序：Model → DAO → Service → Controller → 路由注册

---

## AI 助手功能

项目集成了基于 CloudWeGo Eino 的 AI 助手，作为系统用户 `AI助手` 存在。

- **私聊 Agent**：用户给 `AI助手` 发私聊消息，读取最近 N 条历史作为上下文，生成回复并写入消息表、推送 WebSocket
- **群聊 Agent**：群消息含 `@AI助手`、`@agent` 或 `/ai` 前缀时触发，回复带 `@提问人`，未被触发不主动发言
- **Provider 切换**：通过配置文件 `[llmConfig]` 或环境变量切换 `mock`（本地无 key 调试）/ `openai`（真实 LLM），支持任何 OpenAI 兼容接口
- **超时与降级**：Agent 调用 25 秒超时，失败返回友好提示；Eino 初始化失败自动回退 Mock
- **安全**：复用 JWT 鉴权的 userID，禁止客户端伪造；私聊 Agent 只读当前用户与 Agent 的历史，群聊 Agent 只读当前群历史

### 记忆机制

AI 助手**不维护独立的记忆存储**，记忆 = `message` 表里的聊天记录本身。每次被触发时实时拉最近 N 条历史作为上下文注入 LLM，超出窗口的自动「遗忘」。

| 场景 | 上下文来源 | 窗口大小 | 角色映射 |
|------|-----------|---------|---------|
| 私聊 | `message` 表中「用户 ↔ Agent」双向最近消息 | 最近 10 条 | 用户发 → `user`；Agent 发 → `assistant` |
| 群聊 | `message` 表中该群最近消息 | 最近 20 条 | Agent 发 → `assistant`；群成员发 → `user`（带 `昵称: ` 前缀） |

调用流程：构建 `[system, 历史消息..., 当前问题]` → 调用 LLM → Agent 回复写入 `message` 表 → 推送 WebSocket + 追加到 Redis 消息缓存（1 分钟 TTL，仅读加速）。

设计取舍：
- **优点**：实现极简、记忆与前端聊天记录天然一致（用户能看到的内容就是 Agent「记得」的内容）、无需额外存储
- **局限**：窗口外的对话彻底丢失、长对话 token 成本线性增长、无摘要/用户画像等长期记忆

相关常量在 [pkg/constants/agent.go](pkg/constants/agent.go)：`AgentPrivateContextLen=10`、`AgentGroupContextLen=20`、`AgentMaxInputLen=4000`、`AgentTimeoutSec=25`。

详细配置见 [AI 助手配置](#4-ai-助手配置可选) 一节。

---

## 部署指南

### 环境要求

- Go 1.26.1+
- Node.js 16.0+
- MySQL 8.0+
- Redis 6.0+
- Kafka 3.0+（可选，单机部署可不用）

### 1. 克隆项目

```bash
git clone <repo-url>
cd gochat
```

### 2. 配置数据库

创建 MySQL 数据库：

```sql
CREATE DATABASE gochat CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

> GORM 的 `AutoMigrate` 会在启动时自动创建表，无需手动建表。

### 3. 修改配置文件

复制模板并编辑 `configs/config_local.toml`：

```bash
cp configs/config.toml configs/config_local.toml
```


```toml
[mainConfig]
appName = "gochat"
host = "127.0.0.1"
port = 8000

[mysqlConfig]
host = "127.0.0.1"
port = 3306
user = "root"
password = "your_password"        # 改为你的 MySQL 密码
databaseName = "gochat"

[logConfig]
logPath = "./logs/app.log"

[staticSrcConfig]
staticAvatarPath = "./static/avatars"
staticFilePath = "./static/files"

[redisConfig]
host = "127.0.0.1"
port = 6379
password = ""                     # 改为你的 Redis 密码（如有）
db = 0

[emailConfig]
smtpHost     = "smtp.qq.com"      # SMTP 服务器（默认 QQ 邮箱）
smtpPort     = 465                # SSL 端口
smtpUsername = "your@qq.com"      # 发件邮箱
smtpPassword = "xxxxxxxxxxxxxxxx"  # QQ 邮箱授权码（16 位，非 QQ 密码）
fromName     = "GoChat"           # 发件人名称

[jwtConfig]
secret = "change-this-to-a-random-secret"  # 生产环境必须修改
expireHours = 24

[kafkaConfig]
messageMode = "channel"           # 消息模式：channel（单机）或 kafka（分布式）
hostPort = "127.0.0.1:9092"
loginTopic = "login"
chatTopic = "chat_message"
logoutTopic = "logout"
partition = 0
timeout = 1
```

**注意事项：**
- 邮箱验证码使用 QQ 邮箱 SMTP，需要去 QQ 邮箱设置中开启 SMTP 服务并获取 16 位授权码
- `messageMode` 设为 `channel` 时无需安装 Kafka，适合单机部署
- `messageMode` 设为 `kafka` 时需要安装并启动 Kafka，适合分布式部署
- `jwtConfig.secret` 不能使用默认占位符，启动时会校验并拒绝运行
- 生产环境建议修改 `host` 为 `0.0.0.0`

### 4. AI 助手配置（可选）

AI 助手默认使用 Mock 实现，无需任何配置即可体验。接入真实大模型有两种方式，**环境变量优先级高于配置文件**：

**方式一：配置文件（推荐，持久化）**

在 `configs/config_local.toml` 中配置 `[llmConfig]` 节：

```toml
[llmConfig]
provider = "openai"                                                       # mock | openai
apiKey = "your-api-key"                                                   # API Key（兼容接口可能为 "AppID:Secret" 形式）
baseUrl = "https://maas-coding-api.cn-huabei-1.xf-yun.com/v2"             # OpenAI 兼容接口地址
model = "xopdeepseekv4flash"                                              # 模型名
```

**方式二：环境变量（临时覆盖）**

```bash
export LLM_PROVIDER=openai                          # mock | openai
export OPENAI_API_KEY=your-api-key                  # API Key
export OPENAI_BASE_URL=https://api.deepseek.com/    # OpenAI 兼容接口（可选）
export LLM_MODEL=deepseek-chat                      # 模型名
```

支持的 Provider：
- `mock`（默认）：本地不调用 LLM，返回模拟回复，适合开发调试
- `openai`：通过 Eino 调用 OpenAI 或任何 OpenAI 兼容接口（DeepSeek / Qwen / Moonshot / 讯飞星火 MaaS 等）。讯飞星火 MaaS 的 `apiKey` 为 `AppID:Secret` 形式，整体作为 Bearer token 传入即可

### 3a. 安装 Kafka（可选）

如果需要分布式消息模式，需安装 Kafka：

```bash
# 下载并解压 Kafka
# 启动 ZooKeeper
bin/zookeeper-server-start.sh config/zookeeper.properties

# 启动 Kafka
bin/kafka-server-start.sh config/server.properties

# 创建 Topic
bin/kafka-topics.sh --create --topic login --bootstrap-server localhost:9092
bin/kafka-topics.sh --create --topic chat_message --bootstrap-server localhost:9092
bin/kafka-topics.sh --create --topic logout --bootstrap-server localhost:9092
```

然后将 `config_local.toml` 中 `messageMode` 改为 `kafka`。

> 单机部署跳过此步骤，使用默认的 `channel` 模式即可。

### 5. 启动后端

```bash
# 安装 Go 依赖
go mod download

# 启动服务
go run cmd/gochat/main.go
```

后端默认运行在 `http://127.0.0.1:8000`

#### 生产构建

```bash
go build -o gochat-server cmd/gochat/main.go
./gochat-server
```

### 6. 启动前端

```bash
cd frontend

# 安装依赖
npm install

# 开发模式启动（默认端口 5173）
npm run dev
```

前端开发模式运行在 `http://localhost:5173`，会自动代理 API 请求到后端。

#### 前端配置

编辑 `frontend/src/utils/constants.ts`，修改后端地址：

```typescript
export const BACKEND_URL = 'http://127.0.0.1:8000'  // 后端地址
export const WS_URL = 'ws://127.0.0.1:8000'         // WebSocket 地址
```

#### 生产构建

```bash
cd frontend
npm run build
```

构建产物在 `frontend/dist/`，后端已配置静态文件服务，可直接通过 `http://127.0.0.1:8000` 访问前端页面。

### 7. 验证部署

1. 访问 `http://127.0.0.1:8000` 进入登录页
2. 注册账号（需要邮箱验证码）
3. 登录后即可使用聊天功能
4. 在联系人中找到 `AI助手`，发消息即可触发 Agent 回复

---

## API 接口

### 用户模块

| 接口 | 路由 | 说明 |
|------|------|------|
| Login | `/login` | 邮箱+密码登录 |
| Register | `/register` | 注册 |
| EmailLogin | `/user/emailLogin` | 邮箱+验证码登录 |
| SendEmailCode | `/user/sendEmailCode` | 发送邮箱验证码 |
| VerifyEmailCode | `/user/verifyEmailCode` | 验证邮箱验证码 |
| UpdateUserInfo | `/user/updateUserInfo` | 更新用户信息 |
| GetUserInfo | `/user/getUserInfo` | 获取用户信息 |
| GetUserInfoList | `/user/getUserInfoList` | 获取用户列表 |
| WsLogin | `/user/wsLogin` | WebSocket 连接 |
| WsLogout | `/user/wsLogout` | WebSocket 登出 |
| AbleUsers | `/user/ableUsers` | 启用用户（管理员） |
| DisableUsers | `/user/disableUsers` | 禁用用户（管理员） |
| DeleteUsers | `/user/deleteUsers` | 删除用户（管理员） |
| SetAdmin | `/user/setAdmin` | 设置管理员（管理员） |

### 群组模块

| 接口 | 路由 | 说明 |
|------|------|------|
| CreateGroup | `/group/createGroup` | 创建群聊 |
| GetGroupInfo | `/group/getGroupInfo` | 获取群聊信息 |
| UpdateGroupInfo | `/group/updateGroupInfo` | 更新群聊信息 |
| LoadMyGroup | `/group/loadMyGroup` | 获取我的群聊 |
| CheckGroupAddMode | `/group/checkGroupAddMode` | 检查加群方式 |
| EnterGroupDirectly | `/group/enterGroupDirectly` | 直接加群 |
| GetGroupMemberList | `/group/getGroupMemberList` | 获取群成员列表 |
| RemoveGroupMembers | `/group/removeGroupMembers` | 移除群成员 |
| LeaveGroup | `/group/leaveGroup` | 退出群聊 |
| DismissGroup | `/group/dismissGroup` | 解散群聊 |
| GetGroupInfoList | `/group/getGroupInfoList` | 获取群聊列表（管理员） |
| DeleteGroups | `/group/deleteGroups` | 删除群聊（管理员） |
| SetGroupsStatus | `/group/setGroupsStatus` | 设置群聊状态（管理员） |

### 会话模块

| 接口 | 路由 | 说明 |
|------|------|------|
| OpenSession | `/session/openSession` | 打开/创建会话 |
| GetUserSessionList | `/session/getUserSessionList` | 获取用户会话列表 |
| GetGroupSessionList | `/session/getGroupSessionList` | 获取群聊会话列表 |
| DeleteSession | `/session/deleteSession` | 删除会话 |
| CheckOpenSessionAllowed | `/session/checkOpenSessionAllowed` | 检查是否允许发起会话 |

### 联系人模块

| 接口 | 路由 | 说明 |
|------|------|------|
| GetUserList | `/contact/getUserList` | 获取用户列表 |
| GetContactInfo | `/contact/getContactInfo` | 获取联系人信息 |
| LoadMyJoinedGroup | `/contact/loadMyJoinedGroup` | 获取我加入的群聊 |
| ApplyContact | `/contact/applyContact` | 申请添加联系人 |
| GetNewContactList | `/contact/getNewContactList` | 获取好友申请列表 |
| PassContactApply | `/contact/passContactApply` | 通过好友申请 |
| RefuseContactApply | `/contact/refuseContactApply` | 拒绝好友申请 |
| DeleteContact | `/contact/deleteContact` | 删除联系人 |
| BlackContact | `/contact/blackContact` | 拉黑联系人 |
| CancelBlackContact | `/contact/cancelBlackContact` | 解除拉黑 |
| BlackApply | `/contact/blackApply` | 拉黑申请人 |
| GetAddGroupList | `/contact/getAddGroupList` | 获取加群申请列表 |

### 消息模块

| 接口 | 路由 | 说明 |
|------|------|------|
| GetMessageList | `/message/getMessageList` | 获取私聊消息列表 |
| GetGroupMessageList | `/message/getGroupMessageList` | 获取群聊消息列表 |
| UploadAvatar | `/message/uploadAvatar` | 上传头像 |
| UploadFile | `/message/uploadFile` | 上传文件 |

### 聊天室模块

| 接口 | 路由 | 说明 |
|------|------|------|
| GetCurContactListInChatRoom | `/chatroom/getCurContactListInChatRoom` | 获取聊天室在线成员 |

## 数据库表结构

| 表名 | 说明 | 主要字段 |
|------|------|----------|
| user_info | 用户信息 | uuid, nickname, email, avatar, status, is_admin, gender, signature, birthday |
| group_info | 群聊信息 | uuid, name, avatar, notice, member_cnt, owner_id, add_mode, status |
| session | 会话 | uuid, send_id, receive_id, avatar, nickname, type |
| message | 消息 | uuid, session_id, send_id, receive_id, content, file_url, file_name, file_size, file_type, type, status |
| user_contact | 用户联系人 | uuid, owner_id, contact_id, contact_name, contact_avatar, contact_type, status |
| contact_apply | 联系人申请 | uuid, user_one_id, user_two_id, message, status |

所有表使用 GORM 软删除（`deleted_at` 字段）。

## Uuid 命名规则

| 前缀 | 类型 | 示例 |
|------|------|------|
| U | 用户 UserInfo | U2026042459746164431 |
| G | 群聊 GroupInfo | G2026042459746164431 |
| S | 会话 Session | S2026042459746164431 |
| A | 申请 ContactApply | A2026042459746164431 |
| M | 消息 Message | M2026042459746164431 |

## 统一响应格式

```json
{
    "code": 200,
    "message": "xxx",
    "data": {}
}
```

- `code=200`：成功
- `code=400`：业务错误（如密码错误、用户已存在）
- `code=500`：系统错误
- HTTP 401：未认证（token 缺失/过期/无效）
- HTTP 403：已认证但无权限（如普通用户访问管理员接口）

## 前端路由

| 路径 | 页面 | 权限 |
|------|------|------|
| `/login` | 登录 | 公开 |
| `/register` | 注册 | 公开 |
| `/chat` | 聊天主页面 | 登录用户 |
| `/chat/:id` | 指定会话聊天 | 登录用户 |
| `/chat/contactlist` | 联系人管理 | 登录用户 |
| `/manager` | 管理员后台 | 管理员 |

## 功能特性

- 用户注册/登录（邮箱+密码、邮箱+验证码）
- JWT 认证与三级权限（公开 / 认证用户 / 管理员）
- 一对一私聊、群聊
- AI 助手（私聊 Agent + 群聊 @ 触发 Agent）
- 好友申请/通过/拒绝/拉黑
- 群组创建/加入/退出/解散
- 群成员管理、加群审核
- 文件/图片/头像上传
- WebRTC 音视频通话
- WebSocket 实时消息推送
- Channel / Kafka 双消息模式（单机↔分布式可切换）
- 管理员后台（用户管理、群组管理）
- Redis 缓存旁路（用户信息、群信息、消息列表、会话列表）
- Zap 双输出结构化日志

## License

MIT
