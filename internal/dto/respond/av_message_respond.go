package respond

// AVMessageRespond 音视频通话消息响应结构体
type AVMessageRespond struct {
    SendId     string `json:"send_id"`     // 发送者UUID
    SendName   string `json:"send_name"`   // 发送者昵称
    SendAvatar string `json:"send_avatar"` // 发送者头像
    ReceiveId  string `json:"receive_id"`  // 接收者UUID
    Type       int8   `json:"type"`        // 消息类型（3=音视频通话）
    Content    string `json:"content"`     // 文本内容
    Url        string `json:"url"`         // 文件URL
    FileType   string `json:"file_type"`   // 文件类型
    FileName   string `json:"file_name"`   // 文件名
    FileSize   string `json:"file_size"`   // 文件大小
    CreatedAt  string `json:"created_at"`  // 创建时间
    AVdata     string `json:"av_data"`     // 音视频通话数据（JSON字符串）
}