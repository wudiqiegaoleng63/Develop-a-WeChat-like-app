package request

// 登录请求
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"` // 邮箱（必填）
    Password string `json:"password" binding:"required"`    // 密码（必填）
}

// 注册请求
type RegisterRequest struct {
    Email     string `json:"email" binding:"required,email"` // 邮箱地址
    Password  string `json:"password" binding:"required"`    // 密码
    Nickname  string `json:"nickname" binding:"required"`    // 昵称
    EmailCode string `json:"emailCode" binding:"required"`   // 邮箱验证码
}

// ============================================================
// EmailLoginRequest - 邮箱验证码登录请求
// ============================================================
type EmailLoginRequest struct {
    Email     string `json:"email" binding:"required,email"` // 邮箱地址
    EmailCode string `json:"emailCode" binding:"required"`   // 邮箱验证码
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

// ============================================================
// GetUserInfoListRequest - 获取用户列表请求（管理员）
// ============================================================
type GetUserInfoListRequest struct {
    OwnerId string `json:"owner_id"` // 管理员uuid（用于排除自己）
}

// ============================================================
// AbleUsersRequest - 批量操作用户请求（包含IsAdmin）
// ============================================================
type AbleUsersRequest struct {
    UuidList []string `json:"uuid_list"` // 用户uuid数组
    IsAdmin  int8     `json:"is_admin"`  // 是否设置管理员
}