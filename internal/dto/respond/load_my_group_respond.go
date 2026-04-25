package respond
// LoadMyGroupRespond 获取我创建的群聊响应结构体
type LoadMyGroupRespond struct {
    GroupId   string `json:"group_id"`   // 群聊uuid
    GroupName string `json:"group_name"` // 群聊名称
    Avatar    string `json:"avatar"`     // 群聊头像
}