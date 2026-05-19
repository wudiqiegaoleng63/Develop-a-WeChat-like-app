package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/chat"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WsLogin WebSocket登录
func WsLogin(c *gin.Context) {
	// 从JWT中间件注入的上下文中获取uuid
	uuid, exists := c.Get("uuid")
	if !exists {
		zlog.Error("uuid获取失败")
		c.JSON(http.StatusOK, gin.H{
			"code":    401,
			"message": "未认证",
		})
		return
	}
	clientId := uuid.(string)

	chat.NewClientInit(c, clientId)
}

// WsLogout WebSocket登出
func WsLogout(c *gin.Context) {
	var req request.WsLogoutRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 校验权限：只能登出自己
	if !CheckOwner(c, req.OwnerId) {
		return
	}
	message, ret := chat.ClientLogout(req.OwnerId)
	JsonBack(c, message, ret, nil)
}
