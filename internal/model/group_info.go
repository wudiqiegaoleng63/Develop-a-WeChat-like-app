package model

import (
	"time"
    "gorm.io/gorm"
	"encoding/json"
)

type GroupInfo struct {
    Id        int64           `gorm:"column:id;primaryKey;comment:自增id"`
    Uuid      string          `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:群组唯一id"`
    Name      string          `gorm:"column:name;type:varchar(20);not null;comment:群名称"`
    Notice    string          `gorm:"column:notice;type:varchar(500);comment:群公告"`

    // ★json.RawMessage：GORM不支持[]string，用JSON存储
    Members   json.RawMessage `gorm:"column:members;type:json;comment:群组成员"`

    MemberCnt int             `gorm:"column:member_cnt;default:1;comment:群人数"`
    OwnerId   string          `gorm:"column:owner_id;type:char(20);not null;comment:群主uuid"`
    AddMode   int8            `gorm:"column:add_mode;default:0;comment:加群方式，0.直接，1.审核"`
    Avatar    string          `gorm:"column:avatar;type:char(255);default:https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png; not null;comment:头像"`
    Status    int8            `gorm:"column:status;default:0;comment:状态，0.正常，1.禁用，2.解散"`
    CreatedAt time.Time       `gorm:"column:created_at;index;type:datetime;not null;comment:创建时间"`
    DeletedAt gorm.DeletedAt  `gorm:"column:deleted_at;index;comment:删除时间"`
}