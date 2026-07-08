# GoChat

[中文](README.md) | [English](README.en.md)

A full-stack instant messaging application built with Go, React, and TypeScript, integrated with a CloudWeGo Eino AI assistant.

## 🧰 Tech Stack

### 🖥️ Backend

| Technology | Version | Description |
|------------|---------|-------------|
| Go | 1.26.1+ | Backend language |
| Gin | 1.12.0 | Web framework |
| GORM | 1.31.1 | ORM framework |
| MySQL | 8.0+ | Database |
| Redis | 6.0+ | Cache service |
| Zap | 1.27.1 | Logging framework |
| Gorilla WebSocket | 1.5.3 | WebSocket support |
| golang-jwt/v5 | 5.3.1 | JWT authentication |
| CloudWeGo Eino | 0.9.12 | AI Agent framework |
| segmentio/kafka-go | 0.4.51 | Message queue (optional) |
| QQ SMTP | - | Email verification codes |

### 🎨 Frontend

| Technology | Version | Description |
|------------|---------|-------------|
| React | 18 | Frontend framework |
| TypeScript | 5.x | Type safety |
| Vite | 5.x | Build tool |
| Zustand | 5.x | State management |
| React Router | 6.x | Routing |
| Ant Design | 5.x | UI component library |
| Axios | 1.x | HTTP client |

## 🗂️ Project Structure

```
gochat/
├── cmd/gochat/               # Entry point
├── api/v1/                   # Controller layer
├── internal/
│   ├── agent/                # AI Agent service (Eino + Mock implementations)
│   ├── config/               # Configuration loading (TOML)
│   ├── dao/                  # Database connection
│   ├── dto/
│   │   ├── request/          # Request structs
│   │   └── respond/          # Response structs
│   ├── middleware/           # JWT authentication + admin authorization middleware
│   ├── model/                # Database models (6 tables)
│   ├── service/
│   │   ├── gorm/             # Service layer (business logic)
│   │   ├── redis/            # Redis cache service
│   │   ├── email/            # Email verification code service
│   │   ├── chat/             # WebSocket chat service
│   │   └── kafka/            # Kafka message service (optional)
│   └── https_server/         # HTTP server and route registration
├── pkg/
│   ├── constants/            # Constants
│   ├── enum/                 # Enumerations
│   ├── jwt/                  # JWT generation and parsing
│   ├── util/                 # Utility functions (random generation, etc.)
│   └── zlog/                 # Zap logging wrapper
├── docs/
│   ├── lessons/              # Teaching documents (26 articles, architecture → JWT full flow)
│   └── build-log/            # WeChat-like Go development narrative documents
├── configs/                  # Configuration files
├── static/                   # Static resources (avatars/files)
├── logs/                     # Log files
└── frontend/                 # React frontend project
```

## 🏗️ Three-Layer Architecture

```
Request → Controller → Service → DAO → Database
         ↑ JsonBack ←───────────────┘
```

Development order: Model → DAO → Service → Controller → Route registration

---

## 🤖 AI Assistant

The project integrates an AI assistant based on CloudWeGo Eino. It exists in the system as the user `AI助手`.

- **Private Chat Agent**: When a user sends a private message to `AI助手`, the system reads the latest N historical messages as context, generates a reply, writes it to the message table, and pushes it through WebSocket.
- **Group Chat Agent**: Triggered when a group message contains `@AI助手`, `@agent`, or the `/ai` prefix. The reply mentions the asker with `@username`. The Agent does not speak proactively when not triggered.
- **Provider Switching**: Switch between `mock` (local debugging without keys) and `openai` (real LLM) via `[llmConfig]` in the configuration file or environment variables. Any OpenAI-compatible API is supported.
- **Timeout and Fallback**: Agent calls time out after 25 seconds and return a friendly message on failure. If Eino initialization fails, the system automatically falls back to Mock.
- **Security**: Reuses the authenticated userID from JWT and prevents client-side forgery. The private-chat Agent only reads the current user's history with the Agent, and the group-chat Agent only reads the current group's history.

### 🧠 Memory Mechanism

The AI assistant **does not maintain a separate memory store**. Memory is simply the chat records in the `message` table. Each time the Agent is triggered, it loads the latest N historical messages in real time and injects them into the LLM context. Messages outside the window are automatically "forgotten".

| Scenario | Context Source | Window Size | Role Mapping |
|----------|----------------|-------------|--------------|
| Private chat | Latest bidirectional messages between the user and Agent in the `message` table | Latest 10 messages | User message → `user`; Agent message → `assistant` |
| Group chat | Latest messages in the group from the `message` table | Latest 20 messages | Agent message → `assistant`; group member message → `user` with a `nickname: ` prefix |

Call flow: build `[system, historical messages..., current question]` → call the LLM → write the Agent reply to the `message` table → push through WebSocket + append to the Redis message cache (1-minute TTL, read acceleration only).

Design trade-offs:
- **Advantages**: Extremely simple implementation, memory is naturally consistent with frontend chat history (what the user can see is what the Agent "remembers"), and no extra storage is required.
- **Limitations**: Conversations outside the context window are completely lost, token cost grows linearly with long conversations, and there is no long-term memory such as summaries or user profiles.

Related constants are defined in [pkg/constants/agent.go](pkg/constants/agent.go): `AgentPrivateContextLen=10`, `AgentGroupContextLen=20`, `AgentMaxInputLen=4000`, and `AgentTimeoutSec=25`.

See the "AI Assistant Configuration" section in the Deployment Guide for detailed configuration.

---

## 🚀 Deployment Guide

### ✅ Requirements

- Go 1.26.1+
- Node.js 18.0+
- MySQL 8.0+
- Redis 6.0+
- Kafka 3.0+ (optional; not required for single-node deployment)

### 1. 📦 Clone the Project

```bash
git clone <repo-url>
cd gochat
```

### 2. 🗄️ Configure the Database

Create the MySQL database:

```sql
CREATE DATABASE gochat CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

> GORM `AutoMigrate` automatically creates tables on startup, so manual table creation is not required.

### 3. ⚙️ Modify the Configuration File

Copy the template and edit `configs/config_local.toml`:

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
password = "your_password"        # Change this to your MySQL password
databaseName = "gochat"

[logConfig]
logPath = "./logs/app.log"

[staticSrcConfig]
staticAvatarPath = "./static/avatars"
staticFilePath = "./static/files"

[redisConfig]
host = "127.0.0.1"
port = 6379
password = ""                     # Change this to your Redis password if needed
db = 0

[emailConfig]
smtpHost     = "smtp.qq.com"      # SMTP server (QQ Mail by default)
smtpPort     = 465                # SSL port
smtpUsername = "your@qq.com"      # Sender email
smtpPassword = "xxxxxxxxxxxxxxxx"  # QQ Mail authorization code (16 digits, not QQ password)
fromName     = "GoChat"           # Sender name

[jwtConfig]
secret = "change-this-to-a-random-secret"  # Must be changed in production
expireHours = 24

[kafkaConfig]
messageMode = "channel"           # Message mode: channel (single-node) or kafka (distributed)
hostPort = "127.0.0.1:9092"
loginTopic = "login"
chatTopic = "chat_message"
logoutTopic = "logout"
partition = 0
timeout = 1
```

**Notes:**
- Email verification uses QQ Mail SMTP. You need to enable SMTP in QQ Mail settings and obtain a 16-digit authorization code.
- When `messageMode` is set to `channel`, Kafka is not required. This is suitable for single-node deployment.
- When `messageMode` is set to `kafka`, Kafka must be installed and started. This is suitable for distributed deployment.
- `jwtConfig.secret` cannot use the default placeholder. The server validates it on startup and refuses to run if it is unchanged.
- In production, it is recommended to change `host` to `0.0.0.0`.

### 4. 🤖 AI Assistant Configuration (Optional)

The AI assistant uses the Mock implementation by default, so it works without any configuration. To connect a real large language model, use one of the following methods. **Environment variables have higher priority than the configuration file**.

**Method 1: Configuration file (recommended for persistence)**

Add the `[llmConfig]` section to `configs/config_local.toml`:

```toml
[llmConfig]
provider = "openai"                                                       # mock | openai
apiKey = "your-api-key"                                                   # API Key (compatible APIs may use an "AppID:Secret" format)
baseUrl = "https://maas-coding-api.cn-huabei-1.xf-yun.com/v2"             # OpenAI-compatible API endpoint
model = "xopdeepseekv4flash"                                              # Model name
```

**Method 2: Environment variables (temporary override)**

```bash
export LLM_PROVIDER=openai                          # mock | openai
export OPENAI_API_KEY=your-api-key                  # API Key
export OPENAI_BASE_URL=https://api.deepseek.com/    # OpenAI-compatible API endpoint (optional)
export LLM_MODEL=deepseek-chat                      # Model name
```

Supported providers:
- `mock` (default): Does not call an LLM locally and returns mock replies. Suitable for development and debugging.
- `openai`: Uses Eino to call OpenAI or any OpenAI-compatible API (DeepSeek / Qwen / Moonshot / iFlytek Spark MaaS, etc.). For iFlytek Spark MaaS, the `apiKey` is in the `AppID:Secret` format and should be passed as the whole Bearer token.

### 5. 📨 Install Kafka (Optional)

If distributed message mode is required, install Kafka:

```bash
# Download and extract Kafka
# Start ZooKeeper
bin/zookeeper-server-start.sh config/zookeeper.properties

# Start Kafka
bin/kafka-server-start.sh config/server.properties

# Create topics
bin/kafka-topics.sh --create --topic login --bootstrap-server localhost:9092
bin/kafka-topics.sh --create --topic chat_message --bootstrap-server localhost:9092
bin/kafka-topics.sh --create --topic logout --bootstrap-server localhost:9092
```

Then change `messageMode` in `config_local.toml` to `kafka`.

> Skip this step for single-node deployment and use the default `channel` mode.

### 6. 🖥️ Start the Backend

```bash
# Install Go dependencies
go mod download

# Start the service
go run cmd/gochat/main.go
```

The backend runs at `http://127.0.0.1:8000` by default.

#### 🏭 Production Build

```bash
go build -o gochat-server cmd/gochat/main.go
./gochat-server
```

### 7. 🎨 Start the Frontend

```bash
cd frontend

# Install dependencies
npm install

# Start in development mode (default port: 3000)
npm run dev
```

The frontend development server runs at `http://localhost:3000`. The default backend address is configured in `frontend/src/utils/constants.ts` as `http://127.0.0.1:8000`.

#### 🎨 Frontend Configuration

Edit `frontend/src/utils/constants.ts` to change the backend address:

```typescript
export const BACKEND_URL = 'http://127.0.0.1:8000'  // Backend address
export const WS_URL = 'ws://127.0.0.1:8000'         // WebSocket address
```

#### 🏭 Production Build

```bash
cd frontend
npm run build
```

The build output is located in `frontend/dist/`. The backend currently only serves uploaded static resources (`/static/avatars` and `/static/files`). In production, use Nginx or another static file server to serve `frontend/dist/`.

### 8. ✅ Verify Deployment

1. Visit `http://localhost:3000` to open the login page (development mode).
2. Register an account (email verification code required).
3. Log in and use the chat features.
4. Find `AI助手` in contacts and send a message to trigger the Agent reply.

---

## 🔌 API Endpoints

### 👤 User Module

| API | Route | Description |
|-----|-------|-------------|
| Login | `/login` | Email + password login |
| Register | `/register` | Registration |
| EmailLogin | `/user/emailLogin` | Email verification-code login |
| SendEmailCode | `/user/sendEmailCode` | Send email verification code |
| VerifyEmailCode | `/user/verifyEmailCode` | Verify email verification code |
| UpdateUserInfo | `/user/updateUserInfo` | Update user information |
| GetUserInfo | `/user/getUserInfo` | Get user information |
| GetUserInfoList | `/user/getUserInfoList` | Get user list |
| WsLogin | `/user/wsLogin` | WebSocket connection |
| WsLogout | `/user/wsLogout` | WebSocket logout |
| AbleUsers | `/user/ableUsers` | Enable users (admin) |
| DisableUsers | `/user/disableUsers` | Disable users (admin) |
| DeleteUsers | `/user/deleteUsers` | Delete users (admin) |
| SetAdmin | `/user/setAdmin` | Set administrator (admin) |

### 👥 Group Module

| API | Route | Description |
|-----|-------|-------------|
| CreateGroup | `/group/createGroup` | Create a group chat |
| GetGroupInfo | `/group/getGroupInfo` | Get group information |
| UpdateGroupInfo | `/group/updateGroupInfo` | Update group information |
| LoadMyGroup | `/group/loadMyGroup` | Get my group chats |
| CheckGroupAddMode | `/group/checkGroupAddMode` | Check group join mode |
| EnterGroupDirectly | `/group/enterGroupDirectly` | Join a group directly |
| GetGroupMemberList | `/group/getGroupMemberList` | Get group member list |
| RemoveGroupMembers | `/group/removeGroupMembers` | Remove group members |
| LeaveGroup | `/group/leaveGroup` | Leave a group chat |
| DismissGroup | `/group/dismissGroup` | Dismiss a group chat |
| GetGroupInfoList | `/group/getGroupInfoList` | Get group list (admin) |
| DeleteGroups | `/group/deleteGroups` | Delete groups (admin) |
| SetGroupsStatus | `/group/setGroupsStatus` | Set group status (admin) |

### 💬 Session Module

| API | Route | Description |
|-----|-------|-------------|
| OpenSession | `/session/openSession` | Open/create a session |
| GetUserSessionList | `/session/getUserSessionList` | Get user session list |
| GetGroupSessionList | `/session/getGroupSessionList` | Get group session list |
| DeleteSession | `/session/deleteSession` | Delete a session |
| CheckOpenSessionAllowed | `/session/checkOpenSessionAllowed` | Check whether opening a session is allowed |

### 🤝 Contact Module

| API | Route | Description |
|-----|-------|-------------|
| GetUserList | `/contact/getUserList` | Get user list |
| GetContactInfo | `/contact/getContactInfo` | Get contact information |
| LoadMyJoinedGroup | `/contact/loadMyJoinedGroup` | Get groups I have joined |
| ApplyContact | `/contact/applyContact` | Apply to add a contact |
| GetNewContactList | `/contact/getNewContactList` | Get friend request list |
| PassContactApply | `/contact/passContactApply` | Accept friend request |
| RefuseContactApply | `/contact/refuseContactApply` | Refuse friend request |
| DeleteContact | `/contact/deleteContact` | Delete contact |
| BlackContact | `/contact/blackContact` | Block contact |
| CancelBlackContact | `/contact/cancelBlackContact` | Unblock contact |
| BlackApply | `/contact/blackApply` | Block applicant |
| GetAddGroupList | `/contact/getAddGroupList` | Get group join request list |

### ✉️ Message Module

| API | Route | Description |
|-----|-------|-------------|
| GetMessageList | `/message/getMessageList` | Get private chat message list |
| GetGroupMessageList | `/message/getGroupMessageList` | Get group chat message list |
| UploadAvatar | `/message/uploadAvatar` | Upload avatar |
| UploadFile | `/message/uploadFile` | Upload file |

### 🏠 Chatroom Module

| API | Route | Description |
|-----|-------|-------------|
| GetCurContactListInChatRoom | `/chatroom/getCurContactListInChatRoom` | Get online members in the chatroom |

## 🗄️ Database Tables

| Table | Description | Main Fields |
|-------|-------------|-------------|
| user_info | User information | uuid, nickname, email, avatar, status, is_admin, gender, signature, birthday, last_online_at, last_offline_at |
| group_info | Group chat information | uuid, name, avatar, notice, members, member_cnt, owner_id, add_mode, status |
| session | Session | uuid, send_id, receive_id, receive_name, avatar, last_message, last_message_at |
| message | Message | uuid, session_id, send_id, send_name, send_avatar, receive_id, content, url, file_name, file_size, file_type, type, status, av_data |
| user_contact | User contacts | user_id, contact_id, contact_type, status, update_at |
| contact_apply | Contact application | uuid, user_id, contact_id, contact_type, message, status, last_apply_at |

All core tables except `message` use GORM soft delete (`deleted_at` field).

## 🆔 UUID Naming Rules

| Prefix | Type | Example |
|--------|------|---------|
| U | UserInfo | U2026042459746164431 |
| G | GroupInfo | G2026042459746164431 |
| S | Session | S2026042459746164431 |
| A | ContactApply | A2026042459746164431 |
| M | Message | M2026042459746164431 |

## 📦 Unified Response Format

```json
{
    "code": 200,
    "message": "xxx",
    "data": {}
}
```

- `code=200`: Success
- `code=400`: Business error (for example, wrong password or user already exists)
- `code=500`: System error
- HTTP 401: Unauthenticated (token missing/expired/invalid)
- HTTP 403: Authenticated but unauthorized (for example, a regular user accessing an admin endpoint)

## 🧭 Frontend Routes

| Path | Page | Permission |
|------|------|------------|
| `/login` | Login | Public |
| `/register` | Register | Public |
| `/chat` | Main chat page | Logged-in users |
| `/chat/:id` | Specific session chat | Logged-in users |
| `/chat/contactlist` | Contact management | Logged-in users |
| `/manager` | Admin dashboard | Administrators |

## ✨ Features

- User registration/login (email + password, email + verification code)
- JWT authentication and three-level permissions (public / authenticated user / administrator)
- One-to-one private chat and group chat
- AI assistant (private-chat Agent + group-chat @ trigger Agent)
- Friend request / accept / refuse / block
- Group creation / join / leave / dismiss
- Group member management and group join approval
- File/image/avatar upload
- WebRTC audio/video calls
- WebSocket real-time message push
- Channel / Kafka dual message modes (switchable between single-node and distributed deployment)
- Admin dashboard (user management, group management)
- Redis cache-aside strategy (user information, group information, message lists, session lists)
- Zap dual-output structured logging

## 📄 License

MIT
