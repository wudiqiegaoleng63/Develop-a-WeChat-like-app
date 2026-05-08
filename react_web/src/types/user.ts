export interface UserInfo {
  uuid: string
  nickname: string
  email?: string
  avatar?: string
  telephone?: string
  status: number       // 0=online, 1=offline
  is_admin: number     // 0=normal, 1=admin (API返回snake_case)
  is_deleted?: boolean
}

export interface ContactInfo {
  contact_id: string
  contact_name: string
  contact_avatar: string
  contact_phone: string
  contact_email: string
  contact_gender: number
  contact_signature: string
  contact_birthday: string
  contact_notice: string
  contact_members: string // JSON string
  contact_member_cnt: number
  contact_owner_id: string
  contact_add_mode: number
}

export interface ContactUser {
  uuid: string
  nickname: string
  avatar: string
  status: number
}
