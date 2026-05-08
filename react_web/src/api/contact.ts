import api from './axios'
import type { ApiResponse } from '../types/api'
import type { ContactInfo, ContactUser } from '../types/user'

export async function getUserList(owner_id: string): Promise<ApiResponse<ContactUser[]>> {
  const res = await api.post('/contact/getUserList', { owner_id })
  return res.data
}

export async function getContactInfo(contact_id: string): Promise<ApiResponse<ContactInfo>> {
  const res = await api.post('/contact/getContactInfo', { contact_id })
  return res.data
}

export async function applyContact(data: { owner_id: string; contact_id: string; message?: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/contact/applyContact', data)
  return res.data
}

export async function getNewContactList(owner_id: string): Promise<ApiResponse<ContactInfo[]>> {
  const res = await api.post('/contact/getNewContactList', { owner_id })
  return res.data
}

export async function passContactApply(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/contact/passContactApply', data)
  return res.data
}

export async function refuseContactApply(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/contact/refuseContactApply', data)
  return res.data
}

export async function deleteContact(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/contact/deleteContact', data)
  return res.data
}

export async function blackContact(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/contact/blackContact', data)
  return res.data
}

export async function cancelBlackContact(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/contact/cancelBlackContact', data)
  return res.data
}

export async function blackApply(data: { owner_id: string; contact_id: string }): Promise<ApiResponse<null>> {
  const res = await api.post('/contact/blackApply', data)
  return res.data
}

export async function getAddGroupList(group_id: string): Promise<ApiResponse<ContactInfo[]>> {
  const res = await api.post('/contact/getAddGroupList', { group_id })
  return res.data
}

export async function loadMyJoinedGroup(owner_id: string): Promise<ApiResponse<ContactInfo[]>> {
  const res = await api.post('/contact/loadMyJoinedGroup', { owner_id })
  return res.data
}
