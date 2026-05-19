package https_server

import (
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	v1 "kama-chat-server/api/v1"
	"kama-chat-server/internal/config"
	"kama-chat-server/internal/middleware"
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
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	GE.Use(cors.New(corsConfig))

	// 静态文件服务
	conf := config.GetConfig()
	GE.Static("/static/avatars", conf.StaticSrcConfig.StaticAvatarPath)
	GE.Static("/static/files", conf.StaticSrcConfig.StaticFilePath)

	// 注册路由
	registerRoutes()
}

func registerRoutes() {
	// ===== 公开路由（无需认证） =====
	public := GE.Group("")
	{
		public.POST("/login", v1.Login)                          // 邮箱+密码登录
		public.POST("/register", v1.Register)                    // 注册
		public.POST("/user/emailLogin", v1.EmailLogin)           // 邮箱+验证码登录
		public.POST("/user/sendEmailCode", v1.SendEmailCode)     // 发送邮箱验证码
		public.POST("/user/verifyEmailCode", v1.VerifyEmailCode) // 验证邮箱验证码
	}

	// ===== 认证路由（需要JWT） =====
	auth := GE.Group("").Use(middleware.JWTAuth())
	{
		auth.POST("/user/updateUserInfo", v1.UpdateUserInfo)
		auth.POST("/user/getUserInfo", v1.GetUserInfo)
		auth.GET("/user/wsLogin", v1.WsLogin)       // WebSocket登录（GET请求）
		auth.POST("/user/wsLogout", v1.WsLogout)    // WebSocket登出
		auth.POST("/group/createGroup", v1.CreateGroup)
		auth.POST("/group/loadMyGroup", v1.LoadMyGroup)
		auth.POST("/group/checkGroupAddMode", v1.CheckGroupAddMode)
		auth.POST("/group/enterGroupDirectly", v1.EnterGroupDirectly)
		auth.POST("/group/leaveGroup", v1.LeaveGroup)
		auth.POST("/group/dismissGroup", v1.DismissGroup)
		auth.POST("/group/getGroupInfo", v1.GetGroupInfo)
		auth.POST("/group/updateGroupInfo", v1.UpdateGroupInfo)
		auth.POST("/group/getGroupMemberList", v1.GetGroupMemberList)
		auth.POST("/group/removeGroupMembers", v1.RemoveGroupMembers)
		auth.POST("/session/openSession", v1.OpenSession)
		auth.POST("/session/getUserSessionList", v1.GetUserSessionList)
		auth.POST("/session/getGroupSessionList", v1.GetGroupSessionList)
		auth.POST("/session/deleteSession", v1.DeleteSession)
		auth.POST("/session/checkOpenSessionAllowed", v1.CheckOpenSessionAllowed)
		auth.POST("/contact/getUserList", v1.GetUserList)
		auth.POST("/contact/loadMyJoinedGroup", v1.LoadMyJoinedGroup)
		auth.POST("/contact/getContactInfo", v1.GetContactInfo)
		auth.POST("/contact/deleteContact", v1.DeleteContact)
		auth.POST("/contact/blackContact", v1.BlackContact)
		auth.POST("/contact/cancelBlackContact", v1.CancelBlackContact)
		auth.POST("/contact/applyContact", v1.ApplyContact)              // 申请添加联系人
		auth.POST("/contact/getNewContactList", v1.GetNewContactList)    // 获取新联系人申请列表
		auth.POST("/contact/passContactApply", v1.PassContactApply)      // 通过联系人申请
		auth.POST("/contact/refuseContactApply", v1.RefuseContactApply)  // 拒绝联系人申请
		auth.POST("/contact/blackApply", v1.BlackApply)                  // 拉黑申请人
		auth.POST("/contact/getAddGroupList", v1.GetAddGroupList)        // 获取进群申请列表
		auth.POST("/message/getMessageList", v1.GetMessageList)          // 获取私聊消息列表
		auth.POST("/message/getGroupMessageList", v1.GetGroupMessageList) // 获取群聊消息列表
		auth.POST("/message/uploadAvatar", v1.UploadAvatar)              // 上传头像
		auth.POST("/message/uploadFile", v1.UploadFile)                  // 上传文件
		auth.POST("/chatroom/getCurContactListInChatRoom", v1.GetCurContactListInChatRoom)
	}

	// ===== 管理员路由（需要JWT + 管理员权限） =====
	admin := GE.Group("").Use(middleware.JWTAuth(), middleware.AdminOnly())
	{
		admin.POST("/user/getUserInfoList", v1.GetUserInfoList)
		admin.POST("/user/ableUsers", v1.AbleUsers)
		admin.POST("/user/disableUsers", v1.DisableUsers)
		admin.POST("/user/deleteUsers", v1.DeleteUsers)
		admin.POST("/user/setAdmin", v1.SetAdmin)
		admin.POST("/group/getGroupInfoList", v1.GetGroupInfoList)
		admin.POST("/group/deleteGroups", v1.DeleteGroups)
		admin.POST("/group/setGroupsStatus", v1.SetGroupsStatus)
	}
}

// RunServer 启动HTTP服务器
func RunServer() {
	conf := config.GetConfig()

	// 拼接地址
	addr := conf.MainConfig.Host + ":" + strconv.Itoa(conf.MainConfig.Port)

	GE.Run(addr)
}
