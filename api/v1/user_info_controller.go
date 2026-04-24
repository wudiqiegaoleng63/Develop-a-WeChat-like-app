package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/email"
	"kama-chat-server/internal/service/gorm"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"

	"github.com/gin-gonic/gin"
)

// ============================================================
// Login - 登录接口（邮箱+密码）
// ============================================================
func Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
		return
	}

	message, userInfo, ret := gorm.UserInfoService.Login(req)
	JsonBack(c, message, ret, userInfo)
}
// ============================================================
// Register - 注册接口（邮箱+密码+昵称+邮箱验证码）
// ============================================================
func Register(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
        return
    }

    message, userInfo, ret := gorm.UserInfoService.Register(req)
    JsonBack(c, message, ret, userInfo)
}

// ============================================================
// EmailLogin - 邮箱验证码登录接口
// ============================================================
func EmailLogin(c *gin.Context) {
    var req request.EmailLoginRequest
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
        return
    }

    message, userInfo, ret := gorm.UserInfoService.EmailLogin(req)
    JsonBack(c, message, ret, userInfo)
}


// SendEmailCode 发送邮箱验证码
func SendEmailCode(c *gin.Context) {
	var req request.SendEmailCodeRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
		return
	}

	message, ret := email.SendVerificationCode(req.Email)
	JsonBack(c, message, ret, nil)
}

// VerifyEmailCode 验证邮箱验证码
func VerifyEmailCode(c *gin.Context) {
	 var req request.VerifyEmailCodeRequest
    if err := c.BindJSON(&req); err != nil {
        zlog.Error(err.Error())
        JsonBack(c, constants.SYSTEM_ERROR, -1, nil)
        return
    }

    message, ret := email.VerifyCode(req.Email, req.Code)
    JsonBack(c, message, ret, nil)
}
