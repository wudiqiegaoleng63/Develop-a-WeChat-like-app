package agent

import (
	"gochat/internal/dao"
	"gochat/internal/model"
	"gochat/pkg/constants"
	"gochat/pkg/enum/user_info/user_status_enum"
	"gochat/pkg/zlog"
	"time"
)

// init 在包导入时幂等创建 AI 助手系统用户。
// 通过 blank import _ "gochat/internal/agent" 触发，
// 不需要手动执行 SQL migration。
func init() {
	bootstrapAgentBot()
}

// bootstrapAgentBot 幂等插入 AI 助手用户行。
// 若已存在则跳过，保证多次启动安全。
func bootstrapAgentBot() {
	if dao.GormDB == nil {
		zlog.Error("Agent bootstrap: GormDB 未初始化，跳过")
		return
	}

	var bot model.UserInfo
	res := dao.GormDB.Where("uuid = ?", constants.AgentBotUuid).First(&bot)
	if res.Error == nil {
		// 已存在，确保昵称/头像/状态为预期值（防止被人改坏）
		dao.GormDB.Model(&bot).Updates(map[string]interface{}{
			"nickname": constants.AgentBotNickname,
			"avatar":   constants.AgentBotAvatar,
			"status":   user_status_enum.NORMAL,
			"is_admin": 0,
		})
		zlog.Info("Agent bootstrap: AI助手用户已存在，已刷新配置")
		return
	}

	// 不存在则创建
	bot = model.UserInfo{
		Uuid:      constants.AgentBotUuid,
		Nickname:  constants.AgentBotNickname,
		Avatar:    constants.AgentBotAvatar,
		IsAdmin:   0,
		Status:    user_status_enum.NORMAL,
		CreatedAt: time.Now(),
	}
	if res := dao.GormDB.Create(&bot); res.Error != nil {
		zlog.Error("Agent bootstrap: 创建 AI助手用户失败: " + res.Error.Error())
		return
	}
	zlog.Info("Agent bootstrap: AI助手用户创建成功 uuid=" + constants.AgentBotUuid)
}
