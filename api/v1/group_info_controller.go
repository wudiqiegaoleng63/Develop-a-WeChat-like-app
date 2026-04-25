package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/gorm"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateGroup 创建群聊
func CreateGroup(c *gin.Context) {
	var createGroupReq request.CreateGroupRequest

	if err := c.BindJSON(&createGroupReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}

	message, ret := gorm.GroupInfoService.CreateGroup(createGroupReq)

	JsonBack(c, message, ret, nil)
}


// LoadMyGroup 获取我创建的群聊
func LoadMyGroup(c *gin.Context) {
    var loadMyGroupReq request.OwnlistRequest
    // 解析请求
    if err := c.BindJSON(&loadMyGroupReq); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    // 调用service层获取群聊列表
    message, groupList, ret := gorm.GroupInfoService.LoadMyGroup(loadMyGroupReq.OwnerId)
    // 返回响应
    JsonBack(c, message, ret, groupList)
}

// CheckGroupAddMode 检查群聊加群方式
func CheckGroupAddMode(c *gin.Context) {
    var req request.CheckGroupAddModeRequest
    // 解析请求
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    // 调用service层查询加群方式
    message, addMode, ret := gorm.GroupInfoService.CheckGroupAddMode(req.GroupId)
    // 返回响应
    JsonBack(c, message, ret, addMode)
}

// GetGroupInfo 获取群聊详情
func GetGroupInfo(c *gin.Context) {
    var req request.GetGroupInfoRequest
    // 解析请求
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    // 调用service层获取群聊详情
    message, groupInfo, ret := gorm.GroupInfoService.GetGroupInfo(req.GroupId)
    // 返回响应
    JsonBack(c, message, ret, groupInfo)
}