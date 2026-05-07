package request

// AVData 音视频通话数据结构体
// 用于解析 ChatMessageRequest.AVdata 字段
type AVData struct {
    MessageId string `json:"messageId"` // 消息ID（"PROXY" 表示代理消息）
    Type      string `json:"type"`      // 通话类型：start_call, receive_call, reject_call
}