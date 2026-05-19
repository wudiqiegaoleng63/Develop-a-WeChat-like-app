package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/gorm"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// OpenSession 打开会话
func OpenSession(c *gin.Context) {
	var openSessionReq request.OpenSessionRequest
	if err := c.BindJSON(&openSessionReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 校验权限：只能以自己身份打开会话
	if !CheckOwner(c, openSessionReq.SendId) {
		return
	}
	message, sessionId, ret := gorm.SessionService.OpenSession(openSessionReq)
	JsonBack(c, message, ret, sessionId)
}

// GetUserSessionList 获取用户会话列表
func GetUserSessionList(c *gin.Context) {
	var getUserSessionListReq request.OwnlistRequest
	if err := c.BindJSON(&getUserSessionListReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 校验权限
	if !CheckOwner(c, getUserSessionListReq.OwnerId) {
		return
	}
	message, sessionList, ret := gorm.SessionService.GetUserSessionList(getUserSessionListReq.OwnerId)
	JsonBack(c, message, ret, sessionList)
}

// GetGroupSessionList 获取群聊会话列表
func GetGroupSessionList(c *gin.Context) {
	var getGroupListReq request.OwnlistRequest
	if err := c.BindJSON(&getGroupListReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 校验权限
	if !CheckOwner(c, getGroupListReq.OwnerId) {
		return
	}
	message, groupList, ret := gorm.SessionService.GetGroupSessionList(getGroupListReq.OwnerId)
	JsonBack(c, message, ret, groupList)
}

// DeleteSession 删除会话
func DeleteSession(c *gin.Context) {
	var deleteSessionReq request.DeleteSessionRequest
	if err := c.BindJSON(&deleteSessionReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 校验权限：只能删除自己的会话
	if !CheckOwner(c, deleteSessionReq.OwnerId) {
		return
	}
	message, ret := gorm.SessionService.DeleteSession(deleteSessionReq.OwnerId, deleteSessionReq.SessionId)
	JsonBack(c, message, ret, nil)
}

// CheckOpenSessionAllowed 检查是否可以打开会话
func CheckOpenSessionAllowed(c *gin.Context) {
	var req request.CreateSessionRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 校验权限
	if !CheckOwner(c, req.SendId) {
		return
	}
	message, res, ret := gorm.SessionService.CheckOpenSessionAllowed(req.SendId, req.ReceiveId)
	JsonBack(c, message, ret, res)
}
