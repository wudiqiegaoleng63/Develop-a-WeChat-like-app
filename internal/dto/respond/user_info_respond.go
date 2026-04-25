package respond

// ============================================================
// LoginRespond - 登录响应
// ============================================================
// ★返回给前端的用户信息
type LoginRespond struct {
    Uuid      string `json:"uuid"`
    Telephone string `json:"telephone"`
    Nickname  string `json:"nickname"`
    Email     string `json:"email"`
    Avatar    string `json:"avatar"`
    Gender    int8   `json:"gender"`
    Signature string `json:"signature"`
    Birthday  string `json:"birthday"`
    IsAdmin   int8   `json:"isAdmin"`
    Status    int8   `json:"status"`      // ★用户状态
    CreatedAt string `json:"createdAt"`   // ★创建时间
}

// ============================================================
// RegisterRespond - 注册响应
// ============================================================
type RegisterRespond struct {
    Uuid      string `json:"uuid"`
    Telephone string `json:"telephone"`
    Nickname  string `json:"nickname"`
    Email     string `json:"email"`
    Avatar    string `json:"avatar"`
    Gender    int8   `json:"gender"`
    Birthday  string `json:"birthday"`
    Signature string `json:"signature"`
    IsAdmin   int8   `json:"isAdmin"`
    Status    int8   `json:"status"`
    CreatedAt string `json:"createdAt"`
}

// ============================================================
// GetUserInfoRespond - 获取用户信息响应
// ============================================================
type GetUserInfoRespond struct {
    Uuid      string `json:"uuid"`
    Telephone string `json:"telephone"`
    Nickname  string `json:"nickname"`
    Email     string `json:"email"`
    Avatar    string `json:"avatar"`
    Gender    int8   `json:"gender"`
    Signature string `json:"signature"`
    Birthday  string `json:"birthday"`
    IsAdmin   int8   `json:"isAdmin"`
    Status    int8   `json:"status"`
    CreatedAt string `json:"createdAt"`
}

// ============================================================
// UserListItemRespond - 用户列表项（管理员功能）
// ============================================================
type UserListItemRespond struct {
    Uuid      string `json:"uuid"`
    Telephone string `json:"telephone"`
    Nickname  string `json:"nickname"`
    IsAdmin   int8   `json:"isAdmin"`
    Status    int8   `json:"status"`
    DeletedAt string `json:"deletedAt"` // ★软删除时间
}

// ============================================================
// GetUserListRespond - 用户列表项响应（管理员）
// ============================================================
type GetUserListRespond struct {
    Uuid      string `json:"uuid"`       // 用户唯一标识
    Telephone string `json:"telephone"`  // 手机号
    Nickname  string `json:"nickname"`   // 昵称
    Status    int8   `json:"status"`      // 状态：0=正常，1=禁用
    IsAdmin   int8   `json:"is_admin"`    // 是否管理员
    IsDeleted bool   `json:"is_deleted"`  // 是否已软删除
}