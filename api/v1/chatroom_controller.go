package v1

import (
	"gochat/internal/dto/request"
	"gochat/internal/service/gorm"
	"gochat/pkg/constants"
	"gochat/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetCurContactListInChatRoom 获取当前聊天室联系人列表
func GetCurContactListInChatRoom(c *gin.Context) {
	var req request.GetCurContactListInChatRoomRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 校验权限
	if !CheckOwner(c, req.OwnerId) {
		return
	}
	message, rspList, ret := gorm.ChatRoomService.GetCurContactListInChatRoom(req.OwnerId, req.ContactId)
	JsonBack(c, message, ret, rspList)
}
