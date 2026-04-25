package request

// ============================================================
// WsLogoutRequest - WebSocket登出请求
// ============================================================
type WsLogoutRequest struct {
    OwnerId string `json:"owner_id"` // 登出用户的uuid
}