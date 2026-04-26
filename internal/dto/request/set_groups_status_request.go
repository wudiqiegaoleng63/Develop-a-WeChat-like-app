package request

// SetGroupsStatusRequest 设置群组状态请求
type SetGroupsStatusRequest struct {
    UuidList []string `json:"uuid_list"` // 群组UUID列表
    Status   int8     `json:"status"`    // 目标状态：0=正常，1=禁用
}