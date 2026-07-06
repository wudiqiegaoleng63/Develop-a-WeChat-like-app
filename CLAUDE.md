# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoChat - A full-stack instant messaging application. Go backend (Gin + GORM) with React + TypeScript frontend.

## Build & Run Commands

```bash
# Backend: Build entire project
go build ./...

# Backend: Run server (default port 8000)
go run cmd/gochat/main.go

# Backend: Run tests (if any)
go test ./...

# Frontend: Install dependencies
cd frontend && npm install

# Frontend: Dev server (port 5173)
cd frontend && npm run dev

# Frontend: Production build
cd frontend && npm run build
```

## Backend Architecture: Three-Layer Pattern

**Development order**: Model → DAO → Service → Controller → Route Registration

```
Request → Controller → Service → DAO → Database
         ↑ JsonBack ←───────────────┘
```

### Backend Layer Structure

| Layer | Directory | Purpose |
|-------|-----------|---------|
| Model | `internal/model/` | Database table structs (UserInfo, GroupInfo, Session, Message, UserContact, ContactApply) |
| DAO | `internal/dao/gorm.go` | GORM database connection, global `dao.GormDB` instance |
| Service | `internal/service/gorm/` | Business logic implementations |
| Service | `internal/service/email/` | Email verification code service (QQ SMTP) |
| Service | `internal/service/redis/` | Redis caching for verification codes |
| Controller | `api/v1/` | HTTP handlers, route definitions |
| DTO | `internal/dto/request/`, `internal/dto/respond/` | Request/Response structs |
| Config | `internal/config/config.go` | TOML config loader |
| Utils | `pkg/` | Constants, enums, logging (zlog), random utilities |

### Key Patterns

**Init-based initialization**: Packages use `init()` for auto-setup. Import with `_` to trigger:
```go
import (
    _ "gochat/internal/config"    // Loads config from configs/config_local.toml
    _ "gochat/internal/dao"       // Connects MySQL, AutoMigrate tables
    _ "gochat/internal/service/redis"  // Connects Redis
    _ "gochat/internal/service/email"  // Email service ready
    "gochat/internal/https_server"      // Route registration
)
```

**Unified response**: Use `JsonBack(c, message, ret, data)` in controllers:
- `ret=0`: Success, code=200
- `ret=-2`: Business error, code=400 (e.g., wrong password)
- `ret=-1`: System error, code=500

**Uuid generation**: `"U" + random.GetNowAndLenRandomString(11)` = 20 chars (U + date + random)

## Frontend Architecture (React + TypeScript)

### Tech Stack
- **Framework**: React 18 + TypeScript + Vite
- **State Management**: Zustand (4 stores)
- **Routing**: React Router v6
- **HTTP Client**: Axios
- **UI Library**: Ant Design (Modal.confirm, ConfigProvider)
- **WebSocket**: Native WebSocket with reconnection logic

### Directory Structure (`frontend/src/`)

| Directory | Purpose |
|-----------|---------|
| `api/` | Axios-based API client functions (auth, user, contact, session, message, group, chatroom) |
| `components/chat/` | Chat UI: ChatHeader, ChatInput, MessageList, MessageBubble, EmojiPicker, ContactInfoModal, AVCallModal |
| `components/contact/` | ContactList — friend search, friend requests, group discovery |
| `components/group/` | Group modals: CreateGroupModal, EditGroupInfoModal, RemoveGroupMembersModal, GroupJoinRequestsModal |
| `components/user/` | UserProfileModal — edit profile (avatar, nickname, gender, signature, birthday) |
| `components/layout/` | Layout shells: Sidebar, SessionList, ChatWindow |
| `hooks/` | Custom hooks: useWebSocket, useAutoScroll |
| `pages/` | Route-level pages: LoginPage, RegisterPage, ChatPage, ManagerPage |
| `services/` | WebSocket connection management with auto-reconnect |
| `stores/` | Zustand stores: useAuthStore, useChatStore, useContactStore, useUIStore |
| `styles/` | Global CSS with WeChat-themed CSS custom properties |
| `types/` | TypeScript types: api, message, session, user, group |
| `utils/` | Utilities: constants (BACKEND_URL), avatar normalization, formatting, toast |

### Zustand Stores

| Store | Key State | Key Actions |
|-------|-----------|-------------|
| `useAuthStore` | userInfo | login, register, logout, updateProfile, uploadAndSetAvatar |
| `useChatStore` | userSessionList, groupSessionList, activeContactId, messageList | fetchSessionLists, setActiveChat, sendMessage, addIncomingMessage |
| `useContactStore` | searchUsers, friendRequests, myGroups, loading | fetchSearchUsers, fetchFriendRequests, fetchMyGroups, applyFriend, passRequest, refuseRequest |
| `useUIStore` | sidebarCollapsed, activeTab, searchQuery | toggleSidebar, setActiveTab, setSearchQuery |

### Routes

| Path | Component | Guard |
|------|-----------|-------|
| `/login` | LoginPage | Public |
| `/register` | RegisterPage | Public |
| `/chat` | ChatPage (SessionList + ChatWindow) | ProtectedRoute |
| `/chat/:id` | ChatPage (specific chat) | ProtectedRoute |
| `/chat/contactlist` | ChatPage → ContactList | ProtectedRoute |
| `/manager` | ManagerPage | AdminRoute (isAdmin === 1) |

### API Modules

| Module | File | Functions |
|--------|------|-----------|
| Auth | `api/auth.ts` | login, register, emailLogin, sendEmailCode |
| User | `api/user.ts` | verifyEmailCode, updateUserInfo, getUserInfo, getUserInfoList, wsLogout, ableUsers, disableUsers, deleteUsers, setAdmin |
| Contact | `api/contact.ts` | getUserList, getContactInfo, applyContact, getNewContactList, passContactApply, refuseContactApply, deleteContact, blackContact, cancelBlackContact, blackApply, getAddGroupList, loadMyJoinedGroup |
| Session | `api/session.ts` | openSession, getUserSessionList, getGroupSessionList, deleteSession, checkOpenSessionAllowed |
| Message | `api/message.ts` | getMessageList, getGroupMessageList, uploadFile, uploadAvatar |
| Group | `api/group.ts` | createGroup, checkGroupAddMode, enterGroupDirectly, getGroupInfo, updateGroupInfo, removeGroupMembers, loadMyGroup, getGroupMemberList, leaveGroup, dismissGroup, getGroupInfoList, deleteGroups, setGroupsStatus |
| ChatRoom | `api/chatroom.ts` | getCurContactListInChatRoom |

### Key Type Definitions

- **UserInfo**: `{ uuid, nickname, email?, avatar?, status, isAdmin, gender?, signature?, birthday? }`
- **GroupInfo**: `{ uuid, name, avatar?, notice?, member_cnt?, owner_id, add_mode?, status, is_deleted? }`
- **ChatMessage**: full message with uuid, session, sender/receiver ids, content, file metadata (url, file_name, file_size, file_type), status
- **MessageType**: enum `TEXT=0, VOICE=1, FILE=2, AV=3`
- **ContactInfo**: contact profile with id, name, avatar, phone, email, gender, signature, notice, members, owner, add_mode

### Frontend Conventions

- **Modal pattern**: Use `info-modal-overlay` / `info-modal` CSS classes for custom modals; Ant Design `Modal.confirm` for dangerous confirmations
- **Avatar URLs**: Always pass through `normalizeAvatarUrl()` from `utils/avatar.ts`
- **Toast notifications**: Use `showToast(message, type)` from `utils/toast.ts`
- **API response**: Check `res.code === 200` for success
- **Image messages**: When `file_type` starts with `image/`, render `<img>` instead of file download link

## Configuration

Config file: `configs/config_local.toml`

Required sections: `[mainConfig]`, `[mysqlConfig]`, `[redisConfig]`, `[emailConfig]`, `[logConfig]`, `[staticSrcConfig]`

Email uses QQ SMTP (smtp.qq.com:465 with TLS), not SMS. Requires 16-char authorization code (not QQ password).

## Important File Locations

- Backend entry point: `cmd/gochat/main.go`
- Backend routes: `internal/https_server/https_server.go` - `registerRoutes()`
- JsonBack: `api/v1/controller.go` (same package as controllers, call directly)
- Verification code Redis key: `email_code_` + email address, 5 min TTL
- Frontend entry: `frontend/src/main.tsx`
- Frontend routes: `frontend/src/App.tsx`
- Frontend styles: `frontend/src/styles/global.css`
- Frontend constants: `frontend/src/utils/constants.ts` (BACKEND_URL)

## Teaching Documentation

Detailed step-by-step guides in `lessons/` directory (26 documents, numbered 01-26). Follow document order for implementation.

**IMPORTANT**: When modifying any source code, you MUST同步更新 corresponding teaching documents in `lessons/`. This ensures teaching materials stay accurate with actual implementation.

Examples:
- Modify `pkg/util/random/random_int.go` → Update `07-Redis邮箱验证码详细步骤.md`
- Modify `internal/service/gorm/user_info_service.go` → Update `08-登录注册实现.md`
- Modify `api/v1/controller.go` → Update `09-HTTP服务器与路由.md`

**CRITICAL RULE**: 教学文档内容必须严格按照源码实际实现来写，**不能简化、不能省略**。

必须保持一致的内容包括：
- 完整的import导入路径和导入包
- 完整的函数实现（包括for循环、日志输出、错误处理等）
- 完整的结构体定义（包括所有字段、gorm标签、json标签）
- 源码中的注释说明
- 源码中的日志输出（zlog.Info、log.Println等）
- Redis库版本：`github.com/go-redis/redis/v8`（不是v9）
- 变量命名：`redisClient`（小写，不是RedisClient大写）

参考源码位置：`C:\Users\li\Desktop\cankao go\KamaChat\`

## Import Path Corrections

Common mistakes to avoid:
- `internal/request` → should be `internal/dto/request`
- `internal/constants` → should be `pkg/constants`
- `pkg/util/zlog` → should be `pkg/zlog`
- `internal/enum/` → should be `pkg/enum/`

## Database Models (6 tables)

UserInfo, GroupInfo, Session, Message, UserContact, ContactApply - all use `gorm.DeletedAt` for soft delete.

GORM automatically adds `WHERE deleted_at IS NULL` to queries. Use `Unscoped()` to include soft-deleted records.

## Known JSON Tag Inconsistencies

Backend DTO JSON tags are **camelCase** for user-facing fields: `isAdmin` (not `is_admin`), `isDeleted` (not `is_deleted`). The login API and user list API both return `isAdmin`. Frontend types must use camelCase field names matching the JSON tags.