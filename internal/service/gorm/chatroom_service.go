package gorm

import "kama-chat-server/internal/dto/respond"

type chatRommService struct {
}

var ChatRoomService = new(chatRommService)

type chatRoomKey struct {
	ownerId   string
	contactId string
}

var chatRooms = make(map[chatRoomKey][]string)

// GetCurContactListInChatRoom 获取当前聊天室联系人列表
func (c *chatRommService) GetCurContactListInChatRoom(ownerId string, contactId string) (string, []respond.GetCurContactListInChatRoomRespond, int) {
	var rspList []respond.GetCurContactListInChatRoomRespond
	for _, contactId := range chatRooms[chatRoomKey{ownerId, contactId}] {
		rspList = append(rspList, respond.GetCurContactListInChatRoomRespond{
			ContactId: contactId,
		})
	}

	return "获取聊天室联系人列表成功", rspList, 0
}