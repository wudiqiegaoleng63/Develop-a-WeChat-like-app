# 教学文档 08: HTTP服务器与路由

## 前言：为什么路由注册放在最后？

按照开发顺序：**Model → DAO → Service → Controller → 路由注册**

路由注册必须放在最后，因为：
1. 路由需要引用Controller函数（如 `v1.Login`）
2. Controller依赖Service层
3. Service依赖DAO和Model层

如果先写路由再写Controller，代码无法编译。

---

## 一、Gin框架简介

### 什么是Gin？

Gin是Go最流行的Web框架：
- 高性能HTTP路由
- 中间件支持
- JSON绑定和响应

### 安装Gin

```bash
go get github.com/gin-gonic/gin
```

---

## 二、HTTP服务器文件

**文件位置:** `internal/https_server/https_server.go`

根据文档 `4.后端开发.md` 行8-35：

### 完整代码（带详细注释）

```go
package https_server

// ============================================================
// 导入依赖包
// ============================================================
import (
    "net/http"
    
    // ★Gin框架
    "github.com/gin-gonic/gin"
    
    // ★CORS跨域中间件
    "github.com/gin-contrib/cors"
    
    // ★项目内部依赖
    "kama-chat-server/internal/config"
)

// ============================================================
// 全局Gin引擎
// ============================================================
// ★GE是全局的Gin实例，所有路由都注册到这个引擎上
var GE *gin.Engine

// ============================================================
// init() 函数 - 自动注册所有路由
// ============================================================
func init() {
    // 1. 创建Gin引擎
    // gin.Default() 默认带有Logger和Recovery中间件
    GE = gin.Default()
    
    // 2. CORS跨域配置
    // ★前端和后端不在同域，需要配置跨域
    corsConfig := cors.DefaultConfig()
    corsConfig.AllowOrigins = []string{"*"}  // 允许所有来源
    corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
    corsConfig.AllowHeaders = []string{"Origin", "Content-Type"}
    GE.Use(cors.New(corsConfig))
    
    // 3. 静态文件服务
    // ★让前端可以访问上传的文件
    conf := config.GetConfig()
    GE.Static("/static/avatars", conf.StaticSrcConfig.StaticAvatarPath)
    GE.Static("/static/files", conf.StaticSrcConfig.StaticFilePath)
    
    // 4. 注册所有路由
    registerRoutes()
}

// ============================================================
// registerRoutes - 注册所有API路由
// ============================================================
func registerRoutes() {
    // ----------------------
    // 用户相关路由 (POST)
    // ----------------------
    GE.POST("/login", v1.Login)                        // 登录
    GE.POST("/register", v1.Register)                  // 注册
    GE.POST("/user/updateUserInfo", v1.UpdateUserInfo) // 更新信息
    GE.POST("/user/getUserInfoList", v1.GetUserInfoList) // 用户列表
    GE.POST("/user/ableUsers", v1.AbleUsers)           // 启用用户
    GE.POST("/user/getUserInfo", v1.GetUserInfo)       // 获取信息
    GE.POST("/user/disableUsers", v1.DisableUsers)     // 禁用用户
    GE.POST("/user/deleteUsers", v1.DeleteUsers)       // 删除用户
    GE.POST("/user/setAdmin", v1.SetAdmin)             // 设置管理员
    GE.POST("/user/sendSmsCode", v1.SendSmsCode)       // 发送验证码
    GE.POST("/user/smsLogin", v1.SmsLogin)             // 验证码登录
    GE.POST("/user/wsLogout", v1.WsLogout)              // WebSocket登出（详见文档19）
    
    // ----------------------
    // 群组相关路由 (POST)
    // ----------------------
    GE.POST("/group/createGroup", v1.CreateGroup)      // 创建群组
    GE.POST("/group/loadMyGroup", v1.LoadMyGroup)      // 我的群组
    GE.POST("/group/checkGroupAddMode", v1.CheckGroupAddMode) // 查看加群方式
    GE.POST("/group/enterGroupDirectly", v1.EnterGroupDirectly) // 直接进群
    GE.POST("/group/leaveGroup", v1.LeaveGroup)        // 退群
    GE.POST("/group/dismissGroup", v1.DismissGroup)    // 解散群组
    GE.POST("/group/getGroupInfo", v1.GetGroupInfo)    // 群组信息
    GE.POST("/group/getGroupInfoList", v1.GetGroupInfoList) // ★管理员: 群组列表(含软删除)
    GE.POST("/group/deleteGroups", v1.DeleteGroups)    // ★管理员: 删除群组
    GE.POST("/group/setGroupsStatus", v1.SetGroupsStatus) // ★管理员: 设置群组状态
    GE.POST("/group/updateGroupInfo", v1.UpdateGroupInfo) // 更新群组
    GE.POST("/group/getGroupMemberList", v1.GetGroupMemberList) // 成员列表
    GE.POST("/group/removeGroupMembers", v1.RemoveGroupMembers) // 移除成员
    
    // ----------------------
    // 会话相关路由 (POST)
    // ----------------------
    GE.POST("/session/openSession", v1.OpenSession)    // 打开会话
    GE.POST("/session/getUserSessionList", v1.GetUserSessionList) // 用户会话列表
    GE.POST("/session/getGroupSessionList", v1.GetGroupSessionList) // ★群聊会话列表
    GE.POST("/session/deleteSession", v1.DeleteSession) // 删除会话
    GE.POST("/session/checkOpenSessionAllowed", v1.CheckOpenSessionAllowed) // ★检查是否允许开会话
    
    // ----------------------
    // 联系人相关路由 (POST)
    // ----------------------
    GE.POST("/contact/getUserList", v1.GetUserList)    // 联系人列表
    GE.POST("/contact/loadMyJoinedGroup", v1.LoadMyJoinedGroup) // ★我加入的群组列表
    GE.POST("/contact/getContactInfo", v1.GetContactInfo) // ★获取联系人详细信息
    GE.POST("/contact/deleteContact", v1.DeleteContact) // 删除联系人
    GE.POST("/contact/applyContact", v1.ApplyContact)  // 申请联系人
    GE.POST("/contact/getNewContactList", v1.GetNewContactList) // 新申请列表
    GE.POST("/contact/passContactApply", v1.PassContactApply) // 通过申请
    GE.POST("/contact/blackContact", v1.BlackContact)  // 拉黑
    GE.POST("/contact/cancelBlackContact", v1.CancelBlackContact) // ★取消拉黑
    GE.POST("/contact/getAddGroupList", v1.GetAddGroupList) // ★获取加群申请列表
    GE.POST("/contact/refuseContactApply", v1.RefuseContactApply) // 拒绝申请
    GE.POST("/contact/blackApply", v1.BlackApply)      // ★拉黑申请人
    
    // ----------------------
    // 消息相关路由 (POST)
    // ----------------------
    GE.POST("/message/getMessageList", v1.GetMessageList) // 消息列表
    GE.POST("/message/getGroupMessageList", v1.GetGroupMessageList) // ★群聊消息列表
    GE.POST("/message/uploadAvatar", v1.UploadAvatar)  // 上传头像
    GE.POST("/message/uploadFile", v1.UploadFile)      // 上传文件
    
    // ----------------------
    // 聊天室相关路由 (POST)
    // ----------------------
    GE.POST("/chatroom/getCurContactListInChatRoom", v1.GetCurContactListInChatRoom) // ★聊天室联系人列表
    
    // ----------------------
    // WebSocket路由 (GET)
    // ★WebSocket必须是GET请求
    // ----------------------
    GE.GET("/wss", v1.WsLogin)                         // WebSocket登录
}

// ============================================================
// RunServer - 启动HTTP服务器
// ============================================================
func RunServer() {
    conf := config.GetConfig()
    
    // 拼接地址: 127.0.0.1:8000
    addr := conf.MainConfig.Host + ":" + strconv.Itoa(conf.MainConfig.Port)
    
    // ★GE.Run() 启动服务器，阻塞运行
    GE.Run(addr)
}
```

---

## 三、统一响应格式

### JsonBack函数

根据文档 `4.后端开发.md` 行391-421：

> ★重要：**JsonBack 定义在 `api/v1/controller.go`**，和 Controller 在同一个包！
> 这样 Controller 可以直接调用 JsonBack，不需要导入其他包，避免循环导入问题。

**文件位置:** `api/v1/controller.go`

```go
package v1

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// ============================================================
// JsonBack - 统一响应格式
// ============================================================
// ★所有API都用这个函数返回响应
// 参数:
//   c: Gin上下文
//   message: 提示信息（如"登录成功"、"密码错误")
//   ret: 返回状态码（0成功, -2业务失败, -1系统错误）
//   data: 返回的数据（如用户信息）
func JsonBack(c *gin.Context, message string, ret int, data interface{}) {
    if ret == 0 {
        // ★业务成功: code=200
        if data != nil {
            c.JSON(http.StatusOK, gin.H{
                "code":    200,
                "message": message,
                "data":    data,
            })
        } else {
            c.JSON(http.StatusOK, gin.H{
                "code":    200,
                "message": message,
            })
        }
    } else if ret == -2 {
        // ★业务失败: code=400（如密码错误、用户不存在）
        c.JSON(http.StatusOK, gin.H{
            "code":    400,
            "message": message,
        })
    } else if ret == -1 {
        // ★系统错误: code=500（如数据库连接失败）
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": message,
        })
    }
}
```

### ret返回值含义

| ret值 | code | 说明 | 示例 |
|-------|------|------|------|
| 0 | 200 | 业务成功 | 登录成功、注册成功 |
| -2 | 400 | 业务失败 | 密码错误、用户不存在 |
| -1 | 500 | 系统错误 | Redis连接失败、数据库错误 |

### 使用示例

```go
// 成功响应（同包内直接调用，无需导入）
JsonBack(c, "登录成功", 0, userInfo)

// 业务失败响应
JsonBack(c, "密码不正确", -2, nil)

// 系统错误响应
JsonBack(c, "数据库错误", -1, nil)
```

---

## 四、Controller层结构

### Controller文件位置

```
api/v1/
├── user_info_controller.go   # 用户相关API
├── group_info_controller.go  # 群组相关API
├── session_controller.go     # 会话相关API
├── contact_controller.go     # 联系人相关API
├── message_controller.go     # 消息相关API
└── ws_controller.go          # WebSocket相关
```

### Controller代码模板

根据文档 `4.后端开发.md` 行258-273：

> ★重要：Controller 和 JsonBack 都在 `api/v1` 包，**不需要导入 https_server**！
> 直接调用 `JsonBack()` 即可。

```go
package v1

import (
    "github.com/gin-gonic/gin"
    "kama-chat-server/internal/dto/request"
    "kama-chat-server/internal/service/gorm"
    "kama-chat-server/pkg/zlog"
    "kama-chat-server/pkg/constants"
)

// ============================================================
// Login - 登录接口
// ============================================================
func Login(c *gin.Context) {
    // 1. 绑定请求参数
    var req request.LoginRequest
    // ★c.BindJSON: 将请求JSON绑定到结构体
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        // ★同包内直接调用JsonBack，无需导入
        JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
        return
    }

    // 2. 调用Service层处理业务逻辑
    message, userInfo, ret := gorm.UserInfoService.Login(req)

    // 3. 返回响应（同包内直接调用）
    JsonBack(c, message, ret, userInfo)
}
```

> ⚠️ **避免循环导入**：如果 Controller 导入 `https_server`，而 `https_server` 导入 `api/v1`，
> 就会形成循环导入，编译报错。所以 JsonBack 必须放在 `api/v1` 包！

---

## 五、Request结构体

### 请求数据结构位置

`internal/dto/request/user_info_request.go`

> ★注意：参考文档将 request/respond 放在 `internal/dto/` 目录下，DTO = Data Transfer Object

### LoginRequest示例

```go
package request

// LoginRequest - 登录请求结构
// ★JSON请求体 {"telephone": "12345678901", "password": "123456"}
type LoginRequest struct {
    Telephone string `json:"telephone" binding:"required"` // binding:required表示必填
    Password  string `json:"password" binding:"required"`
}

// RegisterRequest - 注册请求结构
type RegisterRequest struct {
    Telephone string `json:"telephone" binding:"required"`
    Password  string `json:"password" binding:"required"`
    Nickname  string `json:"nickname" binding:"required"`
    SmsCode   string `json:"smsCode" binding:"required"`   // 验证码
}
```

### `json:"xxx"` 标签的作用

```go
Telephone string `json:"telephone"`
```

| 不写标签 | 写标签 | 结果 |
|---------|-------|------|
| 用结构体字段名 | 强制匹配指定名称 | JSON中的 `telephone` 对应 Go中的 `Telephone` |

---

## 六、CORS跨域配置

### 什么是跨域？

```
前端地址: http://localhost:3000
后端地址: http://localhost:8000

浏览器安全策略：不允许前端直接请求不同源的后端
```

### CORS配置

```go
corsConfig := cors.DefaultConfig()
corsConfig.AllowOrigins = []string{"*"}  // 允许所有来源
GE.Use(cors.New(corsConfig))
```

---

## 七、路由注册详解

### GE.POST 和 GE.GET

```go
// POST请求（大部分API）
GE.POST("/login", v1.Login)

// GET请求（WebSocket必须是GET）
GE.GET("/wss", v1.WsLogin)
```

### 静态文件路由

```go
// 让前端访问上传的文件
GE.Static("/static/avatars", "./static/avatars")

// 前端访问: http://localhost:8000/static/avatars/avatar.png
// 实际文件: ./static/avatars/avatar.png
```

---

## 八、main.go入口

**文件位置:** `cmd/kama-chat-server/main.go`

```go
package main

import (
    // ★导入所有包，触发init()自动执行
    "kama-chat-server/internal/config"    // 配置加载
    "kama-chat-server/internal/dao"       // 数据库连接
    "kama-chat-server/internal/https_server" // HTTP服务器
)

func main() {
    // ★导入包后，各包的init()已经自动执行:
    // 1. config.init() → 加载配置
    // 2. dao.init() → 连接数据库、创建表
    // 3. https_server.init() → 注册路由
    
    // 启动HTTP服务器（阻塞运行）
    https_server.RunServer()
}
```

---

## 九、创建文件步骤

### 步骤1: 创建目录

```bash
mkdir internal/https_server
mkdir api/v1
mkdir internal/dto/request
mkdir internal/dto/respond
```

### 步骤2: 创建文件

创建以下文件：
- `api/v1/controller.go` - ★包含 JsonBack 函数
- `internal/https_server/https_server.go` - 路由注册
- `cmd/kama-chat-server/main.go` - 入口文件

### 步骤3: 安装依赖

```bash
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors
```

---

## 十、下一步

HTTP服务器与路由注册完成后，继续学习：
- **10-用户管理接口.md** - 用户信息更新和管理员功能
- **11-群组创建与管理.md** - 群组创建功能