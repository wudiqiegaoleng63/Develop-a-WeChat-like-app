# 教学文档 24: JWT认证与鉴权

## 一、什么是JWT？

JWT（JSON Web Token）是一种开放标准（RFC 7519），用于在各方之间安全地传输信息。在我们的聊天系统中，JWT用于**用户认证**——登录后服务器签发一个token，客户端后续请求携带这个token，服务器通过验证token确认用户身份。

### JWT的结构

一个JWT由三部分组成，用 `.` 分隔：

```
Header.Payload.Signature
```

例如：

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1dWlkIjoiVTIwMjQwMTAxMTIzNDUiLCJpc0FkbWluIjowLCJpc3MiOiJnb2NoYXQiLCJleHAiOjE3MDAwMDAwMDB9.xxxxxxxxxxxxx
```

**Header（头部）**：算法和类型

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

**Payload（载荷）**：存放实际数据（我们的自定义字段 + 标准声明）

```json
{
  "uuid": "U2024010112345",
  "isAdmin": 0,
  "iss": "gochat",
  "exp": 1700000000,
  "nbf": 1699900000,
  "iat": 1699900000
}
```

**Signature（签名）**：用密钥对前两部分签名，防止篡改

```
HMACSHA256(base64UrlEncode(header) + "." + base64UrlEncode(payload), secret)
```

### 为什么用JWT而不是Session？

| 特性 | Session | JWT |
|------|---------|-----|
| 存储位置 | 服务器内存/Redis | 客户端（localStorage/sessionStorage） |
| 扩展性 | 需要共享Session存储 | 无状态，天然支持分布式 |
| 跨域 | 需要额外处理 | 天然支持（token放在Header中） |
| WebSocket | 需要特殊处理 | 通过query参数传递即可 |

我们的聊天系统使用WebSocket，JWT通过 `?token=xxx` 传递给WebSocket，非常方便。

---

## 二、整体认证流程

```
┌─────────── 登录/注册 ───────────┐
│                                  │
│  前端 ──POST /login──→ Controller │
│         (邮箱+密码)       │      │
│                          ↓      │
│                      Service层   │
│                    (验证密码)    │
│                          │      │
│                          ↓      │
│                 myjwt.GenerateToken(uuid, isAdmin) │
│                          │      │
│                          ↓      │
│              返回 { token: "eyJ..." }  │
│                                  │
└──────────────────────────────────┘

┌─────────── 后续请求 ───────────┐
│                                  │
│  前端 ──请求──→ JWTAuth中间件    │
│  (Header: Authorization:        │
│   Bearer eyJ...)         │      │
│                      ↓          │
│              myjwt.ParseToken()  │
│                      │          │
│              ┌───────┴───────┐  │
│              ↓               ↓  │
│          验证成功         验证失败 │
│              │               │  │
│              ↓               ↓  │
│     c.Set("uuid",...)   返回401 │
│     c.Set("isAdmin",..)         │
│              │                   │
│              ↓                   │
│         Controller               │
│     (通过c.Get获取用户信息)      │
│                                  │
└──────────────────────────────────┘
```

---

## 三、涉及的文件

| 顺序 | 层 | 文件位置 | 作用 |
|------|----|---------|------|
| ① | 配置 | `internal/config/config.go` | JWT配置结构体（密钥、过期时间） |
| ② | 配置 | `configs/config_local.toml` | JWT密钥和过期时间配置 |
| ③ | 核心包 | `pkg/jwt/jwt.go` | token生成和解析 |
| ④ | 中间件 | `internal/middleware/auth.go` | Gin中间件：JWT认证 + 管理员鉴权 |
| ⑤ | Service | `internal/service/gorm/user_info_service.go` | 调用GenerateToken（Login/Register/EmailLogin） |
| ⑥ | DTO | `internal/dto/respond/user_info_respond.go` | 响应结构体中的Token字段 |
| ⑦ | Controller | `api/v1/controller.go` | GetTokenUuid、CheckOwner辅助函数 |
| ⑧ | 路由 | `internal/https_server/https_server.go` | 公开路由 vs 认证路由 vs 管理员路由 |
| ⑨ | 前端 | `react_web/src/api/axios.ts` | 请求拦截器附加token + 响应拦截器处理401/403 |

---

## 四、配置层

### 4.1 配置结构体

**文件位置:** `internal/config/config.go`

```go
// JwtConfig - JWT配置
type JwtConfig struct {
	Secret      string `toml:"secret"`      // JWT签名密钥
	ExpireHours int    `toml:"expireHours"` // Token过期时间（小时）
}
```

JwtConfig嵌入到总配置结构体中：

```go
// 总配置
type Config struct {
	MainConfig    	`toml:"mainConfig"`
    MysqlConfig  	`toml:"mysqlConfig"`
    LogConfig      	`toml:"logConfig"`
	StaticSrcConfig `toml:"staticSrcConfig"`
	RedisConfig     `toml:"redisConfig"`
	EmailConfig   	`toml:"emailConfig"`
	KafkaConfig     `toml:"kafkaConfig"`
	JwtConfig       `toml:"jwtConfig"`
}
```

### 4.2 配置文件

**文件位置:** `configs/config_local.toml`

```toml
[jwtConfig]
secret = "53ca0427ad3f11a50a63a934a0c4e47ddd09e75aae66060d2d59ebfbe43005d2"
expireHours = 24
```

- `secret`：HMAC-SHA256签名密钥，必须是随机的高强度字符串
- `expireHours`：token过期时间，24小时

### 4.3 安全校验

在 `LoadConfig()` 中，有一个启动时的安全检查：

```go
func LoadConfig() error {
	_, err := toml.DecodeFile("configs/config_local.toml", config)
	if err != nil {
		log.Fatal("配置加载失败:", err.Error())
		return err
	}
	// 校验JWT密钥不能是默认占位符
	if config.JwtConfig.Secret == "gochat-jwt-secret-key-change-in-production" {
		log.Fatal("JWT secret 不能使用默认值，请在 config_local.toml 中修改 jwtConfig.secret")
	}
	return nil
}
```

如果密钥是默认占位符 `"gochat-jwt-secret-key-change-in-production"`，程序直接 `log.Fatal` 退出，防止开发者忘记修改密钥就上线。

---

## 五、核心JWT包

**文件位置:** `pkg/jwt/jwt.go`

**依赖库:** `github.com/golang-jwt/jwt/v5`（v5版本，当前最新主版本）

### 5.1 Claims结构体（自定义载荷）

```go
package jwt

import (
	"errors"
	"kama-chat-server/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "gochat"

// Claims JWT 载荷
type Claims struct {
	Uuid    string `json:"uuid"`
	IsAdmin int8   `json:"isAdmin"`
	jwt.RegisteredClaims
}
```

**自定义字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `Uuid` | string | 用户唯一标识（如 `U2024010112345`） |
| `IsAdmin` | int8 | 是否管理员（1=管理员，0=普通用户） |

**嵌入的 `jwt.RegisteredClaims`（标准声明）：**

| 字段 | 说明 | 我们的使用 |
|------|------|-----------|
| `ExpiresAt` | 过期时间 | ✅ 必须设置 |
| `NotBefore` | 生效时间 | ✅ 防止token在签发前被使用 |
| `IssuedAt` | 签发时间 | ✅ 记录token创建时间 |
| `Issuer` | 签发者 | ✅ 设为 `"gochat"`，解析时校验 |
| `Subject` | 主题 | ❌ 未使用 |
| `Audience` | 受众 | ❌ 未使用 |
| `ID` | 唯一ID | ❌ 未使用 |

`Issuer` 被提取为常量 `const Issuer = "gochat"`，确保生成和解析时使用同一个值。

### 5.2 GenerateToken（生成token）

```go
// GenerateToken 生成 JWT token
func GenerateToken(uuid string, isAdmin int8) (string, error) {
	conf := config.GetConfig()
	expireHours := conf.JwtConfig.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}

	now := time.Now()
	claims := Claims{
		Uuid:    uuid,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(conf.JwtConfig.Secret))
}
```

**逐行解析：**

1. `conf := config.GetConfig()` — 获取配置实例
2. `expireHours` 默认值保护 — 如果配置为0或负数，默认24小时
3. `now := time.Now()` — 统一获取当前时间，确保 `ExpiresAt`、`NotBefore`、`IssuedAt` 使用同一个时间基准
4. `jwt.NewNumericDate(now.Add(...))` — 将 `time.Time` 转为JWT的数值日期格式
5. `ExpiresAt` — 过期时间 = 当前时间 + expireHours小时
6. `NotBefore` — 生效时间 = 当前时间，防止token在签发前被使用
7. `IssuedAt` — 签发时间 = 当前时间
8. `Issuer` — 签发者 = `"gochat"`
9. `jwt.NewWithClaims(jwt.SigningMethodHS256, claims)` — 用HS256算法创建token对象
10. `token.SignedString([]byte(conf.JwtConfig.Secret))` — 用密钥签名并返回token字符串

### 5.3 ParseToken（解析token）

```go
// ParseToken 解析 JWT token
func ParseToken(tokenString string) (*Claims, error) {
	conf := config.GetConfig()
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtConfig.Secret), nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
```

**逐行解析：**

1. `jwt.ParseWithClaims` — 解析token字符串，将结果映射到 `&Claims{}` 结构体
2. `func(token *jwt.Token) (interface{}, error)` — Keyfunc回调，返回签名密钥
3. **`jwt.WithValidMethods([]string{"HS256"})`** — 🔒 安全选项：只允许HS256签名算法，拒绝其他算法（防止算法混淆攻击）
4. **`jwt.WithExpirationRequired()`** — 🔒 安全选项：强制要求 `exp` 声明存在，防止无过期时间的token永久有效
5. **`jwt.WithIssuer(Issuer)`** — 🔒 安全选项：验证 `iss` 声明等于 `"gochat"`，防止其他服务签发的token被接受
6. 解析失败直接返回错误（v5库的错误类型可以用 `errors.Is` 判断具体原因）
7. 类型断言 `token.Claims.(*Claims)` 提取自定义载荷
8. `token.Valid` 确保token通过了所有验证

### 5.4 三个安全选项详解

#### WithValidMethods — 防止算法混淆攻击

```
攻击场景：
1. 攻击者拿到一个token
2. 将Header中的 alg 改为 "none"（无签名）
3. 或者改为 "RS256"（用公钥验证）
4. 如果服务器不校验算法，可能接受伪造的token
```

**修复前（不安全）：**

```go
// 手动指针比较，不够严谨
if token.Method != jwt.SigningMethodHS256 {
    return nil, errors.New("unexpected signing method")
}
```

**修复后（安全）：**

```go
// 使用v5推荐的WithValidMethods选项
jwt.WithValidMethods([]string{"HS256"})
```

区别：`WithValidMethods` 是 jwt/v5 库官方推荐的写法，在解析阶段就进行校验，比手动在Keyfunc中检查更早、更可靠。

#### WithExpirationRequired — 防止永久token

```
攻击场景：
1. 如果代码Bug导致生成token时没有设置exp
2. 这个token永远不会过期
3. 即使泄露也无法使其失效
```

加上 `WithExpirationRequired()` 后，如果token中没有 `exp` 字段，解析直接返回错误：

```
token has invalid claims: token is missing required claim: exp claim is required
```

#### WithIssuer — 防止跨服务token复用

```
攻击场景：
1. 同一个公司有多个服务，使用相同的JWT密钥
2. 服务A签发的token，如果不在服务B中校验issuer
3. 服务B会错误地接受服务A的token
```

加上 `jwt.WithIssuer(Issuer)` 后，只有 `iss: "gochat"` 的token才会被接受。

---

## 六、认证中间件

**文件位置:** `internal/middleware/auth.go`

### 6.1 JWTAuth中间件

```go
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	myjwt "kama-chat-server/pkg/jwt"
	"kama-chat-server/pkg/zlog"
)

// JWTAuth Gin 中间件：校验 Authorization: Bearer <token> 或 query ?token=<jwt>
// 校验通过后，将 uuid 和 isAdmin 注入 gin.Context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// 1. 优先从 Authorization header 获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			}
		}

		// 2. 如果 header 中没有，从 query 参数获取（用于 WebSocket）
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少认证token",
			})
			c.Abort()
			return
		}

		claims, err := myjwt.ParseToken(tokenStr)
		if err != nil {
			zlog.Error("JWT解析失败: " + err.Error())

			msg := "token无效"
			switch {
			case errors.Is(err, jwt.ErrTokenExpired):
				msg = "token已过期"
			case errors.Is(err, jwt.ErrTokenNotValidYet):
				msg = "token尚未生效"
			case errors.Is(err, jwt.ErrTokenMalformed):
				msg = "token格式错误"
			case errors.Is(err, jwt.ErrTokenSignatureInvalid):
				msg = "token签名无效"
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": msg,
			})
			c.Abort()
			return
		}

		c.Set("uuid", claims.Uuid)
		c.Set("isAdmin", claims.IsAdmin)
		c.Next()
	}
}
```

**关键设计点：**

#### 双通道token提取

```
1. HTTP请求 → Authorization: Bearer <token>
2. WebSocket → ?token=<jwt>（因为WebSocket无法设置自定义Header）
```

优先从Header获取，如果没有则从query参数获取。这样两种场景都能覆盖。

#### 错误类型区分

使用 `errors.Is()` 对JWT错误进行分类，给前端返回不同的错误信息：

| 错误类型 | 常量 | 返回消息 |
|---------|------|---------|
| token过期 | `jwt.ErrTokenExpired` | "token已过期" |
| token未生效 | `jwt.ErrTokenNotValidYet` | "token尚未生效" |
| token格式错误 | `jwt.ErrTokenMalformed` | "token格式错误" |
| 签名无效 | `jwt.ErrTokenSignatureInvalid` | "token签名无效" |
| 其他错误 | — | "token无效" |

这样前端可以根据不同的错误类型展示不同的提示（如过期提示"请重新登录"，格式错误提示"token异常"）。

#### 上下文注入

```go
c.Set("uuid", claims.Uuid)
c.Set("isAdmin", claims.IsAdmin)
```

解析成功后，将 `uuid` 和 `isAdmin` 注入Gin的上下文。后续的Controller可以通过 `c.Get("uuid")` 获取当前用户身份，无需再解析token。

#### HTTP状态码

认证失败返回 `http.StatusUnauthorized`（401），而不是 `http.StatusOK`（200）+ 业务code。这是RESTful API的标准做法：

- 401：未认证（token缺失、过期、无效）
- 403：已认证但无权限（普通用户访问管理员接口）

### 6.2 AdminOnly中间件

```go
// AdminOnly Gin 中间件：要求当前用户必须是管理员（isAdmin == 1）
// 必须在 JWTAuth 之后使用
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("isAdmin")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "无管理员权限",
			})
			c.Abort()
			return
		}
		isAdminInt, ok := isAdmin.(int8)
		if !ok || isAdminInt != 1 {
			c.JSON(http.StatusForbidden, gin.H{
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

**关键设计点：**

1. **必须在 `JWTAuth` 之后使用** — 依赖 `JWTAuth` 注入的 `isAdmin` 字段
2. **两重检查** — 先检查字段是否存在（`exists`），再检查类型断言（`ok`）和值（`isAdminInt != 1`）
3. **返回403而非401** — 用户已经认证（有token），但权限不足，所以是Forbidden（403）而不是Unauthorized（401）

---

## 七、Service层：生成Token

**文件位置:** `internal/service/gorm/user_info_service.go`

在三个地方生成token：

### 7.1 Login（邮箱+密码登录）

```go
// 生成JWT token
token, err := myjwt.GenerateToken(user.Uuid, user.IsAdmin)
if err != nil {
    zlog.Error("生成token失败: " + err.Error())
    return constants.SYSTEM_ERROR, nil, -1
}
loginRsp.Token = token
```

### 7.2 Register（注册）

```go
token, err := myjwt.GenerateToken(newUser.Uuid, newUser.IsAdmin)
if err != nil {
    zlog.Error("生成token失败: " + err.Error())
    return constants.SYSTEM_ERROR, nil, -1
}
registerRsp.Token = token
```

### 7.3 EmailLogin（邮箱+验证码登录）

```go
token, err := myjwt.GenerateToken(user.Uuid, user.IsAdmin)
if err != nil {
    zlog.Error("生成token失败: " + err.Error())
    return constants.SYSTEM_ERROR, nil, -1
}
loginRsp.Token = token
```

三个接口逻辑完全一致：认证成功后，用用户的 `Uuid` 和 `IsAdmin` 生成token，放入响应结构体返回给前端。

---

## 八、DTO层：Token字段

**文件位置:** `internal/dto/respond/user_info_respond.go`

```go
// LoginRespond - 登录响应
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
    Token     string `json:"token"`       // JWT token
}

// RegisterRespond - 注册响应
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
    Token     string `json:"token"`       // JWT token
}
```

`LoginRespond` 和 `RegisterRespond` 都包含 `Token` 字段，通过 `json:"token"` 序列化为前端可读的JSON字段。

---

## 九、Controller层：使用Token信息

**文件位置:** `api/v1/controller.go`

### 9.1 GetTokenUuid — 获取当前用户ID

```go
// GetTokenUuid 从JWT中间件注入的上下文中获取当前用户uuid
func GetTokenUuid(c *gin.Context) string {
	uuid, exists := c.Get("uuid")
	if !exists {
		return ""
	}
	s, ok := uuid.(string)
	if !ok {
		return ""
	}
	return s
}
```

从Gin上下文中取出 `JWTAuth` 中间件注入的 `uuid`，做类型断言后返回。

### 9.2 CheckOwner — 校验资源所有权

```go
// CheckOwner 校验请求中的ownerId是否与token中的uuid一致
// 返回true表示一致（通过），false表示不一致（拒绝）
func CheckOwner(c *gin.Context, ownerId string) bool {
	tokenUuid := GetTokenUuid(c)
	if tokenUuid != ownerId {
		c.JSON(http.StatusOK, gin.H{
			"code":    403,
			"message": "无权操作他人数据",
		})
		return false
	}
	return true
}
```

**使用场景举例：**

- `UpdateUserInfo`：用户只能修改自己的信息
- `WsLogout`：用户只能登出自己的WebSocket

```go
// UpdateUserInfo 中
if !CheckOwner(c, req.Uuid) {
    return
}

// WsLogout 中
if !CheckOwner(c, req.OwnerId) {
    return
}
```

### 9.3 WsLogin中的token使用

**文件位置:** `api/v1/ws_controller.go`

```go
// WsLogin WebSocket登录
func WsLogin(c *gin.Context) {
	// 从JWT中间件注入的上下文中获取uuid
	uuid, exists := c.Get("uuid")
	if !exists {
		zlog.Error("uuid获取失败")
		c.JSON(http.StatusOK, gin.H{
			"code":    401,
			"message": "未认证",
		})
		return
	}
	clientId := uuid.(string)

	chat.NewClientInit(c, clientId)
}
```

WebSocket连接时，直接从上下文中取 `uuid` 作为客户端ID。

---

## 十、路由层：三种路由组

**文件位置:** `internal/https_server/https_server.go`

```go
func registerRoutes() {
	// ===== 公开路由（无需认证） =====
	public := GE.Group("")
	{
		public.POST("/login", v1.Login)
		public.POST("/register", v1.Register)
		public.POST("/user/emailLogin", v1.EmailLogin)
		public.POST("/user/sendEmailCode", v1.SendEmailCode)
		public.POST("/user/verifyEmailCode", v1.VerifyEmailCode)
	}

	// ===== 认证路由（需要JWT） =====
	auth := GE.Group("").Use(middleware.JWTAuth())
	{
		auth.POST("/user/updateUserInfo", v1.UpdateUserInfo)
		auth.POST("/user/getUserInfo", v1.GetUserInfo)
		auth.GET("/user/wsLogin", v1.WsLogin)       // WebSocket登录（GET请求）
		auth.POST("/user/wsLogout", v1.WsLogout)
		auth.POST("/group/createGroup", v1.CreateGroup)
		// ... 更多路由
	}

	// ===== 管理员路由（需要JWT + 管理员权限） =====
	admin := GE.Group("").Use(middleware.JWTAuth(), middleware.AdminOnly())
	{
		admin.POST("/user/getUserInfoList", v1.GetUserInfoList)
		admin.POST("/user/ableUsers", v1.AbleUsers)
		admin.POST("/user/disableUsers", v1.DisableUsers)
		admin.POST("/user/deleteUsers", v1.DeleteUsers)
		admin.POST("/user/setAdmin", v1.SetAdmin)
		admin.POST("/group/getGroupInfoList", v1.GetGroupInfoList)
		admin.POST("/group/deleteGroups", v1.DeleteGroups)
		admin.POST("/group/setGroupsStatus", v1.SetGroupsStatus)
	}
}
```

**三级权限模型：**

| 级别 | 中间件 | 可访问接口 | 说明 |
|------|--------|-----------|------|
| 公开 | 无 | login, register, sendEmailCode... | 不需要token |
| 认证用户 | `JWTAuth()` | updateUserInfo, wsLogin, createGroup... | 需要有效token |
| 管理员 | `JWTAuth()` + `AdminOnly()` | getUserInfoList, deleteUsers, setAdmin... | 需要token且isAdmin=1 |

注意管理员路由组同时使用了 `middleware.JWTAuth()` 和 `middleware.AdminOnly()` 两个中间件，中间件按顺序执行：先验证token有效性，再验证管理员权限。

---

## 十一、前端：Axios拦截器

**文件位置:** `react_web/src/api/axios.ts`

### 11.1 请求拦截器：自动附加token

```typescript
// 请求拦截器：自动附加JWT token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
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

每次请求自动从 `localStorage` 取出token，添加到 `Authorization: Bearer <token>` 头中。

### 11.2 响应拦截器：处理认证错误

```typescript
// 响应拦截器：处理401/403
api.interceptors.response.use(
  (response) => {
    // 处理HTTP 200但业务code为401/403的情况
    const data = response.data as ApiResponse
    if (data.code === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
      useAuthStore.setState({ userInfo: null })
      window.location.href = '/login'
      return Promise.reject(new Error(data.message || '未认证'))
    }
    if (data.code === 403) {
      console.warn('[API Forbidden]', data.message)
      return Promise.reject(new Error(data.message || '无权限'))
    }
    if (data.code === 400) {
      console.warn('[API Warning]', data.message)
    } else if (data.code === 500) {
      console.error('[API Error]', data.message)
    }
    return response
  },
  (error) => {
    // 处理后端返回的HTTP 401/403状态码
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
      useAuthStore.setState({ userInfo: null })
      window.location.href = '/login'
      return Promise.reject(new Error(error.response.data?.message || '未认证'))
    }
    if (error.response?.status === 403) {
      console.warn('[API Forbidden]', error.response.data?.message)
      return Promise.reject(new Error(error.response.data?.message || '无权限'))
    }
    console.error('[API Error]', '网络请求失败')
    return Promise.reject(error)
  }
)
```

**双重处理机制：**

| 场景 | 处理位置 | 原因 |
|------|---------|------|
| HTTP 200 + `data.code === 401` | 成功回调 | 后端业务层返回的认证错误（如Controller中的CheckOwner） |
| HTTP 401 状态码 | 错误回调 | JWT中间件拦截返回的认证错误 |

两种场景都执行相同的逻辑：清除token和用户信息，跳转到登录页。

---

## 十二、完整请求生命周期示例

### 示例1：用户登录

```
1. 前端: POST /login { email: "user@test.com", password: "123456" }
2. Controller: Login() 接收请求
3. Service: 验证邮箱密码
4. Service: myjwt.GenerateToken("U2024010112345", 0)
5. JWT包: 创建Claims，HS256签名，返回token字符串
6. Service: 将token放入LoginRespond
7. Controller: JsonBack返回 { code: 200, data: { ..., token: "eyJ..." } }
8. 前端: 存储token到localStorage
```

### 示例2：发送聊天消息（认证请求）

```
1. 前端: 从localStorage取出token
2. Axios拦截器: 添加 Authorization: Bearer eyJ... 到请求头
3. Gin路由: 匹配到认证路由组，执行JWTAuth()中间件
4. JWTAuth中间件:
   a. 从Authorization头提取token
   b. 调用myjwt.ParseToken(tokenString)
   c. ParseWithClaims验证:
      - 签名算法是HS256 ✓
      - exp声明存在 ✓
      - iss等于"gochat" ✓
      - token未过期 ✓
      - nbf不晚于当前时间 ✓
   d. 解析成功，提取Claims
   e. c.Set("uuid", "U2024010112345")
   f. c.Set("isAdmin", 0)
   g. c.Next() — 放行
5. Controller: 通过GetTokenUuid(c)获取当前用户ID
6. Service: 执行业务逻辑
7. 返回响应
```

### 示例3：管理员删除用户（管理员请求）

```
1. 前端: POST /user/deleteUsers (带管理员token)
2. Gin路由: 匹配到管理员路由组
3. JWTAuth中间件: 验证token，注入uuid和isAdmin=1
4. AdminOnly中间件:
   a. c.Get("isAdmin") → 1
   b. isAdminInt == 1 ✓
   c. c.Next() — 放行
5. Controller: 执行管理员操作
```

### 示例4：token过期

```
1. 前端: 带过期token发送请求
2. JWTAuth中间件: 调用myjwt.ParseToken()
3. ParseWithClaims: 验证失败，返回 jwt.ErrTokenExpired
4. JWTAuth中间件:
   a. errors.Is(err, jwt.ErrTokenExpired) → true
   b. msg = "token已过期"
   c. 返回 HTTP 401 { code: 401, message: "token已过期" }
5. Axios错误回调:
   a. error.response.status === 401
   b. 清除localStorage中的token和userInfo
   c. window.location.href = '/login' — 跳转登录页
```

### 示例5：WebSocket连接

```
1. 前端: new WebSocket("ws://localhost:8000/user/wsLogin?token=eyJ...")
2. Gin路由: 匹配 GET /user/wsLogin，执行JWTAuth()中间件
3. JWTAuth中间件:
   a. Authorization头为空
   b. 从query参数获取: c.Query("token") → "eyJ..."
   c. 验证token成功
   d. 注入uuid和isAdmin
4. WsLogin Controller: c.Get("uuid") 获取用户ID
5. chat.NewClientInit(c, clientId) — 初始化WebSocket连接
```

---

## 十三、安全设计总结

### 我们实现的5层安全防护

| 层级 | 防护措施 | 防御的攻击 |
|------|---------|-----------|
| 1 | `WithValidMethods(["HS256"])` | 算法混淆攻击（将alg改为none或RS256） |
| 2 | `WithExpirationRequired()` | 永久token（无exp声明的token无限有效） |
| 3 | `WithIssuer("gochat")` | 跨服务token复用（其他服务签发的token被接受） |
| 4 | `NotBefore` 声明 | token时间回溯（在签发时间之前使用token） |
| 5 | 启动时密钥校验 | 默认密钥上线（忘记修改占位符密钥） |

### 认证与鉴权的区别

```
认证(Authentication)：你是谁？   → JWTAuth中间件  → 401
鉴权(Authorization)：你能做什么？ → AdminOnly中间件 → 403
```

### 前后端协作要点

| 要点 | 后端 | 前端 |
|------|------|------|
| token存储 | 响应体 `data.token` | `localStorage.setItem('token', ...)` |
| token传递 | 读取Header或query | Axios拦截器自动附加 `Authorization: Bearer` |
| 过期处理 | 返回401 + 具体错误消息 | 清除token + 跳转登录页 |
| 权限不足 | AdminOnly返回403 | 控制台警告 + 拒绝Promise |

### 为什么选择 localStorage 而不是 sessionStorage？

| 特性 | `sessionStorage` | `localStorage` |
|------|-----------------|----------------|
| 关闭标签页 | 数据清空 | 数据保留 |
| 关闭浏览器 | 数据清空 | 数据保留 |
| 跨标签页共享 | 不共享 | 同源下共享 |
| 手动清除 | 需代码清除 | 需代码或手动清除 |

我们选择 `localStorage` 的原因：

1. **关闭页面后仍保持登录** — 用户关闭标签页或浏览器后再次访问，不需要重新登录
2. **token有过期时间保护** — JWT自带 `exp` 声明，即使本地存储的token仍在，后端也会拒绝过期的token
3. **登出时主动清除** — `logout()` 中调用 `localStorage.removeItem('token')` 清除登录状态

如果使用 `sessionStorage`，关闭标签页就会丢失登录状态，用户体验较差。
