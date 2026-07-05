import { create } from 'zustand'
import type { SearchUser, ContactRequest, JoinedGroup } from '../types/user'
import {
  getUserList,
  getNewContactList,
  loadMyJoinedGroup,
  applyContact,
  passContactApply,
  refuseContactApply,
} from '../api/contact'

interface ContactState {
  searchUsers: SearchUser[]
  friendRequests: ContactRequest[]
  myGroups: JoinedGroup[]
  loadingSearch: boolean
  loadingRequests: boolean
  loadingGroups: boolean

  fetchSearchUsers: (ownerId: string) => Promise<void>
  fetchFriendRequests: (ownerId: string) => Promise<void>
  fetchMyGroups: (ownerId: string) => Promise<void>
  applyFriend: (ownerId: string, contactId: string) => Promise<boolean>
  passRequest: (ownerId: string, contactId: string) => Promise<boolean>
  refuseRequest: (ownerId: string, contactId: string) => Promise<boolean>
}

export const useContactStore = create<ContactState>((set, get) => ({
  searchUsers: [],
  friendRequests: [],
  myGroups: [],
  loadingSearch: false,
  loadingRequests: false,
  loadingGroups: false,

  fetchSearchUsers: async (ownerId) => {
    set({ loadingSearch: true })
    try {
      const res = await getUserList(ownerId)
      if (res.code === 200 && res.data) {
        const seen = new Set<string>()
        const unique = res.data.filter(u => {
          if (seen.has(u.user_id)) return false
          seen.add(u.user_id)
          return true
        })
        set({ searchUsers: unique })
      }
    } finally {
      set({ loadingSearch: false })
    }
  },

  fetchFriendRequests: async (ownerId) => {
    set({ loadingRequests: true })
    try {
      const res = await getNewContactList(ownerId)
      if (res.code === 200 && res.data) {
        const seen = new Set<string>()
        const unique = res.data.filter(r => {
          if (seen.has(r.contact_id)) return false
          seen.add(r.contact_id)
          return true
        })
        set({ friendRequests: unique })
      }
    } finally {
      set({ loadingRequests: false })
    }
  },

  fetchMyGroups: async (ownerId) => {
    set({ loadingGroups: true })
    try {
      const res = await loadMyJoinedGroup(ownerId)
      if (res.code === 200 && res.data) {
        const seen = new Set<string>()
        const unique = res.data.filter(g => {
          if (seen.has(g.group_id)) return false
          seen.add(g.group_id)
          return true
        })
        set({ myGroups: unique })
      }
    } finally {
      set({ loadingGroups: false })
    }
  },

  applyFriend: async (ownerId, contactId) => {
    const res = await applyContact({ owner_id: ownerId, contact_id: contactId })
    return res.code === 200
  },

  passRequest: async (ownerId, contactId) => {
    const res = await passContactApply({ owner_id: ownerId, contact_id: contactId })
    if (res.code === 200) {
      get().fetchFriendRequests(ownerId)
      return true
    }
    return false
  },

  refuseRequest: async (ownerId, contactId) => {
    const res = await refuseContactApply({ owner_id: ownerId, contact_id: contactId })
    if (res.code === 200) {
      get().fetchFriendRequests(ownerId)
      return true
    }
    return false
  },
}))
