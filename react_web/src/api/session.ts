import api from './axios'
import type { ApiResponse } from '../types/api'
import type { UserSession, GroupSession } from '../types/session'

export async function openSession(data: { send_id: string; receive_id: string }): Promise<ApiResponse<string>> {
  const res = await api.post('/session/openSession', data)
  return res.data
}

export async function getUserSessionList(owner_id: string): Promise<ApiResponse<UserSession[]>> {
  const res = await api.post('/session/getUserSessionList', { owner_id })
  return res.data
}

export async function getGroupSessionList(owner_id: string): Promise<ApiResponse<GroupSession[]>> {
  const res = await api.post('/session/getGroupSessionList', { owner_id })
  return res.data
}

export async function deleteSession(data: { owner_id: string; session_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/session/deleteSession', data)
  return res.data
}

export async function checkOpenSessionAllowed(data: { send_id: string; receive_id: string }): Promise<ApiResponse<boolean>> {
  const res = await api.post('/session/checkOpenSessionAllowed', data)
  return res.data
}
