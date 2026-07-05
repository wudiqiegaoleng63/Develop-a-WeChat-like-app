import { create } from 'zustand'
import type { UserInfo } from '../types/user'
import { login as loginApi, register as registerApi, emailLogin as emailLoginApi } from '../api/auth'
import { wsLogout, updateUserInfo } from '../api/user'
import { uploadAvatar } from '../api/message'
import type { LoginRequest, RegisterRequest, EmailLoginRequest } from '../types/api'
import { normalizeAvatarUrl } from '../utils/avatar'
import { wsService } from '../services/websocket'
import { WS_URL } from '../utils/constants'
import { showToast } from '../utils/toast'
import { useChatStore } from './useChatStore'

interface AuthState {
  userInfo: UserInfo | null
  login: (data: LoginRequest) => Promise<boolean>
  emailLogin: (data: EmailLoginRequest) => Promise<boolean>
  register: (data: RegisterRequest) => Promise<boolean>
  logout: () => void
  updateProfile: (data: Partial<Pick<UserInfo, 'nickname' | 'gender' | 'signature' | 'birthday' | 'avatar'>>) => Promise<boolean>
  uploadAndSetAvatar: (file: File) => Promise<string | null>
}

function normalizeUserInfo(info: UserInfo): UserInfo {
  return {
    ...info,
    avatar: normalizeAvatarUrl(info.avatar || ''),
  }
}

// 从localStorage恢复登录状态
function getInitialUserInfo(): UserInfo | null {
  const stored = localStorage.getItem('userInfo')
  const token = localStorage.getItem('token')
  if (stored && token) {
    try {
      const user = JSON.parse(stored) as UserInfo
      // 使用token重连WebSocket
      wsService.connect(token, WS_URL)
      return user
    } catch {
      localStorage.removeItem('userInfo')
      localStorage.removeItem('token')
    }
  }
  return null
}

export const useAuthStore = create<AuthState>((set, get) => ({
  userInfo: getInitialUserInfo(),

  login: async (data) => {
    const res = await loginApi(data)
    if (res.code === 200 && res.data) {
      if (res.data.status === 1) {
        showToast('账号已被禁用', 'error')
        return false
      }
      const user = normalizeUserInfo(res.data)
      const token = res.data.token || ''
      localStorage.setItem('userInfo', JSON.stringify(user))
      localStorage.setItem('token', token)
      wsService.connect(token, WS_URL)
      set({ userInfo: user })
      return true
    }
    showToast(res.message || '登录失败', 'error')
    return false
  },

  emailLogin: async (data) => {
    const res = await emailLoginApi(data)
    if (res.code === 200 && res.data) {
      if (res.data.status === 1) {
        showToast('账号已被禁用', 'error')
        return false
      }
      const user = normalizeUserInfo(res.data)
      const token = res.data.token || ''
      localStorage.setItem('userInfo', JSON.stringify(user))
      localStorage.setItem('token', token)
      wsService.connect(token, WS_URL)
      set({ userInfo: user })
      return true
    }
    showToast(res.message || '登录失败', 'error')
    return false
  },

  register: async (data) => {
    const res = await registerApi(data)
    if (res.code === 200 && res.data) {
      if (res.data.status === 1) {
        showToast('账号已被禁用', 'error')
        return false
      }
      const user = normalizeUserInfo(res.data)
      const token = res.data.token || ''
      localStorage.setItem('userInfo', JSON.stringify(user))
      localStorage.setItem('token', token)
      wsService.connect(token, WS_URL)
      set({ userInfo: user })
      return true
    }
    showToast(res.message || '注册失败', 'error')
    return false
  },

  logout: () => {
    const { userInfo } = get()
    if (userInfo) {
      wsLogout(userInfo.uuid).catch(() => {})
    }
    wsService.disconnect()
    localStorage.removeItem('userInfo')
    localStorage.removeItem('token')
    useChatStore.getState().resetAll()
    set({ userInfo: null })
  },

  updateProfile: async (data) => {
    const { userInfo } = get()
    if (!userInfo) return false
    const res = await updateUserInfo({ uuid: userInfo.uuid, ...data })
    if (res.code === 200) {
      const updated = { ...userInfo, ...data }
      localStorage.setItem('userInfo', JSON.stringify(updated))
      set({ userInfo: updated })
      return true
    }
    showToast(res.message || '更新失败', 'error')
    return false
  },

  uploadAndSetAvatar: async (file) => {
    const { userInfo } = get()
    if (!userInfo) return null
    const res = await uploadAvatar(file)
    if (res.code === 200) {
      const avatarPath = '/static/avatars/' + file.name
      const ok = await get().updateProfile({ avatar: avatarPath })
      return ok ? avatarPath : null
    }
    showToast('头像上传失败', 'error')
    return null
  },
}))
