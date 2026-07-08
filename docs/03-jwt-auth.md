# GoChat JWT 认证体系 — 面试详解

## 一、整体架构

```
用户登录/注册
    │
    ▼
┌──────────────────────────┐
│  Service 层              │
│  GenerateToken(uuid,     │
│    isAdmin) → JWT字符串   │
│  算法: HS256              │
│  密钥: 配置文件读取        │
└──────────┬───────────────┘
           │ 返回 token 给前端
           ▼
┌──────────────────────────┐
│  前端 localStorage        │
│  存储 token + userInfo    │
└──────┬──────────┬────────┘
       │          │
  HTTP请求    WebSocket连接
  Header:     Query参数:
  Bearer xxx  ?token=xxx
       │          │
       ▼          ▼
┌──────────────────────────┐
│  JWTAuth 中间件           │
│  1. 优先读 Header         │
│  2. 其次读 Query          │
│  3. ParseToken 校验       │
│  4. 注入 uuid,isAdmin     │
│     到 gin.Context        │
└──────┬───────────────────┘
       │
       ▼
┌──────────┐    ┌──────────┐
│ 普通路由  │    │ Admin路由 │
│ 直接放行  │    │ AdminOnly │
│          │    │ isAdmin==1│
└──────────┘    └──────────┘
```

## 二、Token 生成（`pkg/jwt/jwt.go`）

```go
type Claims struct {
    Uuid    string `json:"uuid"`      // 用户唯一ID
    IsAdmin int8   `json:"isAdmin"`   // 是否管理员 0/1
    jwt.RegisteredClaims               // 标准字段
}
```

### 关键设计点

| 要点 | 实现方式 | 面试怎么说 |
|------|---------|-----------|
| **签名算法** | HS256（HMAC-SHA256 对称加密） | 选 HS256 是因为单体架构，密钥只需服务端持有，不需要 RSA 非对称。如果微服务需要多个服务验签，才考虑 RS256 |
| **自定义载荷** | `Uuid` + `IsAdmin` | 只放用户标识和角色，**不放敏感信息**（密码、邮箱），因为 JWT payload 只做 Base64 编码，不是加密 |
| **标准字段** | `ExpiresAt`、`NotBefore`、`IssuedAt`、`Issuer` | 遵循 RFC 7519 规范，用 `RegisteredClaims` 结构体类型安全地管理 |
| **过期时间** | 配置文件 `expireHours`，默认 24h | 可配置化，不硬编码。生产环境一般设 2-8 小时 |
| **密钥来源** | 配置文件 `jwtConfig.secret` | 启动时校验不能是默认占位符，防止开发者忘记改密钥直接上线 |
| **生成时机** | 登录/注册成功后 | 在 Service 层调用 `GenerateToken(user.Uuid, user.IsAdmin)`，token 随响应体返回 |

### 面试可能追问

> **Q: 为什么 payload 不放更多信息？**
> A: JWT 的 payload 是 Base64 编码不是加密，任何人都能解码。只放 uuid 和 isAdmin 最小必要信息，敏感数据通过 uuid 查库获取。同时 token 体积越小，网络传输开销越低。

> **Q: 为什么不用 RSA 非对称？**
> A: 项目是单体架构，验签只有这一个服务，HS256 对称加密足够。RS256 的优势是公钥可以分发给多个服务验签而私钥只在签发服务，适合微服务场景，但密钥管理和性能开销更大，单体没必要。

## 三、Token 校验（`internal/middleware/auth.go`）

### 双模式提取 — 面试核心亮点

```go
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
```

### 为什么需要双模式？

| 场景 | 方式 | 原因 |
|------|------|------|
| **REST API** | `Authorization: Bearer <token>` | HTTP 标准，Axios 请求拦截器自动附加 |
| **WebSocket** | `?token=<jwt>` | 浏览器 WebSocket API **不支持自定义 Header**，只能在连接 URL 上传参 |

> **Q: WebSocket 为什么不能传 Header？**
> A: 浏览器 `new WebSocket(url)` API 没有提供设置自定义 Header 的能力，这是 WebSocket 协议规范的限制。所以只能通过 query 参数传递 token，这是业界通用做法（如 Socket.IO 的 `auth` 参数也是类似思路）。

### ParseToken 安全校验

```go
token, err := jwt.ParseWithClaims(tokenString, &Claims{},
    func(token *jwt.Token) (interface{}, error) {
        return []byte(conf.JwtConfig.Secret), nil
    },
    jwt.WithValidMethods([]string{"HS256"}),   // ① 限制算法防攻击
    jwt.WithExpirationRequired(),                // ② 强制要求过期时间
    jwt.WithIssuer(Issuer),                      // ③ 校验签发者
)
```

| 校验项 | 作用 | 防什么攻击 |
|--------|------|-----------|
| `WithValidMethods` | 白名单只允许 HS256 | **算法混淆攻击**：攻击者把 Header 的 alg 改成 `none` 或 RS256 绕过签名 |
| `WithExpirationRequired` | 必须有 exp 字段 | 防止生成永不过期的 token |
| `WithIssuer` | 校验 iss 必须是 "gochat" | 防止其他系统签发的 token 被误用 |

### 错误细分处理

```go
switch {
case errors.Is(err, jwt.ErrTokenExpired):       // token过期
case errors.Is(err, jwt.ErrTokenNotValidYet):   // token尚未生效
case errors.Is(err, jwt.ErrTokenMalformed):     // token格式错误
case errors.Is(err, jwt.ErrTokenSignatureInvalid): // 签名无效
}
```

> **Q: token 过期了怎么办？**
> A: 当前实现是直接返回 401 让前端跳转登录页重新登录。没有做 Refresh Token 机制。如果要加，方案是：登录时同时返回 accessToken（短过期 2h）+ refreshToken（长过期 7d），accessToken 过期后用 refreshToken 换新的，refreshToken 过期才需要重新登录。

## 四、上下文注入 & 权限控制

```go
// JWTAuth 中间件：解析成功后注入上下文
c.Set("uuid", claims.Uuid)
c.Set("isAdmin", claims.IsAdmin)

// 下游 Controller 取值
uuid := GetTokenUuid(c)  // 从 context 取 uuid

// CheckOwner 校验：防止操作他人数据
tokenUuid != ownerId → 403 "无权操作他人数据"

// AdminOnly 中间件：管理员专属路由
isAdmin != 1 → 403 "无管理员权限"
```

### 三级路由对应的鉴权

```
公开路由   → 无中间件       → /login, /register
认证路由   → JWTAuth()     → /user/*, /group/*, /session/*, /contact/*, /message/*
管理员路由 → JWTAuth() +   → /user/ableUsers, /user/disableUsers, /user/setAdmin...
             AdminOnly()
```

## 五、前端配合

```typescript
// 请求拦截器：自动附加 token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：401 自动跳转登录
if (data.code === 401) {
  localStorage.removeItem('token')
  localStorage.removeItem('userInfo')
  window.location.href = '/login'
}

// WebSocket 连接：token 放 query 参数
wsService.connect(token, WS_URL)  // → ws://host/user/wsLogin?token=xxx
```

> **Q: token 存 localStorage 还是 Cookie？**
> A: 项目存 localStorage。优点是跨域方便、前端可控；缺点是无法防 XSS 攻击窃取。如果存 HttpOnly Cookie 则防 XSS 但有 CSRF 风险，需要额外加 CSRF Token。两种方案各有取舍，生产环境更推荐 HttpOnly Cookie + CSRF 防护。

## 六、面试速记口诀

1. **生成**：登录/注册后 Service 层调 `GenerateToken`，HS256 签名，payload 只放 uuid + isAdmin
2. **传递**：HTTP 走 Bearer Header，WebSocket 走 Query 参数（浏览器 API 不支持自定义 Header）
3. **校验**：中间件 `JWTAuth` 双模式提取 → `ParseToken` 验签名+过期+签发者+算法白名单 → 注入 Context
4. **权限**：三级路由（公开/认证/管理员），`AdminOnly` 校 isAdmin，`CheckOwner` 防越权操作
5. **安全**：算法白名单防混淆攻击、启动时强制改默认密钥、token 过期强制重新登录
