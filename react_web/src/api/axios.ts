import axios from 'axios'
import { BACKEND_URL } from '../utils/constants'
import type { ApiResponse } from '../types/api'

const api = axios.create({
  baseURL: BACKEND_URL,
  timeout: 10000,
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.response.use(
  (response) => {
    const data = response.data as ApiResponse
    if (data.code === 400) {
      console.warn('[API Warning]', data.message)
    } else if (data.code === 500) {
      console.error('[API Error]', data.message)
    }
    return response
  },
  (error) => {
    console.error('[API Error]', '网络请求失败')
    return Promise.reject(error)
  }
)

export default api
