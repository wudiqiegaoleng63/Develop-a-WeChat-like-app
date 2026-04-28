package request

type PassContactApplyRequest struct {
    OwnerId   string `json:"owner_id"`   // 被申请方uuid（用户或群聊）
    ContactId string `json:"contact_id"` // 申请人的用户uuid
}