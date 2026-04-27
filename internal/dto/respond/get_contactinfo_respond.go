package respond

import "encoding/json"

type GetContactInfoRespond struct {
    ContactId        string          `json:"contact_id"`
    ContactName      string          `json:"contact_name"`
    ContactAvatar    string          `json:"contact_avatar"`
    ContactPhone     string          `json:"contact_phone"`
    ContactEmail     string          `json:"contact_email"`
    ContactGender    int8            `json:"contact_gender"`
    ContactSignature string          `json:"contact_signature"`
    ContactBirthday  string          `json:"contact_birthday"`
    ContactNotice    string          `json:"contact_notice"`
    ContactMembers   json.RawMessage `json:"contact_members"`
    ContactMemberCnt int             `json:"contact_member_cnt"`
    ContactOwnerId   string          `json:"contact_owner_id"`
    ContactAddMode   int8            `json:"contact_add_mode"`
}