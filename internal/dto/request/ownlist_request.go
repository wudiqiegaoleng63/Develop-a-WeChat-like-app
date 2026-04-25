package request
// OwnlistRequest 通用的"获取我的列表"请求结构体
// 用于多种场景： 获取我的群聊、获取我的联系人等
type OwnlistRequest struct {
    OwnerId string `json:"owner_id"` // 用户uuid
}