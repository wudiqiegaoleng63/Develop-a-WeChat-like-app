# KamaChat Server

基于Go语言开发的即时通讯后端服务，采用Gin框架和三层架构设计。

## 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| Go | 1.26.1 | 后端语言 |
| Gin | 1.12.0 | Web框架 |
| GORM | 1.31.1 | ORM框架 |
| MySQL | - | 数据库 |
| Redis | v8 | 缓存服务 |
| Zap | 1.27.1 | 日志框架 |
| QQ SMTP | - | 邮箱验证码 |

## 项目结构

```
kama-chat-server/
├── cmd/kama-chat-server/     # 入口文件
├── api/v1/                   # Controller层
├── internal/
│   ├── config/               # 配置加载
│   ├── dao/                  # 数据库连接
│   ├── dto/
│   │   ├── request/          # 请求结构体
│   │   └ respond/            # 响应结构体
│   ├── model/                # 数据库模型
│   ├── service/
│   │   ├── gorm/             # Service层（业务逻辑）
│   │   ├── redis/            # Redis服务
│   │   ├── email/            # 邮箱服务
│   ├── https_server/         # HTTP服务器与路由
├── pkg/
│   ├── constants/            # 常量定义
│   ├── enum/                 # 枚举定义
│   ├── util/                 # 工具函数
│   ├── zlog/                 # 日志封装
├── configs/                  # 配置文件
├── static/                   # 静态资源
├── logs/                     # 日志文件
└── a teacher/                # 教学文档
```

## 三层架构

```
Request → Controller → Service → DAO → Database
         ↑ JsonBack ←───────────────┘
```

开发顺序：Model → DAO → Service → Controller → 路由注册

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置文件

修改 `configs/config_local.toml`：

```toml
[mainConfig]
appName = "kama-chat"
host = "127.0.0.1"
port = 8000

[mysqlConfig]
host = "127.0.0.1"
port = 3306
user = "root"
password = "your_password"
databaseName = "kama_chat"

[redisConfig]
host = "127.0.0.1"
port = 6379
password = ""
db = 0

[emailConfig]
smtpHost     = "smtp.qq.com"
smtpPort     = 465
smtpUsername = "your_email@qq.com"
smtpPassword = "your_auth_code"  # QQ邮箱授权码
fromName     = "KamaChat"
```

### 3. 创建数据库

```sql
CREATE DATABASE kama_chat CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 4. 启动服务

```bash
go run cmd/kama-chat-server/main.go
```

服务启动后访问：`http://127.0.0.1:8000`

### 5. 构建项目

```bash
go build ./...
```

## 已完成的接口

### 用户模块

| 接口 | 路由 | 说明 |
|------|------|------|
| Login | `/login` | 邮箱+密码登录 |
| Register | `/register` | 注册 |
| EmailLogin | `/user/emailLogin` | 邮箱+验证码登录 |
| SendEmailCode | `/user/sendEmailCode` | 发送邮箱验证码 |
| VerifyEmailCode | `/user/verifyEmailCode` | 验证邮箱验证码 |
| UpdateUserInfo | `/user/updateUserInfo` | 更新用户信息 |
| GetUserInfoList | `/user/getUserInfoList` | 获取用户列表 |

### 群组模块

| 接口 | 路由 | 说明 |
|------|------|------|
| CreateGroup | `/group/createGroup` | 创建群聊 |
| LoadMyGroup | `/group/loadMyGroup` | 获取我的群聊 |
| CheckGroupAddMode | `/group/checkGroupAddMode` | 检查加群方式 |
| EnterGroupDirectly | `/group/enterGroupDirectly` | 直接加群 |
| LeaveGroup | `/group/leaveGroup` | 退出群聊 |
| DismissGroup | `/group/dismissGroup` | 解散群聊 |
| GetGroupInfo | `/group/getGroupInfo` | 获取群聊信息 |
| GetGroupInfoList | `/group/getGroupInfoList` | 获取群聊列表 |
| DeleteGroups | `/group/deleteGroups` | 删除群聊 |
| SetGroupsStatus | `/group/setGroupsStatus` | 设置群聊状态 |
| UpdateGroupInfo | `/group/updateGroupInfo` | 更新群聊信息 |
| GetGroupMemberList | `/group/getGroupMemberList` | 获取群成员列表 |
| RemoveGroupMembers | `/group/removeGroupMembers` | 移除群成员 |

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
| GetUserList | `/contact/getUserList` | 获取好友列表 |
| LoadMyJoinedGroup | `/contact/loadMyJoinedGroup` | 获取我加入的群聊 |
| GetContactInfo | `/contact/getContactInfo` | 获取联系人信息 |
| DeleteContact | `/contact/deleteContact` | 删除联系人 |
| BlackContact | `/contact/blackContact` | 拉黑联系人 |
| CancelBlackContact | `/contact/cancelBlackContact` | 解除拉黑 |

## 数据库表结构

| 表名 | 说明 |
|------|------|
| user_info | 用户信息表 |
| group_info | 群聊信息表 |
| session | 会话表 |
| message | 消息表 |
| user_contact | 用户联系人表 |
| contact_apply | 联系人申请表 |

## Uuid命名规则

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
    "code": 200,      // 200成功, 400业务错误, 500系统错误
    "message": "xxx",
    "data": {}
}
```

返回值约定：
- `ret=0` → code=200, 成功
- `ret=-2` → code=400, 业务错误
- `ret=-1` → code=500, 系统错误

## 教学文档

详细的教学文档位于 `a teacher/` 目录，包含22个文档，按顺序学习：

1. 项目概述与架构设计
2. Go环境配置
3. 项目初始化
4. TOML配置文件
5. 数据库连接
6. Zap日志集成
7. Redis邮箱验证码详细步骤
8. 登录注册实现
9. HTTP服务器与路由
10. 用户信息接口
11. 群组管理接口-上
12. 群组管理接口-下
13. 群组成员管理接口
14. 会话管理
15. 联系人管理
16. 联系人申请流程
17. 消息与文件上传
18. 管理员群组接口
19. WebSocket连接建立
20. 消息发送与接收
21. 管理员用户管理接口
22. 管理员消息管理接口
23. 接口汇总与测试

## 前端项目

前端项目位于 `web/chat-server/` 目录。

### 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | 3.2.13 | 前端框架 |
| Vue Router | 4.0.3 | 路由管理 |
| Vuex | 4.0.0 | 状态管理 |
| Element Plus | 2.9.0 | UI组件库 |
| Axios | 1.7.9 | HTTP请求 |
| Vue CLI | 5.0.0 | 构建工具 |

### 项目结构

```
web/chat-server/
├── src/
│   ├── assets/           # 静态资源
│   │   ├── css/          # 样式文件
│   │   ├── img/          # 图片资源
│   │   └ js/             # 工具函数
│   ├── components/       # 公共组件
│   │   ├── Modal.vue         # 弹窗组件
│   │   ├── SmallModal.vue    # 小弹窗组件
│   │   ├── ContactListModal.vue    # 联系人列表弹窗
│   │   ├── DeleteUserModal.vue     # 删除用户弹窗
│   │   ├── DeleteGroupModal.vue    # 删除群聊弹窗
│   │   ├── DisableUserModal.vue    # 禁用用户弹窗
│   │   ├── DisableGroupModal.vue   # 禁用群聊弹窗
│   │   ├── SetAdminModal.vue       # 设置管理员弹窗
│   │   └ NavigationModal.vue       # 导航弹窗
│   ├── views/            # 页面视图
│   │   ├── access/       # 访问页面
│   │   │   ├── Login.vue       # 登录页
│   │   │   ├── Register.vue    # 注册页
│   │   │   ├── EmailLogin.vue  # 邮箱登录页
│   │   ├── chat/         # 聊天页面
│   │   │   ├── user/OwnInfo.vue        # 个人信息页
│   │   │   ├── contact/ContactList.vue # 联系人列表页
│   │   │   ├── contact/ContactChat.vue # 联系人聊天页
│   │   │   ├── session/SessionList.vue # 会话列表页
│   │   │   ├── manager/Manager.vue     # 管理员页面
│   ├── router/           # 路由配置
│   ├── store/            # Vuex状态管理
│   ├── App.vue           # 根组件
│   └ main.js             # 入口文件
├── package.json          # 依赖配置
```

### 路由配置

| 路由 | 名称 | 说明 |
|------|------|------|
| `/login` | Login | 登录页 |
| `/emailLogin` | EmailLogin | 邮箱验证码登录 |
| `/register` | Register | 注册页 |
| `/chat/owninfo` | OwnInfo | 个人信息 |
| `/chat/contactlist` | ContactList | 联系人列表 |
| `/chat/:id` | ContactChat | 联系人聊天 |
| `/chat/sessionList` | SessionList | 会话列表 |
| `/manager` | Manager | 管理员后台 |

### 启动前端

```bash
cd web/chat-server

# 安装依赖
npm install

# 开发模式启动
npm run serve

# 生产构建
npm run build
```

前端默认运行在 `http://localhost:8080`，会自动代理到后端 `http://127.0.0.1:8000`。

## 开发环境

- Go 1.26.1+
- MySQL 8.0+
- Redis 6.0+
- Node.js 16.0+
- IDE: VS Code / GoLand

## License

MIT