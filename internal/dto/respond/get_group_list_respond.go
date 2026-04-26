package respond

// GetGroupListRespond 获取群组列表响应结构体
type GetGroupListRespond struct {
    Uuid      string `json:"uuid"`       // 群组UUID
    Name      string `json:"name"`       // 群组名称
    OwnerId   string `json:"owner_id"`   // 群主UUID
    Status    int8   `json:"status"`     // 状态：0=正常，1=禁用，2=解散
    IsDeleted bool   `json:"is_deleted"` // 是否已软删除
}