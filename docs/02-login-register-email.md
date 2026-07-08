# GoChat 登录注册与邮箱验证码设计 — 面试详解

## 一、整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         前端                                     │
│                                                                  │
│  LoginPage                  RegisterPage                         │
│  ┌─────────────────┐       ┌──────────────────┐                 │
│  │ Tab1: 账号密码登录│       │ 邮箱 + 密码 + 昵称│                │
│  │ 邮箱 + 密码      │       │ + 邮箱验证码      │                │
│  │                  │       │                  │                 │
│  │ Tab2: 邮箱验证码  │       │  [获取验证码]     │                │
│  │ 邮箱 + 验证码    │       │                  │                 │
│  └────────┬────────┘       └────────┬─────────┘                 │
│           │                         │                           │
│  useAuthStore.login()    useAuthStore.register()                │
│  useAuthStore.emailLogin()                                      │
│           │                         │                           │
│  api/auth.ts → Axios(自动带JWT)                                  │
└───────────┬─────────────────────────┬───────────────────────────┘
            │                         │
            ▼                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                         后端                                     │
│                                                                  │
│  公开路由（无需JWT）：                                             │
│  POST /login              → v1.Login                            │
│  POST /register           → v1.Register                         │
│  POST /user/emailLogin    → v1.EmailLogin                       │
│  POST /user/sendEmailCode → v1.SendEmailCode                    │
│  POST /user/verifyEmailCode → v1.VerifyEmailCode                │
│                                                                  │
│  Controller → Service → DAO(MySQL) / Redis / Email(SMTP)        │
└─────────────────────────────────────────────────────────────────┘
```

## 二、三种登录方式

| 方式 | 接口 | 参数 | 场景 |
|------|------|------|------|
| 邮箱+密码 | `POST /login` | email, password | 常规登录 |
| 邮箱+验证码 | `POST /user/emailLogin` | email, emailCode | 免密码登录 |
| 注册 | `POST /register` | email, password, nickname, emailCode | 新用户注册 |

## 三、邮箱验证码流程

### 3.1 发送验证码（`POST /user/sendEmailCode`）

```
前端                          后端                          Redis/QQ邮箱
 │                             │                              │
 │  POST /user/sendEmailCode   │                              │
 │  {email: "xxx@qq.com"}     │                              │
 │────────────────────────────►│                              │
 │                             │  1. 查Redis: email_code_xxx  │
 │                             │─────────────────────────────►│
 │                             │  2. key不存在（未发过）         │
 │                             │◄─────────────────────────────│
 │                             │                              │
 │                             │  3. 生成6位随机验证码           │
 │                             │     code = 851117             │
 │                             │                              │
 │                             │  4. 存Redis: TTL 5分钟        │
 │                             │     email_code_xxx = 851117  │
 │                             │─────────────────────────────►│
 │                             │                              │
 │                             │  5. SMTP发送邮件               │
 │                             │──────────────────────────────────►QQ邮箱
 │                             │                              │
 │  {code:200, message:"验证码  │                              │
 │   已发送至 xxx@qq.com"}      │                              │
 │◄────────────────────────────│                              │
```

代码实现（`email/auth_code_service.go`）：

```go
func SendVerificationCode(toEmail string) (string, int) {
    // 1. 检查是否已发送（防重复）
    key := "email_code_" + toEmail
    code, err := redis.GetKey(key)
    if code != "" {
        return "验证码已发送，请检查邮箱或5分钟后重试", -2
    }

    // 2. 生成6位随机验证码
    codeInt, _ := random.GetRandomInt(6)
    code = strconv.Itoa(codeInt)

    // 3. 存入Redis，5分钟过期
    redis.SetKeyEx(key, code, 5*time.Minute)

    // 4. 发送邮件
    err = sendEmail(toEmail, code)

    return "验证码已发送至 " + toEmail, 0
}
```

### 3.2 验证验证码（`VerifyCode`）

```go
func VerifyCode(email string, inputCode string) (string, int) {
    key := "email_code_" + email

    storedCode, err := redis.GetKey(key)
    if storedCode == "" {
        return "验证码已过期，请重新获取", -2   // key过期了
    }
    if storedCode != inputCode {
        return "验证码错误", -2                // 码不对
    }

    redis.DelKeyIfExists(key)   // ✅ 验证成功，立即删除，防止复用
    return "验证成功", 0
}
```

> **Q: 为什么验证成功后要立即删除验证码？**
> A: 防止同一个验证码被重复使用。如果验证完不删，在 5 分钟 TTL 内别人拿到这个验证码也能登录。删除后下次必须重新发送，保证"一次一码"。

### 3.3 邮件发送（`sendEmail`）

```go
func sendEmail(toEmail string, code string) error {
    e := email.NewEmail()
    e.From = "GoChat <746894206@qq.com>"
    e.To = []string{toEmail}
    e.Subject = "【GoChat】邮箱验证码"
    e.HTML = []byte(buildEmailHTML(code))   // HTML 美化邮件

    addr := "smtp.qq.com:465"               // QQ邮箱 SSL 端口
    auth := smtp.PlainAuth("", "746894206@qq.com", "授权码", "smtp.qq.com")

    tlsConfig := &tls.Config{
        InsecureSkipVerify: true,
        ServerName:         "smtp.qq.com",
    }

    return e.SendWithTLS(addr, auth, tlsConfig)   // SSL/TLS 加密发送
}
```

| 要点 | 说明 |
|------|------|
| **端口 465** | QQ 邮箱 SSL 直连端口，对应 `SendWithTLS` |
| **端口 587** | STARTTLS 端口，需用 `Send` 方法，不能用 `SendWithTLS` |
| **授权码** | 不是 QQ 密码，是 QQ 邮箱设置里生成的 SMTP 专用授权码 |
| **TLS 加密** | 验证码通过加密通道传输，防止被中间人截获 |
| **HTML 邮件** | 用模板生成带样式的邮件，比纯文本体验好 |

> **Q: 为什么用 465 端口而不是 587？**
> A: 代码用的是 `SendWithTLS` 方法，这个方法是先建立 TLS 连接再发邮件，对应 465 端口（SSL/TLS 直连）。587 端口是 STARTTLS，先明文连接再升级加密，需要用 `Send` 方法。两种方式都能发，但要方法跟端口匹配。

## 四、注册流程（`POST /register`）

```
前端                              后端                        数据库/Redis
 │                                 │                            │
 │  POST /register                 │                            │
 │  {email, password,              │                            │
 │   nickname, emailCode}          │                            │
 │────────────────────────────────►│                            │
 │                                 │  1. 验证邮箱验证码           │
 │                                 │     VerifyCode()            │
 │                                 │───────────────────────────►│ Redis
 │                                 │     验证成功，删除验证码      │
 │                                 │◄───────────────────────────│
 │                                 │                            │
 │                                 │  2. 检查邮箱是否已注册        │
 │                                 │     GORM: WHERE email=?    │
 │                                 │───────────────────────────►│ MySQL
 │                                 │     未注册，继续             │
 │                                 │◄───────────────────────────│
 │                                 │                            │
 │                                 │  3. 创建用户                │
 │                                 │     UUID = "U" + 日期 + 随机│
 │                                 │     默认头像、isAdmin=0      │
 │                                 │     GORM: Create(&newUser)  │
 │                                 │───────────────────────────►│ MySQL
 │                                 │◄───────────────────────────│
 │                                 │                            │
 │                                 │  4. 生成 JWT Token          │
 │                                 │     GenerateToken(uuid, 0)  │
 │                                 │                            │
 │  {code:200, data:{             │                            │
 │   uuid, nickname, email,       │                            │
 │   avatar, token, ...}}         │                            │
 │◄────────────────────────────────│                            │
 │                                 │                            │
 │  前端存 localStorage             │                            │
 │  连接 WebSocket                  │                            │
```

代码实现（`user_info_service.go`）：

```go
func (u *userInfoService) Register(registerReq request.RegisterRequest) (string, *respond.RegisterRespond, int) {
    // 1. 验证邮箱验证码
    message, ret := email.VerifyCode(registerReq.Email, registerReq.EmailCode)
    if ret != 0 {
        return message, nil, ret
    }

    // 2. 检查邮箱是否已注册
    message, ret = u.checkEmailExist(registerReq.Email)
    if ret != 0 {
        return message, nil, ret
    }

    // 3. 创建用户
    uuidStr, _ := random.GetNowAndLenRandomString(11)
    newUser := model.UserInfo{
        Uuid:     "U" + uuidStr,
        Email:    registerReq.Email,
        Password: registerReq.Password,
        Nickname: registerReq.Nickname,
        Avatar:   "默认头像URL",
        IsAdmin:  0,
        Status:   0,
    }
    dao.GormDB.Create(&newUser)

    // 4. 生成 JWT Token
    token, _ := myjwt.GenerateToken(newUser.Uuid, newUser.IsAdmin)

    return "注册成功", &respond.RegisterRespond{..., Token: token}, 0
}
```

> **Q: 注册成功后为什么直接返回 Token，不用再登录？**
> A: 用户体验优化。注册完自动登录，减少一步操作。Token 和登录返回的完全一样，前端拿到后直接存 localStorage + 连 WebSocket，流程和登录一致。

> **Q: 密码为什么没有加密？**
> A: 当前是明文存储，这是一个安全隐患。生产环境应该用 bcrypt 哈希存储密码，验证时用 `bcrypt.CompareHashAndPassword` 比对。明文存储的问题：数据库泄露后所有用户密码暴露。

## 五、邮箱+密码登录流程（`POST /login`）

```
后端处理步骤：

1. 根据邮箱查用户      → GORM: First(&user, "Email=?", email)
2. 用户不存在？         → 返回 "用户不存在，请注册"
3. 密码不匹配？         → 返回 "密码不正确，请重试"
4. 用户被禁用？         → 返回 "用户已禁用"（Status == 1）
5. 构建响应 + 生成Token → GenerateToken(user.Uuid, user.IsAdmin)
6. 返回用户信息 + Token
```

> **Q: 为什么先查用户再验密码，而不是一条 SQL 同时查？**
> A: 因为两种错误要给不同提示——"用户不存在"和"密码错误"。如果用 `WHERE email=? AND password=?` 一条查，查不到不知道是用户不存在还是密码错了。分开处理可以给用户精确提示。但这也有安全隐患：攻击者可以通过错误提示探测邮箱是否注册。更安全的做法是统一提示"邮箱或密码错误"。

## 六、邮箱验证码登录流程（`POST /user/emailLogin`）

```
后端处理步骤：

1. 根据邮箱查用户      → GORM: First(&user, "email=?", email)
2. 用户不存在？         → 返回 "用户不存在, 请注册"
3. 验证邮箱验证码       → VerifyCode(email, code)
4. 验证码过期/错误？    → 返回对应错误
5. 用户被禁用？         → 返回 "用户已禁用"
6. 构建响应 + 生成Token → GenerateToken(user.Uuid, user.IsAdmin)
7. 返回用户信息 + Token
```

> **Q: 邮箱验证码登录和密码登录的区别？**
> A: 密码登录用密码验证身份，验证码登录用邮箱验证码。验证码登录的优势是免密码，不用记密码；劣势是依赖邮箱服务可用性，且验证码有 5 分钟有效期限制。

## 七、前端验证码发送的防刷设计

```typescript
// 60秒倒计时，防止频繁发送
const [codeCooldown, setCodeCooldown] = useState(0)

const handleSendCode = async () => {
    const res = await sendEmailCode({ email })
    if (res.code === 200) {
        setCodeCooldown(60)  // 60秒倒计时
        setInterval(() => {
            setCodeCooldown(prev => prev <= 1 ? 0 : prev - 1)
        }, 1000)
    }
}
```

| 防刷层级 | 实现 | 作用 |
|----------|------|------|
| **前端** | 60秒倒计时按钮禁用 | 防止用户短时间内重复点击 |
| **后端** | Redis 检查 `email_code_{email}` 是否存在 | 5分钟内已发过验证码则拒绝重复发送 |

> **Q: 只有这两层防刷够吗？**
> A: 不够。当前没有限制单个 IP 的发送频率，攻击者可以换不同邮箱地址绕过。生产环境需要加：IP 维度限流（如单 IP 每小时最多发 5 次）、图形验证码（发送前先验证）、邮箱维度全局限流。

## 八、前端登录状态管理（`useAuthStore`）

```typescript
// Zustand Store
export const useAuthStore = create<AuthState>((set, get) => ({
    userInfo: getInitialUserInfo(),  // 从 localStorage 恢复

    login: async (data) => {
        const res = await loginApi(data)
        if (res.code === 200) {
            // 1. 存 localStorage
            localStorage.setItem('userInfo', JSON.stringify(res.data))
            localStorage.setItem('token', res.data.token)
            // 2. 连接 WebSocket
            wsService.connect(res.data.token, WS_URL)
            // 3. 更新状态
            set({ userInfo: res.data })
        }
    },

    logout: () => {
        // 1. 通知后端 WebSocket 登出
        wsLogout(userInfo.uuid)
        // 2. 断开 WebSocket
        wsService.disconnect()
        // 3. 清 localStorage
        localStorage.removeItem('token')
        localStorage.removeItem('userInfo')
        // 4. 清状态
        set({ userInfo: null })
    },
}))
```

登录成功后的三件事：

```
1. localStorage 存储 token + userInfo → 刷新页面不丢失登录状态
2. wsService.connect(token)           → WebSocket 连接（token 放 Query 参数）
3. set({ userInfo })                  → Zustand 状态更新，UI 响应
```

页面刷新时的恢复逻辑：

```typescript
function getInitialUserInfo(): UserInfo | null {
    const stored = localStorage.getItem('userInfo')
    const token = localStorage.getItem('token')
    if (stored && token) {
        const user = JSON.parse(stored)
        wsService.connect(token, WS_URL)  // 用存储的 token 重连 WebSocket
        return user
    }
    return null
}
```

## 九、面试速记口诀

1. **三种登录**：邮箱+密码、邮箱+验证码、注册（注册完自动登录）
2. **验证码流程**：发验证码 → 存 Redis（TTL 5分钟）→ SMTP 发邮件 → 验证后立即删除防复用
3. **防重复发送**：前端 60秒倒计时 + 后端 Redis key 存在性检查
4. **注册流程**：验证码 → 查邮箱是否已注册 → 创建用户（UUID U前缀）→ 生成 JWT → 返回（免二次登录）
5. **密码安全**：当前明文存储（生产应用 bcrypt 哈希）
6. **登录状态**：token 存 localStorage + Zustand 管理状态 + 页面刷新自动恢复 + WebSocket 重连
7. **邮件发送**：QQ 邮箱 SMTP + 465 端口 SSL/TLS + 授权码（非 QQ 密码）
