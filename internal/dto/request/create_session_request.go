package request

type CreateSessionRequest struct {
    SendId    string `json:"send_id"`
    ReceiveId string `json:"receive_id"`
}