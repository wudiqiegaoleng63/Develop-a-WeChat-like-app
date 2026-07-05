export interface GroupInfo {
  uuid: string
  name: string
  avatar?: string
  notice?: string
  member_cnt?: number
  owner_id: string
  add_mode?: number     // 0=直接加入, 1=需审核
  status: number        // 0=正常, 1=禁用
  is_deleted?: boolean
}

// Group member from getGroupMemberList (backend returns user_id, not uuid)
export interface GroupMember {
  user_id: string
  nickname: string
  avatar: string
}
