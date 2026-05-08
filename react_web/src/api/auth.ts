import api from './axios'
import type { ApiResponse, LoginRequest, RegisterRequest, EmailLoginRequest, SendEmailCodeRequest } from '../types/api'
import type { UserInfo } from '../types/user'

export async function login(data: LoginRequest): Promise<ApiResponse<UserInfo>> {
  const res = await api.post<ApiResponse<UserInfo>>('/login', data)
  return res.data
}

export async function register(data: RegisterRequest): Promise<ApiResponse<UserInfo>> {
  const res = await api.post<ApiResponse<UserInfo>>('/register', data)
  return res.data
}

export async function emailLogin(data: EmailLoginRequest): Promise<ApiResponse<UserInfo>> {
  const res = await api.post<ApiResponse<UserInfo>>('/user/emailLogin', data)
  return res.data
}

export async function sendEmailCode(data: SendEmailCodeRequest): Promise<ApiResponse<null>> {
  const res = await api.post<ApiResponse<null>>('/user/sendEmailCode', data)
  return res.data
}
