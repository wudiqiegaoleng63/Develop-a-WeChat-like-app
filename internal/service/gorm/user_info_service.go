package gorm

import (
	"errors"
	"fmt"
	"kama-chat-server/internal/dao"
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/dto/respond"
	"kama-chat-server/internal/model"
	"kama-chat-server/pkg/constants"

	"gorm.io/gorm"
)

// UserInfoServiceInterface - 服务接口定义
type UserInfoServiceInterface interface {
	Login(req request.LoginRequest) (string, *respond.LoginRespond, int)

}

// userInfoService - 服务结构体 要返回的结构体 实现以上所有方法
type userInfoService struct {

}


// 全局服务实例
var UserInfoService UserInfoServiceInterface = &userInfoService{}

// ============================================================
// Login - 登录业务逻辑
// ============================================================
func (u *userInfoService) Login(loginReq request.LoginRequest) (string, *respond.LoginRespond, int) {
	//  1. 根据手机号查询用户	
	var user model.UserInfo

	res := dao.GormDB.First(&user, "telephone = ?", loginReq.Telephone)

	//  2.处理查询结果
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return "用户不存在，请注册", nil, -2
		}
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 	3.验证密码
	if user.Password != loginReq.Password {
		return "密码不正确，请重试", nil, -2
	}

	// 	4.检查用户状态
	if user.Status == 1 {
		return "用户已禁用", nil, -2
	}

	// 	5.构建响应数据
	loginRsp := &respond.LoginRespond{
		Uuid: user.Uuid,
		Telephone: user.Telephone,
        Nickname:  user.Nickname,
        Email:     user.Email,
        Avatar:    user.Avatar,
        Gender:    user.Gender,
        Signature: user.Signature,
        Birthday:  user.Birthday,
        IsAdmin:   user.IsAdmin,
        Status:    user.Status,
	}

	// CreateAt 格式化
	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "登录成功", loginRsp, 0
}

// ============================================================
// Register - 注册业务逻辑
// ============================================================


