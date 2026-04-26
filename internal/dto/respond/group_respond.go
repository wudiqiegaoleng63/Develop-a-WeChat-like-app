package respond

// GetGroupMemberListRespond 群聊成员信息响应
type GetGroupMemberListRespond struct {
    UserId   string `json:"user_id"`   // 成员用户ID
    Nickname string `json:"nickname"`  // 成员昵称
    Avatar   string `json:"avatar"`    // 成员头像URL
}