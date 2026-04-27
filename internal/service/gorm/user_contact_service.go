package gorm

import (
	"encoding/json"
	"errors"
	"kama-chat-server/internal/dao"
	"kama-chat-server/internal/dto/respond"
	"kama-chat-server/internal/model"
	myredis "kama-chat-server/internal/service/redis"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/enum/contact/contact_status_enum"
	"kama-chat-server/pkg/enum/contact/contact_type_enum"
	"kama-chat-server/pkg/enum/group_info/group_status_enum"
	"kama-chat-server/pkg/enum/user_info/user_status_enum"
	"kama-chat-server/pkg/zlog"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type userContactService struct {

}

var UserContactService = new(userContactService)

// GetUserList 获取用户列表
func (u *userContactService) GetUserList(ownerId string) (string, []respond.MyUserListRespond, int) {
    rspString, err := myredis.GetKeyNilIsErr("contact_user_list_" + ownerId)
    if err != nil {
        if errors.Is(err, redis.Nil) {

            // dao
            var contactList []model.UserContact
            // 没有被删除
            if res := dao.GormDB.Order("created_at DESC").Where("user_id = ? AND status != 4", ownerId).Find(&contactList); res.Error != nil {
                // 不存在不是业务问题，用Info，return 0
                if errors.Is(res.Error, gorm.ErrRecordNotFound) {
                    message := "目前不存在联系人"
                    zlog.Info(message)
                    return message, nil, 0
                } else {
                    zlog.Error(res.Error.Error())
                    return constants.SYSTEM_ERROR, nil, -1
                }
            }
            // dto
            var userListRsp []respond.MyUserListRespond
            for _, contact := range contactList {
                // 联系人中是用户的
                if contact.ContactType == contact_type_enum.USER {
                    // 获取用户信息
                    var user model.UserInfo
                    if res := dao.GormDB.First(&user, "uuid = ?", contact.ContactId); res.Error != nil {
                        // 肯定是存在的，不可能无缘无故删掉，目前不用加notfound的判断
                        zlog.Error(res.Error.Error())
                        return constants.SYSTEM_ERROR, nil, -1
                    }
                    userListRsp = append(userListRsp, respond.MyUserListRespond{
                        UserId:   user.Uuid,
                        UserName: user.Nickname,
                        Avatar:   user.Avatar,
                    })
                }
            }
            rspString, err := json.Marshal(userListRsp)
            if err != nil {
                zlog.Error(err.Error())
            }
            if err := myredis.SetKeyEx("contact_user_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
                zlog.Error(err.Error())
            }
            return "获取用户列表成功", userListRsp, 0
        } else {
            zlog.Error(err.Error())
        }
    }
    var rsp []respond.MyUserListRespond
    if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
        zlog.Error(err.Error())
    }
    return "获取用户列表成功", rsp, 0
}

// LoadMyJoinedGroup 获取我加入的群聊
func (u *userContactService) LoadMyJoinedGroup(ownerId string) (string, []respond.LoadMyJoinedGroupRespond, int) {
    rspString, err := myredis.GetKeyNilIsErr("my_joined_group_list_" + ownerId)
    if err != nil {
        if errors.Is(err, redis.Nil) {
            var contactList []model.UserContact
            // 没有退群，也没有被踢出群聊
            if res := dao.GormDB.Order("created_at DESC").Where("user_id = ? AND status != 6 AND status != 7", ownerId).Find(&contactList); res.Error != nil {
                // 不存在不是业务问题，用Info，return 0
                if errors.Is(res.Error, gorm.ErrRecordNotFound) {
                    message := "目前不存在加入的群聊"
                    zlog.Info(message)
                    return message, nil, 0
                } else {
                    zlog.Error(res.Error.Error())
                    return constants.SYSTEM_ERROR, nil, -1
                }
            }
            var groupList []model.GroupInfo
            for _, contact := range contactList {
                if contact.ContactId[0] == 'G' {
                    // 获取群聊信息
                    var group model.GroupInfo
                    if res := dao.GormDB.First(&group, "uuid = ?", contact.ContactId); res.Error != nil {
                        zlog.Error(res.Error.Error())
                        return constants.SYSTEM_ERROR, nil, -1
                    }
                    // 群没被删除，同时群主不是自己
                    // 群主删除或admin删除群聊，status为7，即被踢出群聊，所以不用判断群是否被删除，删除了到不了这步
                    if group.OwnerId != ownerId {
                        groupList = append(groupList, group)
                    }
                }
            }
            var groupListRsp []respond.LoadMyJoinedGroupRespond
            for _, group := range groupList {
                groupListRsp = append(groupListRsp, respond.LoadMyJoinedGroupRespond{
                    GroupId:   group.Uuid,
                    GroupName: group.Name,
                    Avatar:    group.Avatar,
                })
            }
            rspString, err := json.Marshal(groupListRsp)
            if err != nil {
                zlog.Error(err.Error())
            }
            if err := myredis.SetKeyEx("my_joined_group_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
                zlog.Error(err.Error())
            }
            return "获取加入群成功", groupListRsp, 0
        } else {
            zlog.Error(err.Error())
            return constants.SYSTEM_ERROR, nil, -1
        }
    }
    var rsp []respond.LoadMyJoinedGroupRespond
    if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
        zlog.Error(err.Error())
    }
    return "获取加入群成功", rsp, 0
}

// GetContactInfo 获取联系人信息
// 调用这个接口的前提是该联系人没有处在删除或被删除，或者该用户还在群聊中
// redis todo
func (u *userContactService) GetContactInfo(contactId string) (string, respond.GetContactInfoRespond, int) {
    if contactId[0] == 'G' {
        var group model.GroupInfo
        if res := dao.GormDB.First(&group, "uuid = ?", contactId); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
        }
        // 没被禁用
        if group.Status != group_status_enum.DISABLE {
            return "获取联系人信息成功", respond.GetContactInfoRespond{
                ContactId:        group.Uuid,
                ContactName:      group.Name,
                ContactAvatar:    group.Avatar,
                ContactNotice:    group.Notice,
                ContactAddMode:   group.AddMode,
                ContactMembers:   group.Members,
                ContactMemberCnt: group.MemberCnt,
                ContactOwnerId:   group.OwnerId,
            }, 0
        } else {
            zlog.Error("该群聊处于禁用状态")
            return "该群聊处于禁用状态", respond.GetContactInfoRespond{}, -2
        }
    } else {
        var user model.UserInfo
        if res := dao.GormDB.First(&user, "uuid = ?", contactId); res.Error != nil {
            zlog.Error(res.Error.Error())
            return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
        }
        log.Println(user)
        if user.Status != user_status_enum.DISABLE {
            return "获取联系人信息成功", respond.GetContactInfoRespond{
                ContactId:        user.Uuid,
                ContactName:      user.Nickname,
                ContactAvatar:    user.Avatar,
                ContactBirthday:  user.Birthday,
                ContactEmail:     user.Email,
                ContactPhone:     user.Telephone,
                ContactGender:    user.Gender,
                ContactSignature: user.Signature,
            }, 0
        } else {
            zlog.Info("该用户处于禁用状态")
            return "该用户处于禁用状态", respond.GetContactInfoRespond{}, -2
        }
    }
}

// DeleteContact 删除联系人（只包含用户）
func (u *userContactService) DeleteContact(ownerId, contactId string) (string, int) {
    // status改变为删除
    var deletedAt gorm.DeletedAt
    deletedAt.Time = time.Now()
    deletedAt.Valid = true
    if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", ownerId, contactId).Updates(map[string]interface{}{
        "deleted_at": deletedAt,
        "status":     contact_status_enum.DELETE,
    }); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", contactId, ownerId).Updates(map[string]interface{}{
        "deleted_at": deletedAt,
        "status":     contact_status_enum.BE_DELETE,
    }); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    if res := dao.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", ownerId, contactId).Update("deleted_at", deletedAt); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }

    if res := dao.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", contactId, ownerId).Update("deleted_at", deletedAt); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    // 联系人添加的记录得删，这样之后再添加就看新的申请记录，如果申请记录结果是拉黑就没法再添加，如果是拒绝可以再添加
    if res := dao.GormDB.Model(&model.ContactApply{}).Where("contact_id = ? AND user_id = ?", ownerId, contactId).Update("deleted_at", deletedAt); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    if res := dao.GormDB.Model(&model.ContactApply{}).Where("contact_id = ? AND user_id = ?", contactId, ownerId).Update("deleted_at", deletedAt); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    if err := myredis.DelKeysWithPattern("contact_user_list_" + ownerId); err != nil {
        zlog.Error(err.Error())
    }
    return "删除联系人成功", 0
}
// BlackContact 拉黑联系人
func (u *userContactService) BlackContact(ownerId string, contactId string) (string, int) {
    // 拉黑
    if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", ownerId, contactId).Updates(map[string]interface{}{
        "status":    contact_status_enum.BLACK,
        "update_at": time.Now(),
    }); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    // 被拉黑
    if res := dao.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", contactId, ownerId).Updates(map[string]interface{}{
        "status":    contact_status_enum.BE_BLACK,
        "update_at": time.Now(),
    }); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    // 删除会话
    var deletedAt gorm.DeletedAt
    deletedAt.Time = time.Now()
    deletedAt.Valid = true
    if res := dao.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", ownerId, contactId).Update("deleted_at", deletedAt); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    return "已拉黑该联系人", 0
}

// CancelBlackContact 取消拉黑联系人
func (u *userContactService) CancelBlackContact(ownerId string, contactId string) (string, int) {
    // 因为前端的设定，这里需要判断一下ownerId和contactId是不是有拉黑和被拉黑的状态
    var blackContact model.UserContact
    if res := dao.GormDB.Where("user_id = ? AND contact_id = ?", ownerId, contactId).First(&blackContact); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    if blackContact.Status != contact_status_enum.BLACK {
        return "未拉黑该联系人，无需解除拉黑", -2
    }
    var beBlackContact model.UserContact
    if res := dao.GormDB.Where("user_id = ? AND contact_id = ?", contactId, ownerId).First(&beBlackContact); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    if beBlackContact.Status != contact_status_enum.BE_BLACK {
        return "该联系人未被拉黑，无需解除拉黑", -2
    }

    // 取消拉黑
    blackContact.Status = contact_status_enum.NORMAL
    beBlackContact.Status = contact_status_enum.NORMAL
    if res := dao.GormDB.Save(&blackContact); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    if res := dao.GormDB.Save(&beBlackContact); res.Error != nil {
        zlog.Error(res.Error.Error())
        return constants.SYSTEM_ERROR, -1
    }
    return "已解除拉黑该联系人", 0
}
