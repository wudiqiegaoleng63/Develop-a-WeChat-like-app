import api from './axios'
import type { ApiResponse } from '../types/api'
import type { UserInfo } from '../types/user'

export async function verifyEmailCode(data: { email: string; code: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/user/verifyEmailCode', data)
  return res.data
}

export async function updateUserInfo(data: {
  uuid: string
  nickname?: string
  email?: string
  avatar?: string
  gender?: number
  signature?: string
  birthday?: string
}): Promise<ApiResponse<null>> {
  const res = await api.post('/user/updateUserInfo', data)
  return res.data
}

export async function getUserInfo(uuid: string): Promise<ApiResponse<UserInfo>> {
  const res = await api.post('/user/getUserInfo', { uuid })
  return res.data
}

export async function getUserInfoList(owner_id: string): Promise<ApiResponse<UserInfo[]>> {
  const res = await api.post('/user/getUserInfoList', { owner_id })
  return res.data
}

export async function wsLogout(owner_id: string): Promise<ApiResponse<null>> {
  const res = await api.post('/user/wsLogout', { owner_id })
  return res.data
}

// ========== 管理员接口 ==========

export async function ableUsers(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>> {
  const res = await api.post('/user/ableUsers', data)
  return res.data
}

export async function disableUsers(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>> {
  const res = await api.post('/user/disableUsers', data)
  return res.data
}

export async function deleteUsers(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>> {
  const res = await api.post('/user/deleteUsers', data)
  return res.data
}

export async function setAdmin(data: { uuid_list: string[]; is_admin: number }): Promise<ApiResponse<null>> {
  const res = await api.post('/user/setAdmin', data)
  return res.data
}
