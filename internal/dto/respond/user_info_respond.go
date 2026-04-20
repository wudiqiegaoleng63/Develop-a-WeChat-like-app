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