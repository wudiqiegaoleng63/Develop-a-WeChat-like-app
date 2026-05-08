import { create } from 'zustand'
import type { UserInfo } from '../types/user'
import { login as loginApi, register as registerApi, emailLogin as emailLoginApi } from '../api/auth'
import { wsLogout } from '../api/user'
import type { LoginRequest, RegisterRequest, EmailLoginRequest } from '../types/api'
import { normalizeAvatarUrl } from '../utils/avatar'
import { wsService } from '../services/websocket'
import { WS_URL } from '../utils/constants'
import { showToast } from '../utils/toast'

interface AuthState {
  userInfo: UserInfo | null
  login: (data: LoginRequest) => Promise<boolean>
  emailLogin: (data: EmailLoginRequest) => Promise<boolean>
  register: (data: RegisterRequest) => Promise<boolean>
  logout: () => void
}

function normalizeUserInfo(info: UserInfo): UserInfo {
  return {
    ...info,
    avatar: normalizeAvatarUrl(info.avatar || ''),
  }
}

// Initialize from sessionStorage (like Vue version)
function getInitialUserInfo(): UserInfo | null {
  const stored = sessionStorage.getItem('userInfo')
  if (stored) {
    try {
      const user = JSON.parse(stored) as UserInfo
      // Reconnect WebSocket
      wsService.connect(user.uuid, WS_URL)
      return user
    } catch {
      sessionStorage.removeItem('userInfo')
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
      sessionStorage.setItem('userInfo', JSON.stringify(user))
      wsService.connect(user.uuid, WS_URL)
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
      sessionStorage.setItem('userInfo', JSON.stringify(user))
      wsService.connect(user.uuid, WS_URL)
      set({ userInfo: user })
      return true
    }
    showToast(res.message || '登录失败', 'error')
    return false
  },

  register: async (data) => {
    const res = await registerApi(data)
    if (res.code === 200) return true
    showToast(res.message || '注册失败', 'error')
    return false
  },

  logout: () => {
    const { userInfo } = get()
    if (userInfo) {
      wsLogout(userInfo.uuid).catch(() => {})
    }
    wsService.disconnect()
    sessionStorage.removeItem('userInfo')
    set({ userInfo: null })
  },
}))
