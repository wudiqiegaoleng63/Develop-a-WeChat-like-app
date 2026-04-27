package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/gorm"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"log"

	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserList 获取联系人列表
func GetUserList(c *gin.Context) {
    var myUserListReq request.OwnlistRequest
    if err := c.BindJSON(&myUserListReq); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    message, userList, ret := gorm.UserContactService.GetUserList(myUserListReq.OwnerId)
    JsonBack(c, message, ret, userList)
}

// LoadMyJoinedGroup 获取我加入的群聊
func LoadMyJoinedGroup(c *gin.Context) {
    var loadMyJoinedGroupReq request.OwnlistRequest
    if err := c.BindJSON(&loadMyJoinedGroupReq); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    message, groupList, ret := gorm.UserContactService.LoadMyJoinedGroup(loadMyJoinedGroupReq.OwnerId)
    JsonBack(c, message, ret, groupList)
}

// GetContactInfo 获取联系人信息
func GetContactInfo(c *gin.Context) {
    var getContactInfoReq request.GetContactInfoRequest
    if err := c.BindJSON(&getContactInfoReq); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    log.Println(getContactInfoReq)
    message, contactInfo, ret := gorm.UserContactService.GetContactInfo(getContactInfoReq.ContactId)
    JsonBack(c, message, ret, contactInfo)
}

// DeleteContact 删除联系人
func DeleteContact(c *gin.Context) {
    var deleteContactReq request.DeleteContactRequest
    if err := c.BindJSON(&deleteContactReq); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    message, ret := gorm.UserContactService.DeleteContact(deleteContactReq.OwnerId, deleteContactReq.ContactId)
    JsonBack(c, message, ret, nil)
}

// BlackContact 拉黑联系人
func BlackContact(c *gin.Context) {
    var req request.BlackContactRequest
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    message, ret := gorm.UserContactService.BlackContact(req.OwnerId, req.ContactId)
    JsonBack(c, message, ret, nil)
}

// CancelBlackContact 解除拉黑联系人
func CancelBlackContact(c *gin.Context) {
    var req request.BlackContactRequest
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    message, ret := gorm.UserContactService.CancelBlackContact(req.OwnerId, req.ContactId)
    JsonBack(c, message, ret, nil)
}