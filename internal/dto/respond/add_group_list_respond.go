package respond

type AddGroupListRespond struct {
    ContactId     string `json:"contact_id"`     // 申请人的用户uuid
    ContactName   string `json:"contact_name"`   // 申请人的昵称
    ContactAvatar string `json:"contact_avatar"` // 申请人的头像
    Message       string `json:"message"`        // 申请信息（已格式化）
}