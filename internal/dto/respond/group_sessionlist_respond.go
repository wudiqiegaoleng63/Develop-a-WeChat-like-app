package respond

type GroupSessionListRespond struct {
    SessionId string `json:"session_id"`
    Avatar    string `json:"avatar"`
    GroupId   string `json:"group_id"`
    GroupName string `json:"group_name"`
}