package request

// 登录请求
type LoginRequest struct {
	Telephone	string	`json:"telephone" binding:"required"`
	Password	string	`json:"password" binding:"required"`
}

// 注册请求
type RegisterRequest struct {
	Telephone string `json:"telephone" binding:"required"` // 手机号
    Password  string `json:"password" binding:"required"`  // 密码
    Nickname  string `json:"nickname" binding:"required"`  // 昵称
    SmsCode   string `json:"smsCode" binding:"required"`   // 验证码
}

// SmsLoginRequest - 验证码登录请求
type SmsLoginRequest struct {
    Telephone string `json:"telephone" binding:"required"`
    SmsCode   string `json:"smsCode" binding:"required"`
}

// UpdateUserInfoRequest - 更新用户信息请求
type UpdateUserInfoRequest struct {
    Uuid      string `json:"uuid" binding:"required"`
    Nickname  string `json:"nickname"`
    Email     string `json:"email"`
    Avatar    string `json:"avatar"`
    Gender    int8   `json:"gender"`
    Signature string `json:"signature"`
    Birthday  string `json:"birthday"`
}

// GetUserInfoRequest - 获取用户信息请求
type GetUserInfoRequest struct {
    Uuid string `json:"uuid" binding:"required"`
}

// UserUuidsRequest - 用户Uuid列表请求（批量操作）
type UserUuidsRequest struct {
    Uuids []string `json:"uuids" binding:"required"` // 用户Uuid数组
}

// SendEmailCodeRequest 发送邮箱验证码请求
type SendEmailCodeRequest struct {
    Email string `json:"email" binding:"required,email"` // 邮箱地址
}

// VerifyEmailCodeRequest 验证邮箱验证码请求
type VerifyEmailCodeRequest struct {
    Email string `json:"email" binding:"required,email"` // 邮箱地址
    Code  string `json:"code" binding:"required"`        // 验证码
}