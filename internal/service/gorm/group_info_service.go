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