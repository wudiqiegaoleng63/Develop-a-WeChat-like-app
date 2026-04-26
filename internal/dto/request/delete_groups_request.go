package request

// DeleteGroupsRequest 批量删除群组请求
type DeleteGroupsRequest struct {
    UuidList []string `json:"uuid_list"` // 待删除群组UUID列表
}