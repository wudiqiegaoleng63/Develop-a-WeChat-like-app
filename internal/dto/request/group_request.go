package request

// UpdateGroupInfoRequest 更新群聊信息请求
type UpdateGroupInfoRequest struct {
    OwnerId  string `json:"owner_id"`  // 群主UUID（用于缓存清理）
    Uuid     string `json:"uuid"`      // 群组唯一标识（必填）
    Name     string `json:"name"`      // 群名称（可选，空则不更新）
    Notice   string `json:"notice"`    // 群公告（可选，空则不更新）
    AddMode  int8   `json:"add_mode"`  // 加群方式：0=直接加入，1=需审核（可选，-1则不更新）
    Avatar   string `json:"avatar"`    // 群头像URL（可选，空则不更新）
}

// GetGroupMemberListRequest 获取群聊成员列表请求
type GetGroupMemberListRequest struct {
    GroupId string `json:"group_id"`  // 群组唯一标识（必填）
}

// RemoveGroupMembersRequest 移除群聊成员请求
type RemoveGroupMembersRequest struct {
    GroupId  string   `json:"group_id"`   // 群组唯一标识（必填）
    OwnerId  string   `json:"owner_id"`   // 群主UUID（必填，用于校验）
    UuidList []string `json:"uuid_list"`  // 待移除成员UUID列表（必填）
}