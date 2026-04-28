package request

type ApplyContactRequest struct {
    OwnerId   string `json:"owner_id"`   // 申请人的用户uuid
    ContactId string `json:"contact_id"` // 被申请的联系人uuid（用户或群聊）
    Message   string `json:"message"`    // 申请信息/申请理由
}