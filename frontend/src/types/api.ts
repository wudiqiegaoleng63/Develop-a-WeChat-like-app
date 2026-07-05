export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data?: T
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  nickname: string
  email: string
  password: string
  emailCode: string
}

export interface EmailLoginRequest {
  email: string
  emailCode: string
}

export interface SendEmailCodeRequest {
  email: string
}
