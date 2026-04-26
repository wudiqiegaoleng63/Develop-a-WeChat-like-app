package request

// EnterGroupDirectlyRequest 直接进群请求
type EnterGroupDirectlyRequest struct {
    OwnerId   string `json:"owner_id"`   // 群聊ID
    ContactId string `json:"contact_id"` // 新成员用户ID
}