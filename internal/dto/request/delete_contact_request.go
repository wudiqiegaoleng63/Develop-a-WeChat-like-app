package request

type DeleteContactRequest struct {
    OwnerId   string `json:"owner_id"`
    ContactId string `json:"contact_id"`
}
