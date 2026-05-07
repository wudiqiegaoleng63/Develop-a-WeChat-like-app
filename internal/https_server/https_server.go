package https_server

import (
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	v1 "kama-chat-server/api/v1"
	"kama-chat-server/internal/config"
)

// 全局gin实例
var GE *gin.Engine

func init() {
	// 创建 Gin 引擎
	GE = gin.Default()

	// CORS 跨域配置
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "PUT", "POST", "DELETE"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type"}
	GE.Use(cors.New(corsConfig))

	// 静态文件服务
	conf := config.GetConfig()
	GE.Static("/static/avatars", conf.StaticSrcConfig.StaticAvatarPath)
	GE.Static("/static/files", conf.StaticSrcConfig.StaticFilePath)

	// 注册路由
	registerRoutes()
}

func registerRoutes() {
	// 用户相关路由 (POST)
	GE.POST("/login", v1.Login)                          // 邮箱+密码登录
    GE.POST("/register", v1.Register)                    // 注册
    GE.POST("/user/emailLogin", v1.EmailLogin)           // 邮箱+验证码登录
    GE.POST("/user/sendEmailCode", v1.SendEmailCode)     // 发送邮箱验证码
    GE.POST("/user/verifyEmailCode", v1.VerifyEmailCode) // 验证邮箱验证码
	GE.POST("/user/updateUserInfo", v1.UpdateUserInfo)
	GE.POST("/user/getUserInfoList", v1.GetUserInfoList)
	GE.POST("/user/getUserInfo", v1.GetUserInfo)
	GE.POST("/user/ableUsers", v1.AbleUsers)
	GE.POST("/user/disableUsers", v1.DisableUsers)
	GE.POST("/user/deleteUsers", v1.DeleteUsers)
	GE.POST("/user/setAdmin", v1.SetAdmin)
	GE.POST("/group/createGroup", v1.CreateGroup)
	GE.POST("/group/loadMyGroup", v1.LoadMyGroup)
	GE.POST("/group/checkGroupAddMode", v1.CheckGroupAddMode)
	GE.POST("/group/enterGroupDirectly", v1.EnterGroupDirectly)
	GE.POST("/group/leaveGroup", v1.LeaveGroup)
	GE.POST("/group/dismissGroup", v1.DismissGroup)
	GE.POST("/group/getGroupInfo", v1.GetGroupInfo)
	GE.POST("/group/getGroupInfoList", v1.GetGroupInfoList)
	GE.POST("/group/deleteGroups", v1.DeleteGroups)
	GE.POST("/group/setGroupsStatus", v1.SetGroupsStatus)
	GE.POST("/group/updateGroupInfo", v1.UpdateGroupInfo)
	GE.POST("/group/getGroupMemberList", v1.GetGroupMemberList)
	GE.POST("/group/removeGroupMembers", v1.RemoveGroupMembers)
	GE.POST("/session/openSession", v1.OpenSession)
	GE.POST("/session/getUserSessionList", v1.GetUserSessionList)
	GE.POST("/session/getGroupSessionList", v1.GetGroupSessionList)
	GE.POST("/session/deleteSession", v1.DeleteSession)
	GE.POST("/session/checkOpenSessionAllowed", v1.CheckOpenSessionAllowed)
	GE.POST("/contact/getUserList", v1.GetUserList)
	GE.POST("/contact/loadMyJoinedGroup", v1.LoadMyJoinedGroup)
	GE.POST("/contact/getContactInfo", v1.GetContactInfo)
	GE.POST("/contact/deleteContact", v1.DeleteContact)
	GE.POST("/contact/blackContact", v1.BlackContact)
	GE.POST("/contact/cancelBlackContact", v1.CancelBlackContact)
	GE.POST("/contact/applyContact", v1.ApplyContact)              // 申请添加联系人
    GE.POST("/contact/getNewContactList", v1.GetNewContactList)    // 获取新联系人申请列表
    GE.POST("/contact/passContactApply", v1.PassContactApply)      // 通过联系人申请
    GE.POST("/contact/refuseContactApply", v1.RefuseContactApply)  // 拒绝联系人申请
    GE.POST("/contact/blackApply", v1.BlackApply)                  // 拉黑申请人
    GE.POST("/contact/getAddGroupList", v1.GetAddGroupList)        // 获取进群申请列表
	GE.POST("/message/getMessageList", v1.GetMessageList)          // 获取私聊消息列表
	GE.POST("/message/getGroupMessageList", v1.GetGroupMessageList) // 获取群聊消息列表
	GE.POST("/message/uploadAvatar", v1.UploadAvatar)              // 上传头像
	GE.POST("/message/uploadFile", v1.UploadFile)                  // 上传文件
	GE.GET("/user/wsLogin", v1.WsLogin)      // WebSocket登录（GET请求）
    GE.POST("/user/wsLogout", v1.WsLogout)   // WebSocket登出
}

// RunServer 启动HTTP服务器
func RunServer() {
	conf := config.GetConfig()

	// 拼接地址
	addr := conf.MainConfig.Host + ":" + strconv.Itoa(conf.MainConfig.Port)

	GE.Run(addr)
}