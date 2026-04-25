package request

type CreateGroupRequest struct {
    Name      string `json:"name"`       // 群名称
    Notice    string `json:"notice"`     // 群公告（可选）
    OwnerId   string `json:"owner_id"`   // 群主uuid
    AddMode   int8   `json:"add_mode"`   // 加群方式： 0=直接, 1=审核
    Avatar    string `json:"avatar"`     // 群头像（可选）
}