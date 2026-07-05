import axios from 'axios'
import { BACKEND_URL } from '../utils/constants'
import type { ApiResponse } from '../types/api'
import { useAuthStore } from '../stores/useAuthStore'

const api = axios.create({
  baseURL: BACKEND_URL,
  timeout: 10000,
})

// 请求拦截器：自动附加JWT token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  if (config.data instanceof FormData) {
    delete config.headers['Content-Type']
  } else {
    config.headers['Content-Type'] = 'application/json'
  }
  return config
})

// 响应拦截器：处理401/403
api.interceptors.response.use(
  (response) => {
    const data = response.data as ApiResponse
    if (data.code === 401) {
      // token无效或过期，清除登录状态，跳转登录页
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
      useAuthStore.setState({ userInfo: null })
      window.location.href = '/login'
      return Promise.reject(new Error(data.message || '未认证'))
    }
    if (data.code === 403) {
      console.warn('[API Forbidden]', data.message)
      return Promise.reject(new Error(data.message || '无权限'))
    }
    if (data.code === 400) {
      console.warn('[API Warning]', data.message)
    } else if (data.code === 500) {
      console.error('[API Error]', data.message)
    }
    return response
  },
  (error) => {
    // 处理后端返回的HTTP 401/403状态码
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
      useAuthStore.setState({ userInfo: null })
      window.location.href = '/login'
      return Promise.reject(new Error(error.response.data?.message || '未认证'))
    }
    if (error.response?.status === 403) {
      console.warn('[API Forbidden]', error.response.data?.message)
      return Promise.reject(new Error(error.response.data?.message || '无权限'))
    }
    console.error('[API Error]', '网络请求失败')
    return Promise.reject(error)
  }
)

export default api
