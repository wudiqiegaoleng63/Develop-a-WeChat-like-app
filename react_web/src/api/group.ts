import api from './axios'
import type { ApiResponse } from '../types/api'
import type { GroupInfo, GroupMember } from '../types/group'

export async function createGroup(data: { name: string; owner_id: string; notice?: string; add_mode?: number; avatar?: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/group/createGroup', data)
  return res.data
}

export async function checkGroupAddMode(group_id: string): Promise<ApiResponse<number>> {
  const res = await api.post('/group/checkGroupAddMode', { group_id })
  return res.data
}

export async function enterGroupDirectly(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/group/enterGroupDirectly', data)
  return res.data
}

export async function getGroupInfo(group_id: string): Promise<ApiResponse<GroupInfo>> {
  const res = await api.post('/group/getGroupInfo', { group_id })
  return res.data
}

export async function updateGroupInfo(data: { owner_id: string; uuid: string; name: string; notice: string; add_mode: number; avatar: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/group/updateGroupInfo', data)
  return res.data
}

export async function removeGroupMembers(data: { group_id: string; owner_id: string; uuid_list: string[] }): Promise<ApiResponse<null>> {
  const res = await api.post('/group/removeGroupMembers', data)
  return res.data
}

export async function loadMyGroup(owner_id: string): Promise<ApiResponse<GroupInfo[]>> {
  const res = await api.post('/group/loadMyGroup', { owner_id })
  return res.data
}

export async function getGroupMemberList(group_id: string): Promise<ApiResponse<GroupMember[]>> {
  const res = await api.post('/group/getGroupMemberList', { group_id })
  return res.data
}

export async function leaveGroup(data: { user_id: string; group_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/group/leaveGroup', data)
  return res.data
}

export async function dismissGroup(data: { owner_id: string; group_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/group/dismissGroup', data)
  return res.data
}

// ========== 管理员接口 ==========

export async function getGroupInfoList(): Promise<ApiResponse<GroupInfo[]>> {
  const res = await api.post('/group/getGroupInfoList')
  return res.data
}

export async function deleteGroups(uuid_list: string[]): Promise<ApiResponse<null>> {
  const res = await api.post('/group/deleteGroups', { uuid_list })
  return res.data
}

export async function setGroupsStatus(data: { uuid_list: string[]; status: number }): Promise<ApiResponse<null>> {
  const res = await api.post('/group/setGroupsStatus', data)
  return res.data
}
