package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/https_server"
	"kama-chat-server/internal/service/email"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"net"

	"github.com/gin-gonic/gin"
)

// SendEmailCode 发送邮箱验证码
func SendEmailCode(c *gin.Context) {
	var req request.SendEmailCodeRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		https_server.JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
		return
	}

	message, ret := email.SendVerificationCode(req.Email)
	https_server.JsonBack(c, message, ret, nil)
}

// VerifyEmailCode 验证邮箱验证码
func VerifyEmailCode(c *gin.Context) {
	 var req request.VerifyEmailCodeRequest
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        https_server.JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
        return
    }

    message, ret := email.VerifyCode(req.Email, req.Code)
    https_server.JsonBack(c, message, ret, nil)
}


// Login 登录接口
func Login(c *gin.Context){
	
}