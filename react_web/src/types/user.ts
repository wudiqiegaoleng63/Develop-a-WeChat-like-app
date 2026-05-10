export interface UserInfo {
  uuid: string
  nickname: string
  email?: string
  avatar?: string
  telephone?: string
  status: number       // 0=正常, 1=禁用
  isAdmin: number      // 0=普通, 1=管理员
  gender?: number      // 0=男, 1=女
  signature?: string   // 个性签名
  birthday?: string    // 生日
  createdAt?: string   // 注册时间
}

// Full contact info from getContactInfo API
export interface ContactInfo {
  contact_id: string
  contact_name: string
  contact_avatar: string
  contact_phone?: string
  contact_email?: string
  contact_gender?: number
  contact_signature?: string
  contact_birthday?: string
  contact_notice?: string
  contact_members?: string // JSON string
  contact_member_cnt?: number
  contact_owner_id?: string
  contact_add_mode?: number
}

// Search user result from getUserList API (backend returns user_id/user_name)
export interface SearchUser {
  user_id: string
  user_name: string
  avatar: string
}

// Friend request / add group request from getNewContactList / getAddGroupList
export interface ContactRequest {
  contact_id: string
  contact_name: string
  contact_avatar: string
  message?: string
}

// Joined group from loadMyJoinedGroup API (backend returns group_id/group_name)
export interface JoinedGroup {
  group_id: string
  group_name: string
  avatar: string
}
