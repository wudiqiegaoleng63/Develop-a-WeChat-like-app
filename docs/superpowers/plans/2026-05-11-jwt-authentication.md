# JWT 认证实现计划（修正版）

> **目标：** 为所有 API 接口添加 JWT 认证，替代当前只要知道 UUID 就能调用任意接口的无鉴权架构。

> **架构：** 后端在登录/注册时生成 JWT（包含用户 UUID 和管理员标识），通过 Gin 中间件在每个受保护路由上校验 token 并将用户身份注入 `gin.Context`。Controller 层从 Context 获取 token 身份，与请求体中的 owner_id/uuid/send_id 比对，确保用户只能操作自己的数据。管理员接口额外要求 isAdmin == 1。前端存储 token，通过 `Authorization: Bearer <token>` 请求头携带，WebSocket 连接通过 `?token=<jwt>` 参数传递。

> **技术栈：** `github.com/golang-jwt/jwt/v5`、Gin 中间件、Axios 拦截器

---

## 文件清单

| 操作 | 文件 | 说明 |
|------|------|------|
| 新建 | `pkg/jwt/jwt.go` | JWT 生成、解析、Claims 结构体 |
| 新建 | `internal/middleware/auth.go` | JWT 认证中间件 + 管理员权限中间件 |
| 修改 | `internal/https_server/https_server.go` | CORS 放行 Authorization + 路由分三组 |
| 修改 | `internal/dto/respond/user_info_respond.go` | 登录/注册响应结构体增加 Token 字段 |
| 修改 | `internal/service/gorm/user_info_service.go` | Login、Register、EmailLogin 中生成 token |
| 修改 | `api/v1/user_info_controller.go` | 从 Context 获取 token uuid，校验与请求体一致 |
| 修改 | `api/v1/user_contact_controller.go` | 同上 |
| 修改 | `api/v1/session_controller.go` | 同上 |
| 修改 | `api/v1/group_info_controller.go` | 同上 |
| 修改 | `api/v1/message_controller.go` | 同上 |
| 修改 | `api/v1/chatroom_controller.go` | 同上 |
| 修改 | `api/v1/ws_controller.go` | WebSocket 连接改为 token 验证 |
| 修改 | `go.mod` | 添加 `github.com/golang-jwt/jwt/v5` 依赖 |
| 修改 | `configs/config_local.toml` | 添加 `[jwtConfig]` 配置段 |
| 修改 | `internal/config/config.go` | 添加 JWT 配置结构体 |
| 修改 | `react_web/src/types/api.ts` | 添加 AuthResponse 类型 |
| 修改 | `react_web/src/api/auth.ts` | 返回类型改为 AuthResponse |
| 修改 | `react_web/src/stores/useAuthStore.ts` | 存储 token，登录/注册时保存 |
| 修改 | `react_web/src/api/axios.ts` | 请求拦截器添加 Bearer token，响应拦截器处理 401 |
| 修改 | `react_web/src/services/websocket.ts` | WebSocket 连接 URL 使用 token 参数 |

---

## 任务 1：添加 JWT 依赖和配置

**涉及文件：**
- 修改: `go.mod`
- 修改: `configs/config_local.toml`
- 修改: `internal/config/config.go`

- [ ] **步骤 1：安装 golang-jwt/jwt/v5**

```bash
cd c:/Users/li/Desktop/kama-chat-server
go get github.com/golang-jwt/jwt/v5
```

- [ ] **步骤 2：在配置文件中添加 JWT 配置**

在 `configs/config_local.toml` 末尾追加：

```toml
[jwtConfig]
secret = "gochat-jwt-secret-key-change-in-production"
expireHours = 24
```

- [ ] **步骤 3：在 config.go 中添加 JWT 配置结构体**

在 `internal/config/config.go` 中添加结构体（跟现有的 MysqlConfig、RedisConfig 同级）：

```go
type JwtConfig struct {
	Secret      string `toml:"secret"`
	ExpireHours int    `toml:"expireHours"`
}
```

在 `Config` 结构体中添加字段：`JwtConfig JwtConfig \`toml:"jwtConfig"\``

- [ ] **步骤 4：验证编译**

```bash
go build ./...
```

- [ ] **步骤 5：提交**

```bash
git add go.mod go.sum configs/config_local.toml internal/config/config.go
git commit -m "feat: 添加 JWT 依赖和配置"
```

---

## 任务 2：创建 JWT 工具包

**涉及文件：**
- 新建: `pkg/jwt/jwt.go`

- [ ] **步骤 1：创建 `pkg/jwt/jwt.go`**

```go
package jwt

import (
	"errors"
	"kama-chat-server/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 载荷
type Claims struct {
	Uuid    string `json:"uuid"`
	IsAdmin int8   `json:"isAdmin"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT token
func GenerateToken(uuid string, isAdmin int8) (string, error) {
	conf := config.GetConfig()
	expireHours := conf.JwtConfig.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}

	claims := Claims{
		Uuid:    uuid,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gochat",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(conf.JwtConfig.Secret))
}

// ParseToken 解析 JWT token
func ParseToken(tokenString string) (*Claims, error) {
	conf := config.GetConfig()
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 显式校验签名算法，防止算法混淆攻击
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(conf.JwtConfig.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
```

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

- [ ] **步骤 3：提交**

```bash
git add pkg/jwt/jwt.go
git commit -m "feat: 创建 JWT 工具包（GenerateToken、ParseToken）"
```

---

## 任务 3：创建 JWT 认证中间件和管理员权限中间件

**涉及文件：**
- 新建: `internal/middleware/auth.go`

- [ ] **步骤 1：创建 `internal/middleware/auth.go`**

```go
package middleware

import (
	"kama-chat-server/pkg/jwt"
	"kama-chat-server/pkg/zlog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuth Gin 中间件：校验 Authorization: Bearer <token>
// 校验通过后，将 uuid 和 isAdmin 注入 gin.Context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusOK, gin.H{
				"code":    401,
				"message": "缺少认证token",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusOK, gin.H{
				"code":    401,
				"message": "token格式错误",
			})
			c.Abort()
			return
		}

		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			zlog.Error("JWT解析失败: " + err.Error())
			c.JSON(http.StatusOK, gin.H{
				"code":    401,
				"message": "token无效或已过期",
			})
			c.Abort()
			return
		}

		// 将用户信息注入上下文，后续 handler 可通过 c.Get("uuid") 获取
		c.Set("uuid", claims.Uuid)
		c.Set("isAdmin", claims.IsAdmin)
		c.Next()
	}
}

// AdminOnly Gin 中间件：要求当前用户必须是管理员（isAdmin == 1）
// 必须在 JWTAuth 之后使用
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("isAdmin")
		if !exists || isAdmin.(int8) != 1 {
			c.JSON(http.StatusOK, gin.H{
				"code":    403,
				"message": "无管理员权限",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

- [ ] **步骤 3：提交**

```bash
git add internal/middleware/auth.go
git commit -m "feat: 创建 JWT 认证中间件和管理员权限中间件"
```

---

## 任务 4：登录/注册响应结构体添加 Token 字段

**涉及文件：**
- 修改: `internal/dto/respond/user_info_respond.go`

- [ ] **步骤 1：在 LoginRespond 和 RegisterRespond 中添加 Token 字段**

```go
type LoginRespond struct {
	Uuid      string `json:"uuid"`
	Telephone string `json:"telephone"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Gender    int8   `json:"gender"`
	Signature string `json:"signature"`
	Birthday  string `json:"birthday"`
	IsAdmin   int8   `json:"isAdmin"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"createdAt"`
	Token     string `json:"token"`
}

type RegisterRespond struct {
	Uuid      string `json:"uuid"`
	Telephone string `json:"telephone"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Gender    int8   `json:"gender"`
	Birthday  string `json:"birthday"`
	Signature string `json:"signature"`
	IsAdmin   int8   `json:"isAdmin"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"createdAt"`
	Token     string `json:"token"`
}
```

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

- [ ] **步骤 3：提交**

```bash
git add internal/dto/respond/user_info_respond.go
git commit -m "feat: 登录/注册响应结构体添加 Token 字段"
```

---

## 任务 5：在 Login、Register、EmailLogin 中生成 JWT

**涉及文件：**
- 修改: `internal/service/gorm/user_info_service.go`

- [ ] **步骤 1：添加 jwt 包导入**

在 `internal/service/gorm/user_info_service.go` 的 import 中添加 `"kama-chat-server/pkg/jwt"`

- [ ] **步骤 2：在 Login 方法中生成 token**

在 Login 方法中构建 `loginRsp` 之后、`return "登录成功", loginRsp, 0` 之前，添加：

```go
		token, err := jwt.GenerateToken(user.Uuid, user.IsAdmin)
		if err != nil {
			zlog.Error("生成token失败: " + err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
		loginRsp.Token = token
```

- [ ] **步骤 3：在 Register 方法中生成 token**

在 Register 方法中构建 `registerRsp` 之后、return 之前，添加：

```go
		token, err := jwt.GenerateToken(newUser.Uuid, newUser.IsAdmin)
		if err != nil {
			zlog.Error("生成token失败: " + err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
		registerRsp.Token = token
```

注意：编辑前确认 Register 方法中实际的用户结构体变量名。

- [ ] **步骤 4：在 EmailLogin 方法中生成 token**

EmailLogin 方法和 Login 一样返回 `*respond.LoginRespond`，在构建 `loginRsp` 之后、return 之前，添加同样的 token 生成代码：

```go
		token, err := jwt.GenerateToken(user.Uuid, user.IsAdmin)
		if err != nil {
			zlog.Error("生成token失败: " + err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
		loginRsp.Token = token
```

注意：编辑前确认 EmailLogin 方法中实际的变量名。

- [ ] **步骤 5：验证编译**

```bash
go build ./...
```

- [ ] **步骤 6：提交**

```bash
git add internal/service/gorm/user_info_service.go
git commit -m "feat: 登录、注册、邮箱登录时生成 JWT token"
```

---

## 任务 6：CORS 放行 Authorization + 路由分三组

**涉及文件：**
- 修改: `internal/https_server/https_server.go`

- [ ] **步骤 1：CORS 配置放行 Authorization 请求头**

在 `init()` 函数中，修改 cors 配置：

```go
corsConfig.AllowOrigins = []string{"*"}
corsConfig.AllowMethods = []string{"GET", "PUT", "POST", "DELETE", "OPTIONS"}
corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
```

关键改动：`AllowHeaders` 增加 `"Authorization"`，`AllowMethods` 增加 `"OPTIONS"`。

- [ ] **步骤 2：添加 middleware 包导入**

在 import 中添加 `"kama-chat-server/internal/middleware"`

- [ ] **步骤 3：重构 registerRoutes 函数**

路由分为三组：公开路由、普通认证路由、管理员路由：

```go
func registerRoutes() {
	// ========== 公开路由（无需认证）==========
	GE.POST("/login", v1.Login)                          // 邮箱+密码登录
	GE.POST("/register", v1.Register)                    // 注册
	GE.POST("/user/emailLogin", v1.EmailLogin)           // 邮箱+验证码登录
	GE.POST("/user/sendEmailCode", v1.SendEmailCode)     // 发送邮箱验证码
	GE.POST("/user/verifyEmailCode", v1.VerifyEmailCode) // 验证邮箱验证码
	GE.GET("/user/wsLogin", v1.WsLogin)                  // WebSocket 连接（自带 token）

	// ========== 普通认证路由（需要 JWT）==========
	auth := GE.Group("")
	auth.Use(middleware.JWTAuth())

	// 用户
	auth.POST("/user/updateUserInfo", v1.UpdateUserInfo)
	auth.POST("/user/getUserInfo", v1.GetUserInfo)
	auth.POST("/user/wsLogout", v1.WsLogout)

	// 群组
	auth.POST("/group/createGroup", v1.CreateGroup)
	auth.POST("/group/loadMyGroup", v1.LoadMyGroup)
	auth.POST("/group/checkGroupAddMode", v1.CheckGroupAddMode)
	auth.POST("/group/enterGroupDirectly", v1.EnterGroupDirectly)
	auth.POST("/group/leaveGroup", v1.LeaveGroup)
	auth.POST("/group/dismissGroup", v1.DismissGroup)
	auth.POST("/group/getGroupInfo", v1.GetGroupInfo)
	auth.POST("/group/updateGroupInfo", v1.UpdateGroupInfo)
	auth.POST("/group/getGroupMemberList", v1.GetGroupMemberList)
	auth.POST("/group/removeGroupMembers", v1.RemoveGroupMembers)

	// 会话
	auth.POST("/session/openSession", v1.OpenSession)
	auth.POST("/session/getUserSessionList", v1.GetUserSessionList)
	auth.POST("/session/getGroupSessionList", v1.GetGroupSessionList)
	auth.POST("/session/deleteSession", v1.DeleteSession)
	auth.POST("/session/checkOpenSessionAllowed", v1.CheckOpenSessionAllowed)

	// 联系人
	auth.POST("/contact/getUserList", v1.GetUserList)
	auth.POST("/contact/loadMyJoinedGroup", v1.LoadMyJoinedGroup)
	auth.POST("/contact/getContactInfo", v1.GetContactInfo)
	auth.POST("/contact/deleteContact", v1.DeleteContact)
	auth.POST("/contact/blackContact", v1.BlackContact)
	auth.POST("/contact/cancelBlackContact", v1.CancelBlackContact)
	auth.POST("/contact/applyContact", v1.ApplyContact)
	auth.POST("/contact/getNewContactList", v1.GetNewContactList)
	auth.POST("/contact/passContactApply", v1.PassContactApply)
	auth.POST("/contact/refuseContactApply", v1.RefuseContactApply)
	auth.POST("/contact/blackApply", v1.BlackApply)
	auth.POST("/contact/getAddGroupList", v1.GetAddGroupList)

	// 消息
	auth.POST("/message/getMessageList", v1.GetMessageList)
	auth.POST("/message/getGroupMessageList", v1.GetGroupMessageList)
	auth.POST("/message/uploadAvatar", v1.UploadAvatar)
	auth.POST("/message/uploadFile", v1.UploadFile)

	// 聊天室
	auth.POST("/chatroom/getCurContactListInChatRoom", v1.GetCurContactListInChatRoom)

	// ========== 管理员路由（需要 JWT + 管理员权限）==========
	admin := GE.Group("")
	admin.Use(middleware.JWTAuth(), middleware.AdminOnly())

	admin.POST("/user/getUserInfoList", v1.GetUserInfoList)
	admin.POST("/user/ableUsers", v1.AbleUsers)
	admin.POST("/user/disableUsers", v1.DisableUsers)
	admin.POST("/user/deleteUsers", v1.DeleteUsers)
	admin.POST("/user/setAdmin", v1.SetAdmin)
	admin.POST("/group/getGroupInfoList", v1.GetGroupInfoList)
	admin.POST("/group/deleteGroups", v1.DeleteGroups)
	admin.POST("/group/setGroupsStatus", v1.SetGroupsStatus)
}
```

路由分组说明：
- **公开路由**（6 个）：登录、注册、邮箱登录、发送验证码、验证验证码、WebSocket 连接
- **普通认证路由**：需携带有效 JWT token
- **管理员路由**（8 个）：需 JWT + isAdmin == 1

- [ ] **步骤 4：验证编译**

```bash
go build ./...
```

- [ ] **步骤 5：提交**

```bash
git add internal/https_server/https_server.go
git commit -m "feat: CORS 放行 Authorization，路由分三组（公开/认证/管理员）"
```

---

## 任务 7：WebSocket 连接改为 JWT 验证

**涉及文件：**
- 修改: `api/v1/ws_controller.go`

- [ ] **步骤 1：替换 WsLogin 函数**

将原来从 `client_id` 获取用户身份改为从 `token` 参数解析 JWT：

```go
func WsLogin(c *gin.Context) {
	tokenString := c.Query("token")
	if tokenString == "" {
		zlog.Error("token获取失败")
		c.JSON(http.StatusOK, gin.H{
			"code":    401,
			"message": "token获取失败",
		})
		return
	}

	claims, err := jwt.ParseToken(tokenString)
	if err != nil {
		zlog.Error("WebSocket token验证失败: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    401,
			"message": "token无效或已过期",
		})
		return
	}

	chat.NewClientInit(c, claims.Uuid)
}
```

在 import 中添加 `"kama-chat-server/pkg/jwt"`

> **安全提示：** WebSocket 无法设置自定义 Authorization header，只能通过 query 参数传递 token。token 会出现在浏览器网络面板和后端日志中。开发阶段可接受，生产环境建议使用短有效期 token 或 WebSocket 专用临时 token。

- [ ] **步骤 2：验证编译**

```bash
go build ./...
```

- [ ] **步骤 3：提交**

```bash
git add api/v1/ws_controller.go
git commit -m "feat: WebSocket 连接改为 JWT token 验证"
```

---

## 任务 8：Controller 层授权校验——token 身份与请求体 uuid 一致性

**涉及文件：**
- 修改: `api/v1/user_info_controller.go`
- 修改: `api/v1/user_contact_controller.go`
- 修改: `api/v1/session_controller.go`
- 修改: `api/v1/group_info_controller.go`
- 修改: `api/v1/message_controller.go`
- 修改: `api/v1/chatroom_controller.go`

**核心原则：** JWT 中间件已将 `uuid` 和 `isAdmin` 注入 `gin.Context`。Controller 层必须：
1. 从 `c.Get("uuid")` 获取当前登录用户的 uuid
2. 校验请求体中的身份字段（`owner_id`、`send_id`、`user_id` 等）与 token uuid 一致
3. 对于查询类接口（如获取自己的会话列表），直接用 token uuid 替换请求体中的 owner_id，无需前端传入

- [ ] **步骤 1：在 controller.go 中添加授权校验辅助函数**

在 `api/v1/controller.go` 中添加两个辅助函数：

```go
// GetTokenUuid 从 gin.Context 获取当前登录用户的 uuid
func GetTokenUuid(c *gin.Context) string {
	uuid, _ := c.Get("uuid")
	return uuid.(string)
}

// CheckOwner 校验请求体中的 owner_id 与 token uuid 是否一致
// 不一致则返回 true（表示校验失败），并自动响应 403
func CheckOwner(c *gin.Context, ownerId string) bool {
	tokenUuid := GetTokenUuid(c)
	if ownerId != tokenUuid {
		c.JSON(http.StatusOK, gin.H{
			"code":    403,
			"message": "无权操作其他用户数据",
		})
		return true
	}
	return false
}
```

- [ ] **步骤 2：修改 user_info_controller.go**

以 `UpdateUserInfo` 为例，添加授权校验：

```go
func UpdateUserInfo(c *gin.Context) {
	var req request.UpdateUserInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
		return
	}
	// 授权校验：只能修改自己的信息
	if CheckOwner(c, req.Uuid) {
		return
	}
	message, ret := gorm.UserInfoService.UpdateUserInfo(req)
	JsonBack(c, message, ret, nil)
}
```

`GetUserInfo`：校验 `req.Uuid == token uuid`（用户只能查自己，管理员由 AdminOnly 中间件保证）。

`GetUserInfoList`：已在管理员路由组，无需额外校验。

`AbleUsers`、`DisableUsers`、`DeleteUsers`、`SetAdmin`：已在管理员路由组，无需额外校验。

- [ ] **步骤 3：修改 session_controller.go**

`OpenSession`：校验 `req.SendId == token uuid`

`GetUserSessionList`：用 `GetTokenUuid(c)` 替换请求体中的 `OwnerId`，不信任前端传入

`GetGroupSessionList`：同上，用 `GetTokenUuid(c)` 替换

`DeleteSession`：校验 `req.OwnerId == token uuid`

`CheckOpenSessionAllowed`：校验 `req.SendId == token uuid`

- [ ] **步骤 4：修改 user_contact_controller.go**

`GetUserList`：用 `GetTokenUuid(c)` 替换请求体中的 `OwnerId`

`LoadMyJoinedGroup`：同上

`GetContactInfo`：无需校验（只根据 contact_id 查询）

`DeleteContact`：校验 `req.OwnerId == token uuid`

`BlackContact`：同上

`CancelBlackContact`：同上

`ApplyContact`：校验 `req.OwnerId == token uuid`

`GetNewContactList`：用 `GetTokenUuid(c)` 替换 `OwnerId`

`PassContactApply`：校验 `req.OwnerId == token uuid`

`RefuseContactApply`：同上

`BlackApply`：同上

`GetAddGroupList`：校验 `req.GroupId` 对应的群主是 token uuid（或在管理员路由中）

- [ ] **步骤 5：修改 group_info_controller.go**

`CreateGroup`：用 `GetTokenUuid(c)` 替换 `req.OwnerId`

`LoadMyGroup`：用 `GetTokenUuid(c)` 替换请求体中的用户标识

`CheckGroupAddMode`：无需校验（只读查询）

`EnterGroupDirectly`：校验 `req.ContactId == token uuid`

`LeaveGroup`：校验 `req.UserId == token uuid`

`DismissGroup`：校验 `req.OwnerId == token uuid`（群主才能解散）

`GetGroupInfo`：无需校验（只读查询）

`UpdateGroupInfo`：校验 `req.OwnerId == token uuid`（群主才能修改）

`GetGroupMemberList`：无需校验（只读查询）

`RemoveGroupMembers`：校验 `req.OwnerId == token uuid`（群主才能移除）

`GetGroupInfoList`、`DeleteGroups`、`SetGroupsStatus`：已在管理员路由组，无需额外校验

- [ ] **步骤 6：修改 message_controller.go**

`GetMessageList`：用 `GetTokenUuid(c)` 作为查询条件之一

`GetGroupMessageList`：无需校验（只读查询）

`UploadAvatar`：无需校验（上传后关联到当前用户）

`UploadFile`：同上

- [ ] **步骤 7：修改 chatroom_controller.go**

`GetCurContactListInChatRoom`：校验 `req.OwnerId == token uuid`

- [ ] **步骤 8：验证编译**

```bash
go build ./...
```

- [ ] **步骤 9：提交**

```bash
git add api/v1/
git commit -m "feat: Controller 层添加 token 身份与请求体 uuid 授权校验"
```

---

## 任务 9：前端 — AuthResponse 类型 + token 存储

**涉及文件：**
- 修改: `react_web/src/types/api.ts`
- 修改: `react_web/src/api/auth.ts`
- 修改: `react_web/src/stores/useAuthStore.ts`
- 修改: `react_web/src/api/axios.ts`
- 修改: `react_web/src/services/websocket.ts`

- [ ] **步骤 1：添加 AuthResponse 类型**

在 `react_web/src/types/api.ts` 中添加：

```typescript
import type { UserInfo } from './user'

// 登录/注册响应（包含 token）
export type AuthResponse = UserInfo & {
  token: string
}
```

- [ ] **步骤 2：修改 auth.ts 返回类型**

```typescript
import api from './axios'
import type { ApiResponse, LoginRequest, RegisterRequest, EmailLoginRequest, SendEmailCodeRequest } from '../types/api'
import type { AuthResponse } from '../types/api'
import type { UserInfo } from '../types/user'

export async function login(data: LoginRequest): Promise<ApiResponse<AuthResponse>> {
  const res = await api.post<ApiResponse<AuthResponse>>('/login', data)
  return res.data
}

export async function register(data: RegisterRequest): Promise<ApiResponse<AuthResponse>> {
  const res = await api.post<ApiResponse<AuthResponse>>('/register', data)
  return res.data
}

export async function emailLogin(data: EmailLoginRequest): Promise<ApiResponse<AuthResponse>> {
  const res = await api.post<ApiResponse<AuthResponse>>('/user/emailLogin', data)
  return res.data
}

export async function sendEmailCode(data: SendEmailCodeRequest): Promise<ApiResponse<null>> {
  const res = await api.post<ApiResponse<null>>('/user/sendEmailCode', data)
  return res.data
}
```

- [ ] **步骤 3：在 useAuthStore 中添加 token 管理**

在 `AuthState` 接口中添加 `token` 字段：

```typescript
interface AuthState {
  userInfo: UserInfo | null
  token: string | null
  login: (data: LoginRequest) => Promise<boolean>
  emailLogin: (data: EmailLoginRequest) => Promise<boolean>
  register: (data: RegisterRequest) => Promise<boolean>
  logout: () => void
  updateProfile: (data: Partial<Pick<UserInfo, 'nickname' | 'gender' | 'signature' | 'birthday' | 'avatar'>>) => Promise<boolean>
  uploadAndSetAvatar: (file: File) => Promise<string | null>
}
```

修改 `getInitialUserInfo` 函数，同时检查 token：

```typescript
function getInitialUserInfo(): UserInfo | null {
  const stored = sessionStorage.getItem('userInfo')
  const token = sessionStorage.getItem('token')
  if (stored && token) {
    try {
      const user = JSON.parse(stored) as UserInfo
      wsService.connect(token, WS_URL)
      return user
    } catch {
      sessionStorage.removeItem('userInfo')
      sessionStorage.removeItem('token')
    }
  }
  return null
}
```

初始状态添加：`token: sessionStorage.getItem('token')`

在 `login`、`emailLogin`、`register` 方法中，从响应中提取 token 并存储（避免 token 混入 userInfo）：

```typescript
// 以 login 为例，emailLogin 和 register 同理
const { token, ...userData } = res.data
if (!token) {
  showToast('登录异常：未获取到token', 'error')
  return false
}
const user = normalizeUserInfo(userData)
sessionStorage.setItem('userInfo', JSON.stringify(user))
sessionStorage.setItem('token', token)
wsService.connect(token, WS_URL)
set({ userInfo: user, token })
```

修改 `logout` 方法，清除 token：

```typescript
logout: () => {
  const { userInfo } = get()
  if (userInfo) {
    wsLogout(userInfo.uuid).catch(() => {})
  }
  wsService.disconnect()
  sessionStorage.removeItem('userInfo')
  sessionStorage.removeItem('token')
  useChatStore.getState().resetAll()
  set({ userInfo: null, token: null })
},
```

- [ ] **步骤 4：axios 拦截器添加 Bearer token 和 401 处理**

修改 `react_web/src/api/axios.ts`：

请求拦截器 — 自动附加 token：

```typescript
api.interceptors.request.use((config) => {
  const token = sessionStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  if (config.data instanceof FormData) {
    delete config.headers['Content-Type']
  } else {
    config.headers['Content-Type'] = 'application/json'
  }
  return config
})
```

响应拦截器 — 处理 401 过期和 403 权限不足：

```typescript
api.interceptors.response.use(
  (response) => {
    const data = response.data as ApiResponse
    if (data.code === 401) {
      sessionStorage.removeItem('userInfo')
      sessionStorage.removeItem('token')
      window.location.href = '/login'
      return Promise.reject(new Error('登录已过期'))
    }
    if (data.code === 403) {
      showToast('无操作权限', 'error')
      return Promise.reject(new Error('无操作权限'))
    }
    if (data.code === 400) {
      console.warn('[API Warning]', data.message)
    } else if (data.code === 500) {
      console.error('[API Error]', data.message)
    }
    return response
  },
  (error) => {
    console.error('[API Error]', '网络请求失败')
    return Promise.reject(error)
  }
)
```

- [ ] **步骤 5：修改 WebSocket 连接使用 token**

修改 `react_web/src/services/websocket.ts`：

将 `clientId` 字段替换为 `token`，修改 `connect` 方法签名：

```typescript
private token: string = ''

connect(token: string, wsBaseUrl: string): void {
  if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN && this.token === token)) {
    return
  }
  this.token = token
  this.wsBaseUrl = wsBaseUrl
  this.reconnectAttempts = 0
  this.intentionalClose = false
  this.doConnect()
}
```

修改 `doConnect` 中的 URL：

```typescript
const url = `${this.wsBaseUrl}/user/wsLogin?token=${this.token}`
```

删除 `clientId` 字段及其所有引用。

- [ ] **步骤 6：验证前端构建**

```bash
cd react_web && npm run build
```

- [ ] **步骤 7：提交**

```bash
git add react_web/src/types/api.ts react_web/src/api/auth.ts react_web/src/stores/useAuthStore.ts react_web/src/api/axios.ts react_web/src/services/websocket.ts
git commit -m "feat: 前端 JWT token 存储、请求拦截器、WebSocket 认证"
```

---

## 任务 10：端到端集成测试

**涉及文件：** 无（手动测试）

- [ ] **步骤 1：启动后端**

```bash
cd c:/Users/li/Desktop/kama-chat-server
go run cmd/kama-chat-server/main.go
```

- [ ] **步骤 2：启动前端**

```bash
cd react_web && npm run dev
```

- [ ] **步骤 3：测试登录流程**

1. 打开 `http://localhost:5173` 进入登录页
2. 用邮箱+密码登录
3. 检查：响应中包含 `token` 字段，已存入 sessionStorage（与 userInfo 分开存储）
4. 检查：后续 API 请求带有 `Authorization: Bearer <token>` 请求头（浏览器 DevTools → Network 查看）
5. 检查：WebSocket 连接 URL 包含 `?token=<jwt>`

- [ ] **步骤 4：测试 401 过期处理**

1. 浏览器 DevTools → Application → Session Storage
2. 删除 `token` 条目
3. 触发任意 API 调用（如发消息）
4. 检查：响应返回 code 401，自动跳转到登录页

- [ ] **步骤 5：测试 403 权限校验**

1. 用普通用户登录
2. 尝试调用管理员接口（如 `/user/getUserInfoList`）
3. 检查：响应返回 code 403，提示"无管理员权限"
4. 尝试修改请求体中的 `owner_id` 为其他用户 uuid
5. 检查：响应返回 code 403，提示"无权操作其他用户数据"

- [ ] **步骤 6：测试公开路由正常工作**

1. 检查：`/login`、`/register`、`/user/sendEmailCode` 无需 token 即可访问
2. 检查：注册新用户并确认返回了 token

- [ ] **步骤 7：提交最终状态**

```bash
git add -A
git commit -m "feat: 完成 JWT 认证系统"
```

---

## 改动汇总

### 后端（Go）

| 文件 | 改动 |
|------|------|
| `go.mod` | 添加 `github.com/golang-jwt/jwt/v5` 依赖 |
| `configs/config_local.toml` | 添加 `[jwtConfig]` 配置段 |
| `internal/config/config.go` | 添加 `JwtConfig` 结构体 |
| `pkg/jwt/jwt.go` | **新建** — JWT 生成/解析函数（含签名算法校验） |
| `internal/middleware/auth.go` | **新建** — JWT 认证中间件 + AdminOnly 管理员权限中间件 |
| `internal/dto/respond/user_info_respond.go` | LoginRespond、RegisterRespond 增加 Token 字段 |
| `internal/service/gorm/user_info_service.go` | Login、Register、EmailLogin 中生成 token |
| `internal/https_server/https_server.go` | CORS 放行 Authorization + 路由分三组（公开/认证/管理员） |
| `api/v1/controller.go` | 添加 `GetTokenUuid()` 和 `CheckOwner()` 授权辅助函数 |
| `api/v1/user_info_controller.go` | 校验 uuid 与 token 一致 |
| `api/v1/user_contact_controller.go` | 校验 owner_id 与 token 一致 |
| `api/v1/session_controller.go` | 校验 send_id/owner_id 与 token 一致 |
| `api/v1/group_info_controller.go` | 校验 owner_id/user_id 与 token 一致 |
| `api/v1/message_controller.go` | 用 token uuid 替代请求体用户标识 |
| `api/v1/chatroom_controller.go` | 校验 owner_id 与 token 一致 |
| `api/v1/ws_controller.go` | WebSocket 从 client_id 改为 token 验证 |

### 前端（React + TypeScript）

| 文件 | 改动 |
|------|------|
| `react_web/src/types/api.ts` | 添加 `AuthResponse` 类型 |
| `react_web/src/api/auth.ts` | 返回类型改为 `AuthResponse` |
| `react_web/src/stores/useAuthStore.ts` | 存储 token（与 userInfo 分开），登录/注册时保存，登出时清除 |
| `react_web/src/api/axios.ts` | 请求拦截器添加 Bearer token，响应拦截器处理 401/403 |
| `react_web/src/services/websocket.ts` | WebSocket 连接 URL 使用 `?token=<jwt>` 参数 |

---

## 与原计划的主要差异

| 项目 | 原计划 | 修正版 |
|------|--------|--------|
| CORS | 未提及 | 放行 `Authorization` 请求头 |
| 路由分组 | 公开 + 认证两组 | 公开 + 认证 + 管理员三组 |
| 管理员权限 | 无 | 新增 `AdminOnly()` 中间件，管理员接口 403 拦截 |
| 请求体 uuid 校验 | 无 | Controller 层用 `CheckOwner()` 校验 owner_id/send_id/user_id 与 token uuid 一致 |
| JWT 签名算法校验 | 无 | `ParseToken` 中显式校验 `SigningMethodHS256` |
| 前端 token 处理 | `(res.data as any).token` | 新增 `AuthResponse` 类型，解构 `{ token, ...userData }` 避免 token 混入 userInfo |
| 403 响应处理 | 无 | 前端 axios 拦截器增加 403 处理 |
| WebSocket 安全提示 | 无 | 计划中注明 query token 的安全风险 |
