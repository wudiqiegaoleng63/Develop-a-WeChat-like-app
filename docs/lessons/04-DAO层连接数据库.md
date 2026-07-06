# 教学文档 04: DAO层连接数据库

### 这是开发顺序的第二步：DAO层

```
Model → DAO → Service → Controller → 路由注册

① Model层: 数据结构已定义（03文档已完成）
② DAO层: ★当前步骤★ 数据库连接与GORM实例初始化
③ Service层: 业务逻辑实现
④ Controller层: HTTP请求处理
⑤ 路由注册: 在HTTP服务器中注册路由
```

## 一、DAO层职责

### 什么是DAO层？

DAO (Data Access Object) 层是数据访问层：
- 负责初始化数据库连接
- 提供全局的GORM实例
- 自动创建数据库表结构

### DAO层的文件

**文件位置:** `internal/dao/gorm.go`

---

## 二、完整代码（带详细注释）

```go
package dao

// ============================================================
// 导入依赖包
// ============================================================
import (
    "fmt"
    "time"
    
    // ★GORM核心库
    "gorm.io/gorm"
    
    // ★MySQL驱动
    "gorm.io/driver/mysql"
    
    // ★项目内部依赖
    "gochat/internal/config"
    "gochat/internal/model"
)

// ============================================================
// 全局GORM实例 - 单例模式
// ============================================================
// ★所有数据库操作都通过这个实例
// 声明为全局变量，其他文件可以直接使用 dao.GormDB
var GormDB *gorm.DB

// ============================================================
// init() 函数 - Go自动执行
// ============================================================
// ★Go语言特性：init()函数在main()之前自动执行
// 任何包只要有init()，导入时就会自动运行
func init() {
    // 1. 获取配置
    conf := config.GetConfig()
    
    // 2. 构建DSN (Data Source Name)
    // ★DSN格式: 用户名:密码@tcp(地址:端口)/数据库名?参数
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        conf.MysqlConfig.User,         // 用户名
        conf.MysqlConfig.Password,     // 密码
        conf.MysqlConfig.Host,         // 地址
        conf.MysqlConfig.Port,         // 端口
        conf.MysqlConfig.DatabaseName, // 数据库名
    )
    
    // 3. 连接数据库
    // ★gorm.Open(驱动, 配置)
    GormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        // 连接失败是致命错误，直接panic
        panic("数据库连接失败: " + err.Error())
    }
    
    // 4. 配置连接池
    // ★获取底层sql.DB来设置连接池参数
    sqlDB, err := GormDB.DB()
    if err != nil {
        panic("获取sql.DB失败: " + err.Error())
    }
    
    // 设置连接池参数
    sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
    sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
    sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大存活时间
    
    // 5. 自动迁移表结构
    // ★AutoMigrate: 根据模型自动创建/更新表
    // 只需要传入模型结构体，GORM会自动创建对应的表
    GormDB.AutoMigrate(
        &model.UserInfo{},       // 用户表
        &model.GroupInfo{},      // 群组表
        &model.UserContact{},    // 联系人表
        &model.ContactApply{},   // 申请表
        &model.Session{},        // 会话表
        &model.Message{},        // 消息表
    )
    
    fmt.Println("数据库连接成功，表结构已创建")
}
```

---

## 三、DSN详解

### DSN格式

```
用户名:密码@tcp(地址:端口)/数据库名?参数
```

### 参数说明

| 参数 | 作用 | 说明 |
|------|------|------|
| `charset=utf8mb4` | 字符集 | utf8mb4支持中文和emoji |
| `parseTime=True` | 时间解析 | 自动将DATETIME转为time.Time |
| `loc=Local` | 时区 | 使用本地时区 |

### DSN示例

```go
// 根据你的配置文件
dsn := "root:Li2006116.@tcp(127.0.0.1:3306)/gochat?charset=utf8mb4&parseTime=True&loc=Local"
```

---

## 四、init()函数详解

### Go的init()机制

```go
// 任何包都可以有init()函数
package dao

func init() {
    // 这段代码在main()之前自动执行
}

// 执行顺序:
// 1. 导入包 → 2. 初始化全局变量 → 3. 执行init() → 4. 执行main()
```

### 为什么用init()？

```go
// 不用init() (错误做法)
// main.go需要手动调用
func main() {
    dao.ConnectDB()  // 手动调用
    dao.CreateTable()
}

// 用init() (正确做法)
// 导入dao包就自动连接
func main() {
    // dao.GormDB已经可用，无需手动调用
    dao.GormDB.First(&user)
}
```

---

## 五、AutoMigrate详解

### AutoMigrate的作用

```go
GormDB.AutoMigrate(&model.UserInfo{})
```

做了什么：
1. 检查表是否存在
2. 不存在 → 创建表
3. 存在 → 检查字段是否匹配，不匹配则添加新字段

### 生成的SQL

```sql
CREATE TABLE `user_info` (
    `id` bigint PRIMARY KEY AUTO_INCREMENT,
    `uuid` char(20) UNIQUE,
    `telephone` char(11),
    `nickname` varchar(20) NOT NULL,
    `created_at` datetime,
    `deleted_at` datetime,
    -- 其他字段...
)
```

### 注意事项

```go
// ★AutoMigrate只会添加字段，不会删除字段
// 如果你在模型中删除了一个字段，数据库中的列不会自动删除

// ★AutoMigrate不会修改字段类型
// 如果修改了字段类型，需要手动修改数据库
```

---

## 六、连接池参数详解

### 为什么需要连接池？

```
没有连接池:
请求1 → 创建连接 → 执行SQL → 关闭连接 (耗时)
请求2 → 创建连接 → 执行SQL → 关闭连接 (耗时)
...

有连接池:
请求1 → 从池中获取空闲连接 → 执行SQL → 放回池中 (快)
请求2 → 从池中获取空闲连接 → 执行SQL → 放回池中 (快)
...
```

### 参数含义

```go
sqlDB.SetMaxIdleConns(10)    // 池中最多保留10个空闲连接
sqlDB.SetMaxOpenConns(100)   // 最多同时打开100个连接
sqlDB.SetConnMaxLifetime(time.Hour) // 连接存活1小时后自动关闭
```

---

## 七、使用GormDB

### 在其他文件中使用

```go
package service

import "gochat/internal/dao"

func GetUser(uuid string) {
    var user model.UserInfo
    
    // ★直接使用 dao.GormDB
    dao.GormDB.First(&user, "uuid = ?", uuid)
}
```

### 常用查询方法

```go
// 查询第一条
dao.GormDB.First(&user, "uuid = ?", "U12345678901")

// 查询所有
dao.GormDB.Find(&users)

// 条件查询
dao.GormDB.Where("status = ?", 0).Find(&users)

// 创建记录
dao.GormDB.Create(&newUser)

// 更新记录
dao.GormDB.Save(&user)

// 软删除 (设置deleted_at)
dao.GormDB.Delete(&user)

// 查询包含软删除的数据
dao.GormDB.Unscoped().Find(&users)
```

---

## 八、创建文件步骤

### 步骤1: 创建目录

```bash
mkdir internal/dao
```

### 步骤2: 创建文件

创建 `internal/dao/gorm.go`，写入上面的完整代码

### 步骤3: 安装依赖

```bash
go get gorm.io/gorm
go get gorm.io/driver/mysql
```

### 步骤4: 验证连接

在 `main.go` 中导入dao包：

```go
package main

import (
    "fmt"
    "gochat/internal/dao"  // ★导入就会自动执行init()
)

func main() {
    // dao.GormDB已经可用
    fmt.Println("GormDB:", dao.GormDB)
}
```

运行：
```bash
go run cmd/gochat/main.go
```

如果输出 "数据库连接成功，表结构已创建"，说明连接成功。

---

## 九、常见错误

### 错误1: 密码错误

```
Error: Access denied for user 'root'@'localhost'
```

解决：检查config_local.toml中的password是否正确

### 错误2: 数据库不存在

```
Error: Unknown gochat
```

解决：先在MySQL中创建数据库
```sql
CREATE DATABASE gochat CHARACTER SET utf8mb4;
```

### 错误3: MySQL未启动

```
Error: Can't connect to MySQL server on '127.0.0.1'
```

解决：启动MySQL服务

---

## 十、下一步

DAO层完成后，继续学习：
- **06-HTTP服务器与路由.md** - Gin框架和路由注册
- **07-日志系统.md** - Zap日志库实现