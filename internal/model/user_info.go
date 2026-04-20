package model

import (
    "time"
    "gorm.io/gorm"
)

// UserInfo表

type UserInfo struct {
	Id            int64          `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid          string         `gorm:"column:uuid;uniqueIndex;type:char(20);comment:用户唯一id"`
	Nickname      string         `gorm:"column:nickname;type:varchar(20);not null;comment:昵称"`
	Telephone     string         `gorm:"column:telephone;index;not null;type:char(11);comment:电话"`
	Email         string         `gorm:"column:email;type:char(30);comment:邮箱"`
	Avatar        string         `gorm:"column:avatar;type:char(255);default:https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png;not null;comment:头像"`
	Gender        int8           `gorm:"column:gender;comment:性别，0.男，1.女"`
	Signature     string         `gorm:"column:signature;type:varchar(100);comment:个性签名"`
	Password      string         `gorm:"column:password;type:char(18);not null;comment:密码"`
	Birthday      string         `gorm:"column:birthday;type:char(8);comment:生日"`
	CreatedAt     time.Time      `gorm:"column:created_at;index;type:datetime;not null;comment:创建时间"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;comment:删除时间"`
	IsAdmin       int8           `gorm:"column:is_admin;not null;comment:是否是管理员，0.不是，1.是"`
	Status        int8           `gorm:"column:status;not null;comment:状态，0.正常，1.禁用"`
}