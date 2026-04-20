package dao

import (
	"fmt"
	"kama-chat-server/internal/config"
	"kama-chat-server/internal/model"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 数据库链接实例
var GormDB *gorm.DB

func init() {
	// 获取配置
	conf := config.GetConfig()

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
	conf.MysqlConfig.User,
	conf.MysqlConfig.Password,
	conf.MysqlConfig.Host,
	conf.MysqlConfig.Port,
	conf.MysqlConfig.DatabaseName,
)

	// 连接
	var err error
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败：" + err.Error())
	}

	// 配置连接池
	sqlDB, err := GormDB.DB()
	if err != nil {
		panic("获取sql.DB失败:" + err.Error())
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移
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