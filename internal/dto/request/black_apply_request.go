package request

type BlackApplyRequest struct {
    OwnerId   string `json:"owner_id"`   // 被申请方uuid
    ContactId string `json:"contact_id"` // 申请人的用户uuid
}