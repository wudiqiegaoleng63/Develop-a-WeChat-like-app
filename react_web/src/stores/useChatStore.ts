import { create } from 'zustand'
import type { ChatMessage, ChatMessageRequest } from '../types/message'
import { MessageType } from '../types/message'
import type { UserSession, GroupSession } from '../types/session'
import type { ContactInfo } from '../types/user'
import { getUserSessionList, getGroupSessionList, openSession } from '../api/session'
import { getContactInfo } from '../api/contact'
import { getMessageList, getGroupMessageList } from '../api/message'
import { normalizeAvatarUrl } from '../utils/avatar'
import { wsService } from '../services/websocket'
import { showToast } from '../utils/toast'

let activeChatGeneration = 0

interface ChatState {
  userSessionList: UserSession[]
  groupSessionList: GroupSession[]
  activeContactId: string | null
  activeSessionId: string | null
  contactInfo: ContactInfo | null
  messageList: ChatMessage[]

  fetchUserSessionList: (ownerId: string) => Promise<void>
  fetchGroupSessionList: (ownerId: string) => Promise<void>
  setActiveChat: (contactId: string, userId: string) => Promise<void>
  sendMessage: (content: string, type: MessageType, userInfo: { uuid: string; nickname: string; avatar: string }, extra?: Partial<ChatMessageRequest>) => void
  addIncomingMessage: (msg: ChatMessage, currentUserId: string) => void
  clearChat: () => void
}

function normalizeSessionAvatar(session: UserSession): UserSession {
  return { ...session, avatar: normalizeAvatarUrl(session.avatar) }
}

function normalizeGroupSessionAvatar(session: GroupSession): GroupSession {
  return { ...session, avatar: normalizeAvatarUrl(session.avatar) }
}

function normalizeMessageAvatar(msg: ChatMessage): ChatMessage {
  return { ...msg, send_avatar: normalizeAvatarUrl(msg.send_avatar) }
}

export const useChatStore = create<ChatState>((set, get) => ({
  userSessionList: [],
  groupSessionList: [],
  activeContactId: null,
  activeSessionId: null,
  contactInfo: null,
  messageList: [],

  fetchUserSessionList: async (ownerId) => {
    const res = await getUserSessionList(ownerId)
    if (res.code === 200 && res.data) {
      set({ userSessionList: res.data.map(normalizeSessionAvatar) })
    }
  },

  fetchGroupSessionList: async (ownerId) => {
    const res = await getGroupSessionList(ownerId)
    if (res.code === 200 && res.data) {
      set({ groupSessionList: res.data.map(normalizeGroupSessionAvatar) })
    }
  },

  setActiveChat: async (contactId, userId) => {
    const generation = ++activeChatGeneration

    const contactRes = await getContactInfo(contactId)
    if (generation !== activeChatGeneration) return
    if (contactRes.code !== 200 || !contactRes.data) return
    const contact = contactRes.data
    contact.contact_avatar = normalizeAvatarUrl(contact.contact_avatar)

    const sessionRes = await openSession({ send_id: userId, receive_id: contactId })
    if (generation !== activeChatGeneration) return
    if (sessionRes.code !== 200 || !sessionRes.data) return
    const sessionId = sessionRes.data

    let messages: ChatMessage[] = []
    if (contactId.startsWith('G')) {
      const msgRes = await getGroupMessageList(contactId)
      if (generation !== activeChatGeneration) return
      if (msgRes.code === 200 && msgRes.data) {
        messages = msgRes.data.map(normalizeMessageAvatar)
      }
    } else {
      const msgRes = await getMessageList({ user_one_id: userId, user_two_id: contactId })
      if (generation !== activeChatGeneration) return
      if (msgRes.code === 200 && msgRes.data) {
        messages = msgRes.data.map(normalizeMessageAvatar)
      }
    }

    set({
      activeContactId: contactId,
      activeSessionId: sessionId,
      contactInfo: contact,
      messageList: messages,
    })
  },

  sendMessage: (content, type, userInfo, extra = {}) => {
    const { activeSessionId, activeContactId } = get()
    if (!activeSessionId || !activeContactId) return

    if (!wsService.connected) {
      showToast('网络连接已断开，消息发送失败', 'error')
      return
    }

    const request: ChatMessageRequest = {
      session_id: activeSessionId,
      type,
      content,
      url: extra?.url || '',
      send_id: userInfo.uuid,
      send_name: userInfo.nickname,
      send_avatar: userInfo.avatar,
      receive_id: activeContactId,
      file_size: extra?.file_size || '0B',
      file_name: extra?.file_name || '',
      file_type: extra?.file_type || '',
    }

    const localMsg: ChatMessage = {
      uuid: `local-${Date.now()}`,
      session_id: activeSessionId,
      send_id: userInfo.uuid,
      send_name: userInfo.nickname,
      send_avatar: userInfo.avatar,
      receive_id: activeContactId,
      type,
      content,
      url: extra?.url || '',
      file_size: extra?.file_size || '0B',
      file_name: extra?.file_name || '',
      file_type: extra?.file_type || '',
      created_at: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
      status: 1,
    }
    set(state => ({ messageList: [...state.messageList, localMsg] }))

    wsService.send(request)
  },

  addIncomingMessage: (msg, currentUserId) => {
    const { activeContactId } = get()
    if (!activeContactId) return

    const normalized = normalizeMessageAvatar(msg)

    // Show message if it belongs to current chat (same logic as Vue version)
    // Note: own messages (send_id === currentUserId) are already shown via optimistic update,
    // so we only add incoming messages from others
    const isForCurrentChat =
      (msg.receive_id.startsWith('G') && msg.receive_id === activeContactId && msg.send_id !== currentUserId) ||
      (msg.receive_id.startsWith('U') && msg.receive_id === currentUserId && msg.send_id === activeContactId)

    if (isForCurrentChat) {
      set(state => ({ messageList: [...state.messageList, normalized] }))
    }
  },

  clearChat: () => {
    set({
      activeContactId: null,
      activeSessionId: null,
      contactInfo: null,
      messageList: [],
    })
  },
}))
