package request

type DismissGroupRequest struct {
	OwnerId string `json:"owner_id"`
	GroupId string `json:"group_id"`
}