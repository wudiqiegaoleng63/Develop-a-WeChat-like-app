export interface GroupInfo {
  uuid: string          // 群组唯一ID (API返回字段名)
  name: string          // 群名称 (API返回字段名)
  group_avatar?: string
  owner_id: string
  member_count?: number
  status: number        // 0=正常, 1=禁用
  is_deleted?: boolean
}

export interface GroupMember {
  uuid: string
  nickname: string
  avatar: string
  status: number
}
