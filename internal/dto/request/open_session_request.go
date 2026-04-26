package request

type OpenSessionRequest struct {
    SendId    string `json:"send_id"`
    ReceiveId string `json:"receive_id"`
}