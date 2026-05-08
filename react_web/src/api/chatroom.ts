import api from './axios'
import type { ApiResponse } from '../types/api'
import type { ContactInfo } from '../types/user'

export async function getCurContactListInChatRoom(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<ContactInfo[]>> {
  const res = await api.post('/chatroom/getCurContactListInChatRoom', data)
  return res.data
}
