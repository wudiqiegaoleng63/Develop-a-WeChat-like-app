package gorm

import (
	"encoding/json"
	"errors"
	"fmt"
	"kama-chat-server/internal/dao"
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/dto/respond"
	"kama-chat-server/internal/model"
	myredis "kama-chat-server/internal/service/redis"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/enum/contact/contact_status_enum"
	"kama-chat-server/pkg/enum/contact/contact_type_enum"
	"kama-chat-server/pkg/enum/group_info/group_status_enum"
	"kama-chat-server/pkg/util/random"
	"kama-chat-server/pkg/zlog"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type groupInfoService struct {
}

var GroupInfoService = new(groupInfoService)

// CreateGroup 创建群聊
func (g *groupInfoService) CreateGroup(groupReq request.CreateGroupRequest) (string, int) {
	value, er := random.GetNowAndLenRandomString(11)
	if er != nil {
		zlog.Error(er.Error())
		return constants.SYSTEM_ERROR, -1
	}
	
	group := model.GroupInfo{
		Uuid:      fmt.Sprintf("G%s", value),
		Name:      groupReq.Name,
		Notice:    groupReq.Notice,
		OwnerId:   groupReq.OwnerId,
		MemberCnt: 1,                          // 初始群人数为1（只有群主）
		AddMode:   groupReq.AddMode,
		Avatar:    groupReq.Avatar,
		Status:    group_status_enum.NORMAL,   // 状态: 正常
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	var members []string
	members = append(members, groupReq.OwnerId)


	var err error
	group.Members, err = json.Marshal(members)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 创建群聊记录
	if res := dao.GormDB.Create(&group); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	contact := model.UserContact{
		UserId:      groupReq.OwnerId,              // 群主的uuid
		ContactId:   group.Uuid,                    // 群聊的uuid
		ContactType: contact_type_enum.GROUP,       // 联系类型: 群聊
		Status:      contact_status_enum.NORMAL,    // 状态: 正常
		CreatedAt:   time.Now(),
		UpdateAt:    time.Now(),
	}

	if res := dao.GormDB.Create(&contact); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	if err := myredis.DelKeysWithPattern("contact_mygroup_list_" + groupReq.OwnerId);
	err != nil {
		zlog.Error(err.Error())
	}
	return "创建成功", 0
}


// LoadMyGroup 获取我创建的群聊
func (g *groupInfoService) LoadMyGroup(ownerId string) (string, []respond.LoadMyGroupRespond, int) {
	// 1.先读redis
	rspString, err := myredis.GetKeyNilIsErr("contact_mygroup_list_" + ownerId)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 2.数据库查询
			var groupList []model.GroupInfo

			if res := dao.GormDB.Order("created_at DESC").Where("owner_id = ?", ownerId).Find(&groupList); res.Error != nil {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}

			// 3.构建响应DTO
			var groupListRsp []respond.LoadMyGroupRespond
			for _, group := range groupList {
				groupListRsp = append(groupListRsp, respond.LoadMyGroupRespond{
					GroupId: group.Uuid,
					GroupName: group.Name,
					Avatar: group.Avatar,
				})
			}


			// 4.序列化写入缓存
			resString, err := json.Marshal(groupListRsp)

			if err != nil {
				zlog.Error(err.Error())
			}
			if err := myredis.SetKeyEx("contact_mygroup_list_"+ownerId, string(resString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(err.Error())
			}

			return "获取成功", groupListRsp, 0
		} else {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	// 5.缓存存在，直接反序列化返回
	var groupListRsp []respond.LoadMyGroupRespond
	if err := json.Unmarshal([]byte(rspString), &groupListRsp); err != nil {
		zlog.Error(err.Error())
	}

	return "获取成功", groupListRsp, 0
}




 // CheckGroupAddMode 检查群聊加群方式 0=直接加入, 1=需要审核，带Redis缓存
 func (g *groupInfoService) CheckGroupAddMode(groupId string) (string, int8, int){
	// ★1. 先从Redis缓存读取群聊详情
	rspString, err := myredis.GetKeyNilIsErr("group_info_" + groupId)
	if err != nil {
		// 缓存不存在，从数据库查询
		if errors.Is(err, redis.Nil) {
			// 2. 查询群聊信息
			var group model.GroupInfo
			if res := dao.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1, -1
			}

			// 3. 返回加群方式
			return "加群方式获取成功", group.AddMode, 0
		} else {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1, -1
		}
	}

	// ★4. 缓存存在，从缓存数据中解析AddMode
	var rsp respond.GetGroupInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
	}
	return "加群方式获取成功", rsp.AddMode, 0
}

// GetGroupInfo 获取群聊详情
// ★返回群聊详情DTO，带Redis缓存
func (g *groupInfoService) GetGroupInfo(groupId string) (string, *respond.GetGroupInfoRespond, int) {
	// ★1. 先从Redis缓存读取
	rspString, err := myredis.GetKeyNilIsErr("group_info_" + groupId)
	if err != nil {
		// 缓存不存在，从数据库查询
		if errors.Is(err, redis.Nil) {
			// 2. 查询群聊信息
			var group model.GroupInfo
			if res := dao.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}

			// 3. 构建响应DTO
			rsp := &respond.GetGroupInfoRespond{
				Uuid:      group.Uuid,
				Name:      group.Name,
				Notice:    group.Notice,
				Avatar:    group.Avatar,
				MemberCnt: group.MemberCnt,
				OwnerId:   group.OwnerId,
				AddMode:   group.AddMode,
				Status:    group.Status,
			}

			// ★4. 判断是否已解散（软删除）
			// DeletedAt.Valid为true表示已软删除
			if group.DeletedAt.Valid {
				rsp.IsDeleted = true
			} else {
				rsp.IsDeleted = false
			}

			// ★5. 写入Redis缓存（当前代码被注释，暂不启用）
			rspString, err := json.Marshal(rsp)
			if err != nil {
				zlog.Error(err.Error())
			}
			if err := myredis.SetKeyEx("group_info_"+groupId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(err.Error())
			}

			return "获取成功", rsp, 0
		} else {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	// ★6. 缓存存在，直接反序列化返回
	var rsp *respond.GetGroupInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
	}
	return "获取成功", rsp, 0
}

// EnterGroupDirectly 直接进群
// ownerId 是群聊id
func (g *groupInfoService) EnterGroupDirectly(ownerId, contactId string) (string, int) {
	//  1. 查询群聊信息
	var group model.GroupInfo
	if res := dao.GormDB.First(&group, "uuid=?", ownerId); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 2.json 序列化
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 3.添加新成员

	members = append(members, contactId)

	// 4.json序列化
	if data, err := json.Marshal(members); err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	} else {
		group.Members = data
	}


	// 5.更新数量
	group.MemberCnt += 1

	// 6.保存群基本信息
	if res := dao.GormDB.Save(&group); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 7.创建用户联系人记录
	newContact := model.UserContact{
        UserId:      contactId,
        ContactId:   ownerId,
        ContactType: contact_type_enum.GROUP,    // 群聊类型
        Status:      contact_status_enum.NORMAL, // 正常状态
        CreatedAt:   time.Now(),
        UpdateAt:    time.Now(),
    }
    if res := dao.GormDB.Create(&newContact); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 8. ★清除Redis缓存（带错误处理）
    if err := myredis.DelKeysWithPattern("group_session_list_" + ownerId); err != nil {
        zlog.Error(err.Error())
    }
    if err := myredis.DelKeysWithPattern("my_joined_group_list_" + ownerId); err != nil {
        zlog.Error(err.Error())
    }

    return "进群成功", 0
}


// LeaveGroup 退群
func (g *groupInfoService) LeaveGroup(userId string, groupId string) (string, int) {
    // 1. 查询群聊信息
    var group model.GroupInfo
    if res := dao.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 2. JSON反序列化：将Members字段转换为切片
    var members []string
    if err := json.Unmarshal(group.Members, &members); err != nil {
        zlog.Error(err.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 3. 从成员列表中删除该用户
    for i, member := range members {
        if member == userId {
            // ★核心：使用append删除切片元素
            members = append(members[:i], members[i+1:]...)
            break
        }
    }

    // 4. JSON序列化：将切片转换回JSON
    if data, err := json.Marshal(members); err != nil {
        zlog.Error(err.Error())
        return constants.SYSTEM_ERROR, -1
    } else {
        group.Members = data
    }

    // 5. 更新群成员数量
    group.MemberCnt -= 1

    // 6. 保存群聊信息
    if res := dao.GormDB.Save(&group); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 7. 软删除会话记录
    var deletedAt gorm.DeletedAt
    deletedAt.Time = time.Now()
    deletedAt.Valid = true
    if res := dao.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", userId, groupId).Update("deleted_at", deletedAt); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 8. 软删除联系人记录，并更新状态为"退群"
    if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", userId, groupId).Updates(map[string]interface{}{
        "deleted_at": deletedAt,
        "status":     contact_status_enum.QUIT_GROUP, // 退群状态
    }); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 9. 软删除申请记录
    if res := dao.GormDB.Model(&model.ContactApply{}).Where("contact_id = ? AND user_id = ?", groupId, userId).Update("deleted_at", deletedAt); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 10. ★清除Redis缓存（带错误处理）
    if err := myredis.DelKeysWithPattern("group_session_list_" + userId); err != nil {
        zlog.Error(err.Error())
    }
    if err := myredis.DelKeysWithPattern("my_joined_group_list_ " + userId); err != nil {
        zlog.Error(err.Error())
    }

    return "退群成功", 0
}

// DismissGroup 解散群聊
func (g *groupInfoService) DismissGroup(ownerId, groupId string) (string, int) {
    // 初始化软删除时间戳
    var deletedAt gorm.DeletedAt
    deletedAt.Time = time.Now()
    deletedAt.Valid = true

    // 1. 软删除群聊信息
    if res := dao.GormDB.Model(&model.GroupInfo{}).Where("uuid = ?", groupId).Updates(
        map[string]interface{}{
            "deleted_at": deletedAt,
            "updated_at": deletedAt.Time,
        }); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 2. 软删除所有相关会话记录
    var sessionList []model.Session
    if res := dao.GormDB.Model(&model.Session{}).Where("receive_id = ?", groupId).Find(&sessionList); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    for _, session := range sessionList {
        if res := dao.GormDB.Model(&session).Updates(
            map[string]interface{}{
                "deleted_at": deletedAt,
            }); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }
    }

    // 3. 软删除所有相关联系人记录
    var userContactList []model.UserContact
    if res := dao.GormDB.Model(&model.UserContact{}).Where("contact_id = ?", groupId).Find(&userContactList); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    for _, userContact := range userContactList {
        if res := dao.GormDB.Model(&userContact).Update("deleted_at", deletedAt); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }
    }

    // 4. 软删除所有相关申请记录
    var contactApplys []model.ContactApply
    if res := dao.GormDB.Model(&contactApplys).Where("contact_id = ?", groupId).Find(&contactApplys); res.Error != nil {
        if res.Error != gorm.ErrRecordNotFound {
            zlog.Info(res.Error.Error())
            return "无响应的申请记录需要删除", 0
        }
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    for _, contactApply := range contactApplys {
        if res := dao.GormDB.Model(&contactApply).Update("deleted_at", deletedAt); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }
    }

    // 5. ★清除Redis缓存（带错误处理）
    if err := myredis.DelKeysWithPattern("contact_mygroup_list_" + ownerId); err != nil {
        zlog.Error(err.Error())
    }
    if err := myredis.DelKeysWithPattern("group_session_list_" + ownerId); err != nil {
        zlog.Error(err.Error())
    }
    if err := myredis.DelKeysWithPrefix("my_joined_group_list"); err != nil {
        zlog.Error(err.Error())
    }

    return "解散群聊成功", 0
}
