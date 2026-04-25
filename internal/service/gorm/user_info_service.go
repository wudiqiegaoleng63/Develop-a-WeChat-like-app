package gorm

import (
	"encoding/json"
	"errors"
	"fmt"
	"kama-chat-server/internal/dao"
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/dto/respond"
	"kama-chat-server/internal/model"
	"kama-chat-server/internal/service/email"
	myredis "kama-chat-server/internal/service/redis"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/enum/user_info/user_status_enum"
	"kama-chat-server/pkg/util/random"
	"kama-chat-server/pkg/zlog"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// UserInfoServiceInterface - 服务接口定义
type UserInfoServiceInterface interface {
	Login(req request.LoginRequest) (string, *respond.LoginRespond, int)
	Register(req request.RegisterRequest) (string, *respond.RegisterRespond, int)
	EmailLogin(req request.EmailLoginRequest) (string, *respond.LoginRespond, int)
	UpdateUserInfo(updateReq request.UpdateUserInfoRequest) (string, int)
	GetUserInfo(uuid string) (string, *respond.GetUserInfoRespond, int)
	GetUserInfoList(ownerId string) (string, []respond.GetUserListRespond, int)
	SetAdmin(uuidList []string, isAdmin int8) (string, int)
}

// userInfoService - 服务结构体 要返回的结构体 实现以上所有方法
type userInfoService struct{}

// 全局服务实例
var UserInfoService UserInfoServiceInterface = &userInfoService{}

// ============================================================
// Login - 登录业务逻辑
// ============================================================
func (u *userInfoService) Login(loginReq request.LoginRequest) (string, *respond.LoginRespond, int) {
	//  1. 根据Email查询用户
	var user model.UserInfo

	res := dao.GormDB.First(&user, "Email=?", loginReq.Email)

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
		Uuid:       user.Uuid,
		Telephone:  user.Telephone,
		Nickname:   user.Nickname,
		Email:      user.Email,
		Avatar:     user.Avatar,
		Gender:     user.Gender,
		Signature:  user.Signature,
		Birthday:   user.Birthday,
		IsAdmin:    user.IsAdmin,
		Status:     user.Status,
	}

	// CreateAt 格式化
	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "登录成功", loginRsp, 0
}

// ============================================================
// Register - 注册业务逻辑（邮箱+密码+昵称+邮箱验证码）
// ============================================================
func (u *userInfoService) Register(registerReq request.RegisterRequest) (string, *respond.RegisterRespond, int) {
	// 1.验证邮箱验证码
	message, ret := email.VerifyCode(registerReq.Email, registerReq.EmailCode)
	if ret != 0 {
		return message, nil, ret
	}

	// 2.验证邮箱是否已经注册
	message, ret = u.checkEmailExist(registerReq.Email)
	if ret != 0 {
		return message, nil, ret
	}

	// 3.创建用户
	var newUser model.UserInfo

	// ★生成Uuid: "U" + 日期(8位) + 机(11位) = 20位字符串
	uuidStr, err := random.GetNowAndLenRandomString(11)
	if err != nil {
		return constants.SYSTEM_ERROR, nil, -1
	}
	newUser.Uuid = "U" + uuidStr

	newUser.Email = registerReq.Email
	newUser.Password = registerReq.Password
	newUser.Nickname = registerReq.Nickname
	newUser.Avatar = "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png"
	newUser.CreatedAt = time.Now()
	newUser.IsAdmin = 0                       // 默认普通用户
	newUser.Status = user_status_enum.NORMAL  // 默认正常状态

	// ★dao.GormDB.Create: 创建记录
	res := dao.GormDB.Create(&newUser)
	if res.Error != nil {
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 4. 构建响应
	registerRsp := &respond.RegisterRespond{
		Uuid:       newUser.Uuid,
		Telephone:  newUser.Telephone,
		Nickname:   newUser.Nickname,
		Email:      newUser.Email,
		Avatar:     newUser.Avatar,
		Gender:     newUser.Gender,
		Birthday:   newUser.Birthday,
		Signature:  newUser.Signature,
		IsAdmin:    newUser.IsAdmin,
		Status:     newUser.Status,
	}
	// ★CreatedAt格式化
	year, month, day := newUser.CreatedAt.Date()
	registerRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "注册成功", registerRsp, 0
}

// ============================================================
// EmailLogin - 邮箱验证码登录（邮箱+验证码，免密码）
// ============================================================
func (u *userInfoService) EmailLogin(req request.EmailLoginRequest) (string, *respond.LoginRespond, int) {
	// 1.查询用户
	var user model.UserInfo

	res := dao.GormDB.First(&user, "email=?", req.Email)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return "用户不存在, 请注册", nil, -2
		}
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 2.验证邮箱验证码
	message, ret := email.VerifyCode(req.Email, req.EmailCode)
	if ret != 0 {
		return message, nil, ret
	}

	// 3.检查用户状态
	if user.Status == 1 {
		return "用户已禁用", nil, -2
	}

	// 4.构建响应
	loginRsp := &respond.LoginRespond{
		Uuid:       user.Uuid,
		Telephone:  user.Telephone,
		Nickname:   user.Nickname,
		Email:      user.Email,
		Avatar:     user.Avatar,
		Gender:     user.Gender,
		Birthday:   user.Birthday,
		Signature:  user.Signature,
		IsAdmin:    user.IsAdmin,
		Status:     user.Status,
	}

	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "登录成功", loginRsp, 0
}

// ============================================================
// checkEmailExist - 检查邮箱是否已存在
// ============================================================
func (u *userInfoService) checkEmailExist(email string) (string, int) {
	var user model.UserInfo

	res := dao.GormDB.First(&user, "email=?", email)

	if res.Error == nil {
		return "邮箱已注册", -2
	}

	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return "", 0
	}

	return constants.SYSTEM_ERROR, -1
}

// ============================================================
// UpdateUserInfo - 更新用户信息
// ============================================================
func (u *userInfoService) UpdateUserInfo(updateReq request.UpdateUserInfoRequest) (string, int) {
	// 1.根据uuid查询用户
	var user model.UserInfo

	if res := dao.GormDB.First(&user, "uuid=?", updateReq.Uuid); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 2.按需更新字段
	if updateReq.Email != "" {
		user.Email = updateReq.Email
	}
	if updateReq.Nickname != "" {
		user.Nickname = updateReq.Nickname
	}
	if updateReq.Birthday != "" {
		user.Birthday = updateReq.Birthday
	}
	if updateReq.Signature != "" {
		user.Signature = updateReq.Signature
	}
	if updateReq.Avatar != "" {
		user.Avatar = updateReq.Avatar
	}

	// 3.存数据库
	if res := dao.GormDB.Save(&user); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// ★4. 删除Redis缓存（当前代码被注释，暂不启用）
	// 更新用户信息后，需要删除user_info缓存，下次查询会从数据库重新加载最新数据
	if err := myredis.DelKeysWithPattern("user_info_" + updateReq.Uuid); err != nil {
		zlog.Error(err.Error())
	}

	return "修改用户信息成功", 0
}

// ============================================================
// GetUserInfo - 获取用户信息
// ============================================================
func (u *userInfoService) GetUserInfo(uuid string) (string, *respond.GetUserInfoRespond, int) {
	zlog.Info(uuid)

	rspString, err := myredis.GetKeyNilIsErr("user_info_" + uuid)

	if err != nil {
		// 缓存未命中
		if errors.Is(err, redis.Nil) {
			zlog.Info(err.Error())

			// 查询用户
			var user model.UserInfo
			if res := dao.GormDB.Where("uuid=?", uuid).Find(&user); res.Error != nil {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}

			rsp := respond.GetUserInfoRespond{
				Uuid:       user.Uuid,
				Telephone:  user.Telephone,
				Nickname:   user.Nickname,
				Avatar:     user.Avatar,
				Birthday:   user.Birthday,
				Email:      user.Email,
				Gender:     user.Gender,
				Signature:  user.Signature,
				CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
				IsAdmin:    user.IsAdmin,
				Status:     user.Status,
			}

			// 写入缓存
			rspBytes, err := json.Marshal(rsp)
			if err != nil {
				zlog.Error(err.Error())
			}

			if err := myredis.SetKeyEx("user_info_"+uuid, string(rspBytes), constants.REDIS_TIMEOUT*time.Minute); err != nil {
				zlog.Error(err.Error())
			}
			return "获取用户信息成功", &rsp, 0
		} else {
			// 其他Redis错误
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	// 缓存命中
	var rsp respond.GetUserInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
	}
	return "获取用户信息成功", &rsp, 0
}


// ============================================================
// GetUserInfoList - 获取用户列表（管理员）
// ============================================================
func (u *userInfoService) GetUserInfoList(ownerId string) (string, []respond.GetUserListRespond, int) {
	var users []model.UserInfo
	// 1.获取所有用户(除了管理员)
	if res := dao.GormDB.Unscoped().Where("uuid != ?", ownerId).Find(&users); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 2.构建响应
	var rsp []respond.GetUserListRespond
	for _, user := range users {
		rp := respond.GetUserListRespond{
			Uuid:      user.Uuid,
			Telephone: user.Telephone,
			Nickname:  user.Nickname,
			Status:    user.Status,
			IsAdmin:   user.IsAdmin,
		}

		if user.DeletedAt.Valid {
			rp.IsDeleted = true
		} else {
			rp.IsDeleted = false
		}
		rsp = append(rsp, rp)
	}

	return "获取用户列表成功", rsp, 0
}

// ============================================================
// SetAdmin - 设置管理员
// ============================================================
func (u *userInfoService) SetAdmin(uuidList []string, isAdmin int8) (string, int) {
	var users []model.UserInfo

	if res := dao.GormDB.Where("uuid = (?)", uuidList).Find(&users); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	for _, user := range users {
		user.IsAdmin = isAdmin
		if res := dao.GormDB.Save(&user); res.Error != nil {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}
	return "设置管理员成功", 0
}

