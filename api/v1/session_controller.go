package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/gorm"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"

	"github.com/gin-gonic/gin"

	"net/http"
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
    message, sessionId, ret := gorm.SessionService.OpenSession(openSessionReq)
    JsonBack(c, message, ret, sessionId)
}

// GetUserSessionList 获取用户会话列表
func GetUserSessionList(c *gin.Context) {
    var getUserSessionListReq request.OwnlistRequest
    if err := c.BindJSON(&getUserSessionListReq); err != nil {
        zlog.Error(err.Error())
        c.JSON(http.StatusOK, gin.H{
            "code": 500,
            "message": constants.SYSTEM_ERROR,
        })
        return
    }
    message, sessionList, ret := gorm.SessionService.GetUserSessionList(getUserSessionListReq.OwnerId)
    JsonBack(c, message, ret, sessionList)

}