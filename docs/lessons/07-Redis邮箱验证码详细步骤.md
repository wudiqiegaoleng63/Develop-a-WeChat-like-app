# 教学文档 07: Redis邮箱验证码详细步骤

> **目标读者**：有Go基础的开发者
> **前置条件**：已安装Redis并能正常运行

---

## 一、Redis环境验证

### 1.1 验证Redis是否运行

```bash
redis-cli ping
```

**预期输出**：
```
PONG
```

### 1.2 基本命令测试

```bash
redis-cli

# 设置带过期时间的键（60秒）
SET test_code "123456" EX 60

# 获取
GET test_code

# 查看剩余过期时间
TTL test_code

# 退出
exit
```

---

## 二、QQ邮箱SMTP配置

### 2.1 开启SMTP服务

1. 登录 [QQ邮箱](https://mail.qq.com/)
2. 点击右上角 **设置** → **账户**
3. 向下滚动找到 **POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务**
4. 开启 **POP3/SMTP服务**
5. 按提示发送短信验证
6. **记录授权码**（16位字符，只显示一次！）

> ⚠️ **重要**：授权码不是QQ密码，是专门用于SMTP登录的密码

### 2.2 SMTP信息汇总

| 配置项 | 值 |
|--------|-----|
| SMTP服务器 | smtp.qq.com |
| 端口 | 465（SSL加密） |
| 用户名 | 你的QQ邮箱 |
| 密码 | 授权码（16位） |

---

## 三、安装Go依赖

### 3.1 安装Redis客户端

```bash
cd C:\Users\li\Desktop\kama-chat-server
go get github.com/redis/go-redis/v9
```

### 3.2 安装邮件发送库

```bash
go get github.com/jordan-wright/email
```

### 3.3 验证依赖安装

```bash
grep -E "redis|email" go.mod
```

**预期输出**：
```
github.com/jordan-wright/email v5.x.x
github.com/redis/go-redis/v9 v9.x.x
```

---

## 四、更新配置文件

### 4.1 修改config_local.toml

打开 `configs/config_local.toml`，将 `[authCodeConfig]` 改为邮箱配置：

```toml
[redisConfig]
host = "127.0.0.1"
port = 6379
password = ""
db = 0

# 邮箱验证码配置
[emailConfig]
smtpHost     = "smtp.qq.com"           # SMTP服务器
smtpPort     = 465                      # 端口（SSL加密）
smtpUsername = "your_email@qq.com"      # 发件邮箱
smtpPassword = "xxxxxxxxxxxxxxxx"       # 授权码（16位）
fromName     = "KamaChat"               # 发件人名称
```

### 4.2 添加EmailConfig结构体

打开 `internal/config/config.go`，添加：

```go
// EmailConfig 邮箱验证码配置
type EmailConfig struct {
	SmtpHost     string `toml:"smtpHost"`     // SMTP服务器
	SmtpPort     int    `toml:"smtpPort"`     // 端口
	SmtpUsername string `toml:"smtpUsername"` // 发件邮箱
	SmtpPassword string `toml:"smtpPassword"` // 授权码
	FromName     string `toml:"fromName"`     // 发件人名称
}
```

### 4.3 更新Config结构体

```go
// Config 总配置
type Config struct {
	MainConfig     `toml:"mainConfig"`
	MysqlConfig    `toml:"mysqlConfig"`
	LogConfig      `toml:"logConfig"`
	StaticSrcConfig `toml:"staticSrcConfig"`
	AuthCodeConfig `toml:"authCodeConfig"`
	RedisConfig    `toml:"redisConfig"`
	EmailConfig    `toml:"emailConfig"` // ★新增
}
```

---

## 五、创建随机数工具

文件路径：`pkg/util/random/random_int.go`

```go
package random

import (
	"errors"
	"math/rand/v2"
	"strconv"
	"time"
)

// ============================================================
// GetRandomInt - 生成指定位数的随机数字
// ============================================================
// n: 位数，有效范围 1-18
// 返回: n位随机数字，如 n=6 返回 123456~999999
func GetRandomInt(n int) (int, error) {
	// 参数校验
	if n <= 0 {
		return 0, errors.New("位数必须大于0")
	}
	if n > 18 {
		return 0, errors.New("位数不能超过18（防止int溢出）")
	}

	min := pow(10, n-1)
	max := pow(10, n) - 1
	return rand.IntN(max-min+1) + min, nil
}

// ============================================================
// GetNowAndLenRandomString - 生成日期+随机数字字符串
// ============================================================
// 格式: 日期(8位) + 随机数字(n位) = 8+n位字符串
// 例: n=11 → "20260424" + "12345678901" = "2026042412345678901" (19位)
// 用途: 生成Uuid，如 "U" + GetNowAndLenRandomString(11) = 20位Uuid
func GetNowAndLenRandomString(n int) (string, error) {
	randomInt, err := GetRandomInt(n)
	if err != nil {
		return "", err
	}
	return time.Now().Format("20060102") + strconv.Itoa(randomInt), nil
}

// ============================================================
// pow - 整数幂运算（辅助函数）
// ============================================================
func pow(base, n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= base
	}
	return result
}
```

**关键说明**：

| 函数 | 返回值 | 用途 |
|------|--------|------|
| `GetRandomInt(6)` | `(int, error)` | 生成6位验证码 |
| `GetNowAndLenRandomString(11)` | `(string, error)` | 生成Uuid（日期+随机） |

---

## 六、创建Redis服务

文件路径：`internal/service/redis/redis_service.go`

```go
package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"kama-chat-server/internal/config"
	"kama-chat-server/pkg/zlog"
)

var redisClient *redis.Client
var ctx = context.Background()

func init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.Host
	port := conf.RedisConfig.Port
	password := conf.RedisConfig.Password
	db := conf.Db
	addr := host + ":" + strconv.Itoa(port)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// SetKeyEx 设置键值（带过期时间）
func SetKeyEx(key string, value string, timeout time.Duration) error {
	err := redisClient.Set(ctx, key, value, timeout).Err()
	if err != nil {
		return err
	}
	return nil
}

// GetKey 获取键值（不存在返回空字符串）
func GetKey(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zlog.Info("该key不存在")
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// GetKeyNilIsErr 获取键值（不存在返回redis.Nil错误）
func GetKeyNilIsErr(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

// GetKeyWithPrefixNilIsErr 根据前缀查找单个key（不存在返回redis.Nil错误）
// ★用途：查找特定前缀开头的key，如 "session_U123_G456" 开头的key
// ★注意：只能有一个匹配key，多个匹配会报错
// ★使用Scan增量迭代，不会阻塞Redis，适合生产环境
func GetKeyWithPrefixNilIsErr(prefix string) (string, error) {
	var cursor uint64
	var foundKeys []string

	for {
		// 使用Scan增量迭代，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return "", err
		}

		// 收集找到的键
		foundKeys = append(foundKeys, keys...)

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	if len(foundKeys) == 0 {
		zlog.Info("没有找到相关前缀key")
		return "", redis.Nil
	}

	if len(foundKeys) == 1 {
		zlog.Info(fmt.Sprintln("成功找到了相关前缀key", foundKeys))
		return foundKeys[0], nil
	} else {
		zlog.Error("找到了数量大于1的key，查找异常")
		return "", errors.New("找到了数量大于1的key，查找异常")
	}
}

// GetKeyWithSuffixNilIsErr 根据后缀查找单个key（不存在返回redis.Nil错误）
// ★用途：查找特定后缀结尾的key
// ★注意：只能有一个匹配key，多个匹配会报错
// ★使用Scan增量迭代，不会阻塞Redis，适合生产环境
func GetKeyWithSuffixNilIsErr(suffix string) (string, error) {
	var cursor uint64
	var foundKeys []string

	for {
		// 使用Scan增量迭代，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*"+suffix, 100).Result()
		if err != nil {
			return "", err
		}

		// 收集找到的键
		foundKeys = append(foundKeys, keys...)

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	if len(foundKeys) == 0 {
		zlog.Info("没有找到相关后缀key")
		return "", redis.Nil
	}

	if len(foundKeys) == 1 {
		zlog.Info(fmt.Sprintln("成功找到了相关后缀key", foundKeys))
		return foundKeys[0], nil
	} else {
		zlog.Error("找到了数量大于1的key，查找异常")
		return "", errors.New("找到了数量大于1的key，查找异常")
	}
}

// DelKeyIfExists 删除单个键（先检查是否存在）
func DelKeyIfExists(key string) error {
	exists, err := redisClient.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 1 {
		delErr := redisClient.Del(ctx, key).Err()
		if delErr != nil {
			return delErr
		}
	}
	return nil
}

// DelKeysWithPattern 删除精确匹配pattern的key
// ★使用Scan增量迭代，不会阻塞Redis，适合生产环境
func DelKeysWithPattern(pattern string) error {
	var cursor uint64
	for {
		// 使用Scan代替Keys，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			log.Println("成功删除相关对应key", keys)
		}

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	return nil
}

// DelKeysWithPrefix 删除带前缀的所有键（模糊匹配 prefix*）
// ★使用Scan增量迭代，不会阻塞Redis，适合生产环境
func DelKeysWithPrefix(prefix string) error {
	var cursor uint64
	for {
		// ★使用Scan代替Keys，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			log.Println("成功删除相关前缀key", keys)
		}

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	return nil
}

// DelKeysWithSuffix 删除带后缀的所有键（模糊匹配 *suffix）
// ★使用Scan增量迭代，不会阻塞Redis，适合生产环境
func DelKeysWithSuffix(suffix string) error {
	var cursor uint64
	for {
		// ★使用Scan代替Keys，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*"+suffix, 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			log.Println("成功删除相关后缀key", keys)
		}

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	return nil
}

// DeleteAllRedisKeys 清空所有Redis键（测试用，生产慎用）
// ★警告：此方法会删除Redis中所有数据，生产环境禁止使用！
func DeleteAllRedisKeys() error {
	var cursor uint64
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			return err
		}
		cursor = nextCursor

		if len(keys) > 0 {
			_, err := redisClient.Del(ctx, keys...).Result()
			if err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}
```

### ★★★ Keys vs Scan 重要对比 ★★★

| 方法 | Keys(pattern) | Scan(cursor, pattern, count) |
|------|---------------|------------------------------|
| **执行方式** | 一次性返回所有匹配key | 分批增量返回，每次count条 |
| **性能影响** | ❌ **阻塞Redis**，大量数据时性能急剧下降 | ✅ **不阻塞**，不影响其他操作 |
| **内存使用** | ❌ 一次性加载所有key，内存可能爆 | ✅ 每批count条，内存可控 |
| **生产环境** | ⚠️ 不推荐用于大量数据 | ✅ 安全可用 |

```go
// ❌ Keys命令 - 阻塞式（不推荐）
// 当Redis有100万条数据时，Keys会遍历整个数据库，阻塞所有其他操作数秒
keys, err := redisClient.Keys(ctx, "user_*").Result()

// ✅ Scan命令 - 增量式（推荐）
// 每次只返回100条，分批处理，不影响其他操作
var cursor uint64
for {
    keys, nextCursor, err := redisClient.Scan(ctx, cursor, "user_*", 100).Result()
    // 处理keys...
    cursor = nextCursor
    if cursor == 0 { break }
}
```

> ★当前实现了10个Redis方法。各方法用途：
> - SetKeyEx: 设置键值（带过期时间）
> - GetKey: 获取键值（不存在返回空字符串）
> - GetKeyNilIsErr: 获取键值（不存在返回redis.Nil错误）
> - GetKeyWithPrefixNilIsErr: 前缀查找单个key
> - GetKeyWithSuffixNilIsErr: 后缀查找单个key
> - DelKeyIfExists: 删除单个键
> - DelKeysWithPattern: 删除精确匹配pattern的key
> - DelKeysWithPrefix: 删除带前缀的所有键（使用Scan安全）
> - DelKeysWithSuffix: 删除带后缀的所有键（使用Scan安全）
> - DeleteAllRedisKeys: 清空所有Redis键

---

## 七、创建邮箱验证码服务

### 7.1 创建目录

```bash
mkdir -p internal/service/email
```

### 7.2 创建服务文件

文件路径：`internal/service/email/auth_code_service.go`

```go
package email

import (
	"crypto/tls"
	"net/smtp"
	"strconv"
	"time"

	"github.com/jordan-wright/email"

	"kama-chat-server/internal/config"
	"kama-chat-server/internal/service/redis"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/util/random"
)

// ============================================================
// SendVerificationCode - 发送邮箱验证码
// ============================================================
// 功能：
// 1. 检查是否已有未过期的验证码
// 2. 生成6位随机验证码
// 3. 存入Redis（5分钟过期）
// 4. 发送邮件
//
// 参数：
//   - toEmail: 目标邮箱地址
//
// 返回：
//   - string: 消息
//   - int: 状态码（0成功，-1系统错误，-2业务错误）

func SendVerificationCode(toEmail string) (string, int) {
	// 1. 检查Redis中是否已有验证码
	key := "email_code_" + toEmail
	code, err := redis.GetKey(key)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}

	// 2. 如果验证码已存在且未过期
	if code != "" {
		return "验证码已发送，请检查邮箱或5分钟后重试", -2
	}

	// 3. 生成6位随机验证码
	// ★GetRandomInt返回(int, error)，需要处理error
	codeInt, err := random.GetRandomInt(6)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}
	code = strconv.Itoa(codeInt)

	// 4. 存入Redis，5分钟过期
	err = redis.SetKeyEx(key, code, 5*time.Minute)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}

	// 5. 发送邮件
	err = sendEmail(toEmail, code)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}

	return "验证码已发送至 " + toEmail, 0
}

// ============================================================
// VerifyCode - 验证验证码
// ============================================================
// 参数：
//   - email: 邮箱地址
//   - inputCode: 用户输入的验证码
//
// 返回：
//   - string: 消息
//   - int: 状态码

func VerifyCode(email string, inputCode string) (string, int) {
	key := "email_code_" + email
	
	// 从Redis获取验证码
	storedCode, err := redis.GetKey(key)
	if err != nil {
		return constants.SYSTEM_ERROR, -1
	}

	// 验证码不存在或已过期
	if storedCode == "" {
		return "验证码已过期，请重新获取", -2
	}

	// 验证码不匹配
	if storedCode != inputCode {
		return "验证码错误", -2
	}

	// 验证成功，删除验证码
	redis.DelKeyIfExists(key)

	return "验证成功", 0
}

// ============================================================
// sendEmail - 发送邮件（内部函数）
// ============================================================
// ★重要：端口465是SSL端口，必须用SendWithTLS，不能用Send！

func sendEmail(toEmail string, code string) error {
	conf := config.GetConfig()

	// 创建邮件
	e := email.NewEmail()
	e.From = conf.EmailConfig.FromName + " <" + conf.EmailConfig.SmtpUsername + ">"
	e.To = []string{toEmail}
	e.Subject = "【KamaChat】邮箱验证码"
	e.HTML = []byte(buildEmailHTML(code))

	// ★端口465是SSL端口，需要用SendWithTLS
	addr := conf.EmailConfig.SmtpHost + ":" + strconv.Itoa(conf.EmailConfig.SmtpPort)
	auth := smtp.PlainAuth("", conf.EmailConfig.SmtpUsername, conf.EmailConfig.SmtpPassword, conf.EmailConfig.SmtpHost)

	// ★创建TLS配置（QQ邮箱SSL端口465必须配置）
	// InsecureSkipVerify: true - 跳过证书验证（避免证书问题）
	// ServerName: SMTP服务器地址
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         conf.EmailConfig.SmtpHost,
	}

	// 发送SSL加密邮件
	return e.SendWithTLS(addr, auth, tlsConfig)
}

// ============================================================
// buildEmailHTML - 构建邮件HTML内容
// ============================================================

func buildEmailHTML(code string) string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; background-color: #f5f5f5; padding: 20px;">
    <div style="max-width: 500px; margin: 0 auto; background: white; border-radius: 8px; padding: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
        <h2 style="color: #333; margin-bottom: 20px;">邮箱验证码</h2>
        <p style="color: #666; font-size: 14px;">您好，您正在验证邮箱，验证码如下：</p>
        <div style="background: #f8f9fa; border-radius: 4px; padding: 20px; text-align: center; margin: 20px 0;">
            <span style="font-size: 32px; font-weight: bold; color: #007bff; letter-spacing: 8px;">` + code + `</span>
        </div>
        <p style="color: #999; font-size: 12px;">验证码有效期5分钟，请勿泄露给他人。</p>
        <p style="color: #999; font-size: 12px; margin-top: 30px;">此邮件由系统自动发送，请勿回复。</p>
    </div>
</body>
</html>
`
}
```

> ⚠️ **重要提示**：
> 1. 端口 **465** 是 SSL 端口，必须用 `SendWithTLS()`，不能用 `Send()`
> 2. `SendWithTLS(nil)` 会导致空指针错误（panic），必须传入正确的 TLS 配置
> 3. `InsecureSkipVerify: true` 跳过证书验证，避免 SSL 证书问题

---

## 八、创建Controller接口

### 8.1 添加请求结构体

在 `internal/dto/request/user_info_request.go` 中添加：

```go
// SendEmailCodeRequest 发送邮箱验证码请求
type SendEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"` // 邮箱地址
}

// VerifyEmailCodeRequest 验证邮箱验证码请求
type VerifyEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"` // 邮箱地址
	Code  string `json:"code" binding:"required"`        // 验证码
}
```

### 8.2 添加Controller函数

在 `api/v1/user_info_controller.go` 中添加：

> ★重要：JsonBack 和 Controller 在同一个 `v1` 包，**直接调用 JsonBack()，不需要导入 https_server**！

```go
import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/email"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"

	"github.com/gin-gonic/gin"
)

// SendEmailCode 发送邮箱验证码
func SendEmailCode(c *gin.Context) {
	var req request.SendEmailCodeRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		JsonBack(c, constants.SYSTEM_ERROR, -1, nil)  // 直接调用，无需导入
		return
	}

	message, ret := email.SendVerificationCode(req.Email)
	JsonBack(c, message, ret, nil)  // 直接调用
}

// VerifyEmailCode 验证邮箱验证码
func VerifyEmailCode(c *gin.Context) {
	var req request.VerifyEmailCodeRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		JsonBack(c, constants.SYSTEM_ERROR, -1, nil)  // 直接调用
		return
	}

	message, ret := email.VerifyCode(req.Email, req.Code)
	JsonBack(c, message, ret, nil)  // 直接调用
}
```

### 8.3 注册路由

在 `internal/https_server/https_server.go` 的 `init()` 函数中添加路由：

```go
import (
	v1 "kama-chat-server/api/v1"  // 导入v1包
	// ... 其他导入
)

func init() {
	// ... 其他初始化代码

	// ★邮箱验证码路由（与其他路由一起注册）
	GE.POST("/user/sendEmailCode", v1.SendEmailCode)
	GE.POST("/user/verifyEmailCode", v1.VerifyEmailCode)
}
```

---

## 九、完整测试流程

### 9.1 启动服务

```bash
cd C:\Users\li\Desktop\kama-chat-server
go run cmd/kama-chat-server/main.go
```

### 9.2 发送验证码

**接口**：`POST http://127.0.0.1:8000/user/sendEmailCode`

**请求Body**：
```json
{
    "email": "your_email@example.com"
}
```

**成功响应**：
```json
{
    "code": 200,
    "message": "验证码已发送至 your_email@example.com"
}
```

### 9.3 验证验证码

**接口**：`POST http://127.0.0.1:8000/user/verifyEmailCode`

**请求Body**：
```json
{
    "email": "your_email@example.com",
    "code": "123456"
}
```

**成功响应**：
```json
{
    "code": 200,
    "message": "验证成功"
}
```

### 9.4 验证Redis数据

```bash
redis-cli

# 查看验证码
GET email_code_your_email@example.com

# 查看过期时间
TTL email_code_your_email@example.com
```

---

## 十、常见问题

### Q1: 邮件发送失败 "535 Login Fail"

**原因**：授权码错误或未开启SMTP服务

**解决方案**：
1. 确认已在QQ邮箱开启SMTP服务
2. 使用**授权码**而非QQ密码
3. 重新生成授权码并更新配置

### Q2: 邮件发送超时

**原因**：端口被防火墙拦截

**解决方案**：
1. 尝试端口587（TLS）
2. 检查防火墙设置
3. 确认网络可访问smtp.qq.com

### Q3: 邮件进入垃圾箱

**原因**：发件人名称或内容问题

**解决方案**：
1. 设置合理的发件人名称
2. 避免邮件内容过于简单
3. 使用HTML格式邮件

### Q4: 验证码收不到

**排查步骤**：
1. 检查邮箱地址是否正确
2. 查看垃圾邮件文件夹
3. 检查Redis中是否存储成功

### Q5: 程序崩溃 "nil pointer dereference"

**原因**：`SendWithTLS(addr, auth, nil)` 传入了 nil TLS 配置

**解决方案**：
```go
// ❌ 错误：传入 nil 导致崩溃
e.SendWithTLS(addr, auth, nil)

// ✅ 正确：传入 TLS 配置
tlsConfig := &tls.Config{
    InsecureSkipVerify: true,
    ServerName:         "smtp.qq.com",
}
e.SendWithTLS(addr, auth, tlsConfig)
```

---

## 十一、邮件效果预览

用户收到的邮件效果：

```
┌─────────────────────────────────────┐
│                                     │
│         邮箱验证码                   │
│                                     │
│   您好，您正在验证邮箱，验证码如下：  │
│                                     │
│   ┌─────────────────────────────┐   │
│   │         1 2 3 4 5 6         │   │
│   └─────────────────────────────┘   │
│                                     │
│   验证码有效期5分钟，请勿泄露给他人   │
│                                     │
│   此邮件由系统自动发送，请勿回复     │
│                                     │
└─────────────────────────────────────┘
```

---

## 十二、文件清单

```
kama-chat-server/
├── configs/
│   └── config_local.toml           # 添加emailConfig
├── internal/
│   ├── config/
│   │   └── config.go               # 添加EmailConfig
│   ├── service/
│   │   ├── redis/
│   │   │   └── redis_service.go    # Redis服务
│   │   └── email/
│   │       └── auth_code_service.go # 邮箱验证码服务
│   └── dto/request/
│       └── user_info_request.go    # 添加邮箱请求结构体
├── pkg/
│   ├── constants/
│   │   └── constants.go
│   └── util/random/
│       └── random_int.go           # 随机数工具
└── api/v1/
    ├── controller.go               # ★包含JsonBack函数
    └── user_info_controller.go     # 添加邮箱接口
```

---

## 十三、对比：邮箱 vs 短信

| 对比项 | 邮箱验证码 | 短信验证码 |
|--------|-----------|-----------|
| 费用 | 免费 | 每条约0.04元 |
| 配置复杂度 | 简单（授权码） | 复杂（AccessKey+签名+模板） |
| 审核要求 | 无 | 需要实名+审核 |
| 到达速度 | 较快 | 快 |
| 适用场景 | 注册、找回密码 | 登录、支付验证 |
| 开发测试 | 友好 | 需要正式环境 |

---

## 十四、下一步

完成邮箱验证码后，可以：

1. **实现邮箱注册** - 验证码验证后创建用户
2. **实现找回密码** - 通过邮箱重置密码
3. **集成到登录流程** - 邮箱+验证码登录
