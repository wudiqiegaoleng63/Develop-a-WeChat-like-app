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
	clientId := c.Query("client_id")
	if clientId == "" {
		zlog.Error("clientId获取失败")
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"message": "clientId获取失败",
		})
		return
	}

	chat.NewClientInit(c, clientId)
}

// WsLogout WebSocket登出
// 路由: POST /user/wsLogout
func WsLogout(c *gin.Context) {
    // 1. 绑定请求参数
    var req request.WsLogoutRequest
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code":    500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }

    // 2. 调用Chat服务层关闭WebSocket连接
    message, ret := chat.ClientLogout(req.OwnerId)

    // 3. 返回响应
    JsonBack(c, message, ret, nil)
}