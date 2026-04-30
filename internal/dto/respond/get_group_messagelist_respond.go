package respond

type GetGroupMessageListRespond struct {
    SendId     string `json:"send_id"`
    SendName   string `json:"send_name"`
    SendAvatar string `json:"send_avatar"`
    ReceiveId  string `json:"receive_id"`
    Type       int8   `json:"type"`
    Content    string `json:"content"`
    Url        string `json:"url"`
    FileType   string `json:"file_type"`
    FileName   string `json:"file_name"`
    FileSize   string `json:"file_size"`
    CreatedAt  string `json:"created_at"`
}