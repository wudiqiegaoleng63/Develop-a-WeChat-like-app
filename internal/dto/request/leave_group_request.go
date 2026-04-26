package request

type LeaveGroupRequest struct {
	UserId  string `json:"user_id"`
	GroupId string `json:"group_id"`
}