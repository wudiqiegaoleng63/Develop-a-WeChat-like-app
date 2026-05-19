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
	"log"
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


// checkGroupOwner 校验用户是否为群主
func (g *groupInfoService) checkGroupOwner(ownerId, groupId string) (string, int) {
	var group model.GroupInfo
	if res := dao.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return "群聊不存在", -2
		}
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if group.OwnerId != ownerId {
		return "只有群主才能执行此操作", -2
	}
	return "", 0
}

// DismissGroup 解散群聊
func (g *groupInfoService) DismissGroup(ownerId, groupId string) (string, int) {
	// 校验群主身份
	if msg, ret := g.checkGroupOwner(ownerId, groupId); ret != 0 {
		return msg, ret
	}

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


// UpdateGroupInfo 更新群聊消息
func (g *groupInfoService) UpdateGroupInfo(req request.UpdateGroupInfoRequest) (string, int) {
    // 1. 查询群组信息
    var group model.GroupInfo
    if res := dao.GormDB.First(&group, "uuid = ?", req.Uuid); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 2. 校验群主身份
    if group.OwnerId != req.OwnerId {
        return "只有群主才能执行此操作", -2
    }

    // 3. 选择性更新群组字段
    if req.Name != "" {
        group.Name = req.Name
    }
    if req.AddMode != -1 {
        group.AddMode = req.AddMode
    }
    if req.Notice != "" {
        group.Notice = req.Notice
    }
    if req.Avatar != "" {
        group.Avatar = req.Avatar
    }

    // 3. 保存群组更新
    if res := dao.GormDB.Save(&group); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // ★4. 同步更新Session表（关键步骤）
    var sessionList []model.Session
    if res := dao.GormDB.Where("receive_id = ?", req.Uuid).Find(&sessionList); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    for _, session := range sessionList {
        session.ReceiveName = group.Name  // 更新会话显示的群名称
        session.Avatar = group.Avatar     // 更新会话显示的群头像
        log.Println(session)
        if res := dao.GormDB.Save(&session); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }
    }

    // 5. Redis缓存清理
	if err := myredis.DelKeysWithPattern("group_info_" + req.Uuid); err != nil {
		zlog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPattern("contact_mygroup_list_" + req.OwnerId); err != nil {
		zlog.Error(err.Error())
	}

    return "更新成功", 0
}


// GetGroupMemberList 获取群聊成员列表
func (g *groupInfoService)GetGroupMemberList(groupId string) (string, []respond.GetGroupMemberListRespond, int) {
	// 先读redis
	rspString, err := myredis.GetKeyNilIsErr("group_memberlist_" + groupId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 2. 数据库查群组
			var group model.GroupInfo

			if res := dao.GormDB.First(&group, "uuid=?", groupId); res.Error != nil {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}

			// 3.解析
			var members []string
			if err := json.Unmarshal(group.Members, &members); err != nil {
				zlog.Error(err.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}

			// 4。遍历UUid
			var rspList []respond.GetGroupMemberListRespond
			for _, member := range members {
				var user model.UserInfo
				if res := dao.GormDB.First(&user, "uuid=?", member); res.Error != nil {
					zlog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, nil , -1
				}

				rspList = append(rspList, respond.GetGroupMemberListRespond{
					UserId: user.Uuid,
					Nickname: user.Nickname,
					Avatar: user.Avatar,
				})
			}

			// 写入redis
			rspString, err := json.Marshal(rspList)
			if err != nil {
				zlog.Error(err.Error())
			}

			if err := myredis.SetKeyEx("group_memberlist_"+groupId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
               zlog.Error(err.Error())
            }

			return "获取群聊成员列表成功", rspList, 0
		} else {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	// 6.缓存命中，直接返回
	var rsp []respond.GetGroupMemberListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
	}

	return "获取群聊成员列表成功", rsp, 0
}

// RemoveGroupMembers 移除群聊成员
func (g *groupInfoService) RemoveGroupMembers(req request.RemoveGroupMembersRequest) (string, int) {
	// 校验群主身份
	if msg, ret := g.checkGroupOwner(req.OwnerId, req.GroupId); ret != 0 {
		return msg, ret
	}

	// 1. 查询群组信息
    var group model.GroupInfo
    if res := dao.GormDB.First(&group, "uuid = ?", req.GroupId); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 2. 解析Members字段
    var members []string
    if err := json.Unmarshal(group.Members, &members); err != nil {
        zlog.Error(err.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 3. 构造软删除时间戳
    var deletedAt gorm.DeletedAt
    deletedAt.Time = time.Now()
    deletedAt.Valid = true
    log.Println(req.UuidList, req.OwnerId)

    // ★4. 遍历待移除成员列表
    for _, uuid := range req.UuidList {
        // ★关键校验：不能移除群主
        if req.OwnerId == uuid {
            return "不能移除群主", -2
        }

        // 5. 从members切片中移除该UUID
        filtered := make([]string, 0, len(members))
        for _, member := range members {
            if member != uuid {
                filtered = append(filtered, member)
            }
        }
        members = filtered
        group.MemberCnt -= 1  // 减少成员计数

        // ★6. 删除会话记录（Session表）
        if res := dao.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", uuid, req.GroupId).Update("deleted_at", deletedAt); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }

        // ★7. 删除联系人记录（UserContact表）
        if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", uuid, req.GroupId).Update("deleted_at", deletedAt); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }

        // 8. 删除申请记录（ContactApply表）
        if res := dao.GormDB.Model(&model.ContactApply{}).Where("user_id = ? AND contact_id = ?", uuid, req.GroupId).Update("deleted_at", deletedAt); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }
    }

    // 9. 更新群组的Members字段
    group.Members, _ = json.Marshal(members)
    if res := dao.GormDB.Save(&group); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    // 10. 清理Redis缓存
    if err := myredis.DelKeysWithPrefix("group_session_list"); err != nil {
        zlog.Error(err.Error())
    }
    if err := myredis.DelKeysWithPrefix("my_joined_group_list"); err != nil {
        zlog.Error(err.Error())
    }

    return "移除群聊成员成功", 0
}


// GetGroupInfoList 获取群聊列表 - 管理员
func (g *groupInfoService) GetGroupInfoList() (string, []respond.GetGroupListRespond, int) {
    var groupList []model.GroupInfo
    // ★Unscoped()：查询包含软删除记录
    if res := dao.GormDB.Unscoped().Find(&groupList); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, nil, -1
    }
    var rsp []respond.GetGroupListRespond
    for _, group := range groupList {
        rp := respond.GetGroupListRespond{
            Uuid:    group.Uuid,
            Name:    group.Name,
            OwnerId: group.OwnerId,
            Status:  group.Status,
        }
        // ★判断是否已软删除
        if group.DeletedAt.Valid {
            rp.IsDeleted = true
        } else {
            rp.IsDeleted = false
        }
        rsp = append(rsp, rp)
    }
    return "获取成功", rsp, 0
}

// DeleteGroups 删除列表中群聊 - 管理员
func (g *groupInfoService) DeleteGroups(uuidList []string) (string, int) {
    for _, uuid := range uuidList {
        var deletedAt gorm.DeletedAt
        deletedAt.Time = time.Now()
        deletedAt.Valid = true

        // 1. 软删除群聊
        if res := dao.GormDB.Model(&model.GroupInfo{}).Where("uuid = ?", uuid).Update("deleted_at", deletedAt); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }

        // 2. 删除会话
        var sessionList []model.Session
        if res := dao.GormDB.Model(&model.Session{}).Where("receive_id = ?", uuid).Find(&sessionList); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }
        for _, session := range sessionList {
            if res := dao.GormDB.Model(&session).Update("deleted_at", deletedAt); res.Error != nil {
                zlog.Error(res.Error.Error())
                return constants.SYSTEM_ERROR, -1
            }
        }

        // 3. 删除联系人
        var userContactList []model.UserContact
        if res := dao.GormDB.Model(&model.UserContact{}).Where("contact_id = ?", uuid).Find(&userContactList); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }

        for _, userContact := range userContactList {
            if res := dao.GormDB.Model(&userContact).Update("deleted_at", deletedAt); res.Error != nil {
                zlog.Error(res.Error.Error())
                return constants.SYSTEM_ERROR, -1
            }
        }

        // 4. 删除申请记录
        var contactApplys []model.ContactApply
        if res := dao.GormDB.Model(&contactApplys).Where("contact_id = ?", uuid).Find(&contactApplys); res.Error != nil {
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
    }

    // ★5. 清理Redis缓存（带错误处理）
    if err := myredis.DelKeysWithPrefix("contact_mygroup_list"); err != nil {
        zlog.Error(err.Error())
    }
    if err := myredis.DelKeysWithPrefix("group_session_list"); err != nil {
        zlog.Error(err.Error())
    }
    for _, uuid := range uuidList {
        if err := myredis.DelKeysWithPattern("group_info_" + uuid); err != nil {
            zlog.Error(err.Error())
        }
    }
    if err := myredis.DelKeysWithPrefix("contact_user_list"); err != nil {
        zlog.Error(err.Error())
    }
    if err := myredis.DelKeysWithPrefix("session_list"); err != nil {
        zlog.Error(err.Error())
    }
    return "解散/删除群聊成功", 0
}

// SetGroupsStatus 设置群聊是否启用
func (g *groupInfoService) SetGroupsStatus(uuidList []string, status int8) (string, int) {
    var deletedAt gorm.DeletedAt
    deletedAt.Time = time.Now()
    deletedAt.Valid = true

    for _, uuid := range uuidList {
        // 1. 更新群组状态
        if res := dao.GormDB.Model(&model.GroupInfo{}).Where("uuid = ?", uuid).Update("status", status); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, -1
        }

        // ★2. 如果是禁用状态，同步删除会话
        if status == group_status_enum.DISABLE {
            var sessionList []model.Session
            if res := dao.GormDB.Model(&sessionList).Where("receive_id = ?", uuid).Find(&sessionList); res.Error != nil {
                zlog.Error(res.Error.Error())
                return constants.SYSTEM_ERROR, -1
            }
            for _, session := range sessionList {
                if res := dao.GormDB.Model(&session).Update("deleted_at", deletedAt); res.Error != nil {
                    zlog.Error(res.Error.Error())
                    return constants.SYSTEM_ERROR, -1
                }
            }
        }
    }

    // ★3. Redis缓存清理（当前代码被注释，暂不启用）
    for _, uuid := range uuidList {
       if err := myredis.DelKeysWithPattern("group_info_" + uuid); err != nil {
           zlog.Error(err.Error())
       }
    }

    return "设置成功", 0
}
