package request

type ChatMessageRequest struct {
    SessionId  string `json:"session_id"`   // 会话ID
    Type       int8   `json:"type"`          // 消息类型（0=文本,1=语音,2=文件,3=通话）
    Content    string `json:"content"`       // 文本内容
    Url        string `json:"url"`           // 文件URL
    SendId     string `json:"send_id"`       // 发送者UUID
    SendName   string `json:"send_name"`     // 发送者昵称
    SendAvatar string `json:"send_avatar"`   // 发送者头像
    ReceiveId  string `json:"receive_id"`    // 接收者UUID（U开头=用户，G开头=群）
    FileSize   string `json:"file_size"`     // 文件大小
    FileType   string `json:"file_type"`     // 文件类型
    FileName   string `json:"file_name"`     // 文件名
    AVdata     string `json:"av_data"`       // 音视频通话数据
}