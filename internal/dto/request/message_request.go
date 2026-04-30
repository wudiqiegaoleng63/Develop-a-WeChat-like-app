package request

// GetMessageListRequest 获取聊天记录请求
type GetMessageListRequest struct {
    UserOneId string `json:"user_one_id"`  // 用户一的uuid（通常是当前登录用户）
    UserTwoId string `json:"user_two_id"`  // 用户二的uuid（聊天对象）
}

type GetGroupMessageListRequest struct {
    GroupId string `json:"group_id"`  // 群聊uuid
}