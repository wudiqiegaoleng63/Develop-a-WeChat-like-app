package respond

type MyUserListRespond struct {
    UserId   string `json:"user_id"`
    UserName string `json:"user_name"`
    Avatar   string `json:"avatar"`
}