package model

import (
	"time"
    "gorm.io/gorm"
)

type Session struct {
    Id            int64          `gorm:"column:id;primaryKey;comment:自增id"`
    Uuid          string         `gorm:"column:uuid;uniqueIndex;type:char(20);comment:会话uuid"`
    SendId        string         `gorm:"column:send_id;index;type:char(20);not null;comment:创建会话人id"`
    ReceiveId     string         `gorm:"column:receive_id;index;type:char(20);not null;comment:接受会话人id"`
    ReceiveName   string         `gorm:"column:receive_name;type:varchar(20);not null;comment:名称"`
    Avatar        string         `gorm:"column:avatar;type:char(255);default:default_avatar.png;not null;comment:头像"`
    CreatedAt     time.Time      `gorm:"column:created_at;index;type:datetime;comment:创建时间"`
    DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index;type:datetime;comment:删除时间"`
}