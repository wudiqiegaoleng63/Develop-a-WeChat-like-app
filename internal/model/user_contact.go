package model

import (
	"time"
    "gorm.io/gorm"
)

type UserContact struct {
    Id          int64          `gorm:"column:id;primaryKey;comment:自增id"`
    UserId      string         `gorm:"column:user_id;index;type:char(20);not null;comment:用户唯一id"`
    ContactId   string         `gorm:"column:contact_id;index;type:char(20);not null;comment:对应联系id"`
    ContactType int8           `gorm:"column:contact_type;not null;comment:联系类型，0.用户，1.群聊"`
    Status      int8           `gorm:"column:status;not null;comment:联系状态，0.正常，1.拉黑，2.被拉黑，3.删除好友..."`
    CreatedAt   time.Time      `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
    DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index;comment:删除时间"`
}