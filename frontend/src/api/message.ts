import api from './axios'
import type { ApiResponse } from '../types/api'
import type { ChatMessage } from '../types/message'

export async function getMessageList(data: { user_one_id: string; user_two_id: string }): Promise<ApiResponse<ChatMessage[]>> {
  const res = await api.post('/message/getMessageList', data)
  return res.data
}

export async function getGroupMessageList(group_id: string): Promise<ApiResponse<ChatMessage[]>> {
  const res = await api.post('/message/getGroupMessageList', { group_id })
  return res.data
}

export async function uploadFile(file: File): Promise<ApiResponse<null>> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await api.post('/message/uploadFile', formData)
  return res.data
}

export async function uploadAvatar(file: File): Promise<ApiResponse<null>> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await api.post('/message/uploadAvatar', formData)
  return res.data
}
