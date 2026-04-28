package v1

import (
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/service/email"
	"kama-chat-server/internal/service/gorm"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"net/http"

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

// ============================================================
// UpdateUserInfo - 更新用户信息接口
// ============================================================
func UpdateUserInfo(c *gin.Context) {
	// 1. 绑定请求参数
	var req request.UpdateUserInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	// 2. 调用Service层
	message, ret := gorm.UserInfoService.UpdateUserInfo(req)
	// 3. 返回响应
	JsonBack(c, message, ret, nil)
}

// ============================================================
// GetUserInfo - 获取用户信息
// ============================================================
func GetUserInfo(c *gin.Context) {
	var req request.GetUserInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":	500,
			"message":	constants.SYSTEM_ERROR,
		})
		return
	}
	// 调用service
	message, userInfo, ret := gorm.UserInfoService.GetUserInfo(req.Uuid)

	JsonBack(c, message, ret, userInfo)
}


// ============================================================
// GetUserInfoList - 获取用户列表（管理员）
// ============================================================
func GetUserInfoList(c *gin.Context) {
	var req request.GetUserInfoListRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}

	message, userList, ret := gorm.UserInfoService.GetUserInfoList(req.OwnerId)
	
	JsonBack(c, message, ret, userList)
}

// ============================================================
// SetAdmin - 设置管理员
// ============================================================

func SetAdmin(c *gin.Context) {
	var req request.AbleUsersRequest

	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"message": constants.SYSTEM_ERROR,
		})
		return 
	}

	message, ret := gorm.UserInfoService.SetAdmin(req.UuidList, req.IsAdmin)

	JsonBack(c, message, ret, nil)
}

// ============================================================
// AbleUsers - 启用用户（管理员）
// ============================================================
func AbleUsers(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.AbleUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}

// ============================================================
// DisableUsers - 禁用用户（管理员）
// ============================================================
func DisableUsers(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.DisableUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}

// ============================================================
// DeleteUsers - 删除用户（管理员，软删除）
// ============================================================
func DeleteUsers(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.DeleteUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}