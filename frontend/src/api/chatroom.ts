import api from './axios'
import type { ApiResponse } from '../types/api'

interface ChatRoomContact {
  contact_id: string
}

export async function getCurContactListInChatRoom(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<ChatRoomContact[]>> {
  const res = await api.post('/chatroom/getCurContactListInChatRoom', data)
  return res.data
}
