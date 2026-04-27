package request

type DeleteSessionRequest struct {
    OwnerId   string `json:"owner_id"`
    SessionId string `json:"session_id"`
}