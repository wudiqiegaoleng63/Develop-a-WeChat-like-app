import { create } from 'zustand'
import type { ChatMessage, ChatMessageRequest } from '../types/message'
import { MessageType } from '../types/message'
import type { UserSession, GroupSession } from '../types/session'
import type { ContactInfo } from '../types/user'
import { getUserSessionList, getGroupSessionList, openSession, checkOpenSessionAllowed } from '../api/session'
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
  pendingMessages: Map<string, ChatMessage[]>

  fetchUserSessionList: (ownerId: string) => Promise<void>
  fetchGroupSessionList: (ownerId: string) => Promise<void>
  setActiveChat: (contactId: string, userId: string) => Promise<void>
  sendMessage: (content: string, type: MessageType, userInfo: { uuid: string; nickname: string; avatar: string }, extra?: Partial<ChatMessageRequest>) => void
  addIncomingMessage: (msg: ChatMessage, currentUserId: string) => void
  clearChat: () => void
  resetAll: () => void
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
  pendingMessages: new Map(),

  fetchUserSessionList: async (ownerId) => {
    try {
      const res = await getUserSessionList(ownerId)
      if (res.code === 200 && res.data) {
        set({ userSessionList: res.data.map(normalizeSessionAvatar) })
      }
    } catch (e) {
      console.error('fetchUserSessionList error:', e)
    }
  },

  fetchGroupSessionList: async (ownerId) => {
    try {
      const res = await getGroupSessionList(ownerId)
      if (res.code === 200 && res.data) {
        set({ groupSessionList: res.data.map(normalizeGroupSessionAvatar) })
      }
    } catch (e) {
      console.error('fetchGroupSessionList error:', e)
    }
  },

  setActiveChat: async (contactId, userId) => {
    const generation = ++activeChatGeneration

    try {
      // Check if session is allowed (e.g. not blacklisted)
      const allowedRes = await checkOpenSessionAllowed({ send_id: userId, receive_id: contactId })
      if (generation !== activeChatGeneration) return
      if (allowedRes.code === 200 && allowedRes.data === false) {
        showToast('无法与该用户发起会话', 'error')
        return
      }

      const contactRes = await getContactInfo(contactId)
      if (generation !== activeChatGeneration) return
      // Use contact info if available, otherwise create a minimal fallback
      let contact: ContactInfo
      if (contactRes.code === 200 && contactRes.data) {
        contact = contactRes.data
        contact.contact_avatar = normalizeAvatarUrl(contact.contact_avatar)
      } else {
        const isGroup = contactId.startsWith('G')
        contact = {
          contact_id: contactId,
          contact_name: isGroup ? '群聊' : contactId,
          contact_avatar: '',
        }
      }

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

      // Prepend any pending messages for this chat
      const pending = get().pendingMessages.get(contactId) || []

      set({
        activeContactId: contactId,
        activeSessionId: sessionId,
        contactInfo: contact,
        messageList: [...messages, ...pending],
      })
    } catch (e) {
      console.error('setActiveChat error:', e)
    }
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
    const normalized = normalizeMessageAvatar(msg)

    // Skip own messages (already shown via optimistic update)
    if (msg.send_id === currentUserId) return

    // Determine which chat this message belongs to
    const isGroup = msg.receive_id.startsWith('G')
    const chatId = isGroup ? msg.receive_id : msg.send_id

    // Check if message belongs to currently active chat
    const isForCurrentChat = chatId === activeContactId

    if (isForCurrentChat) {
      set(state => ({ messageList: [...state.messageList, normalized] }))
    } else {
      // Store as pending for when user opens that chat
      set(state => {
        const pending = new Map(state.pendingMessages)
        const existing = pending.get(chatId) || []
        pending.set(chatId, [...existing, normalized])
        return { pendingMessages: pending }
      })
      showToast(`${msg.send_name}: ${msg.content || '发来一条消息'}`, 'info')
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

  resetAll: () => {
    set({
      userSessionList: [],
      groupSessionList: [],
      activeContactId: null,
      activeSessionId: null,
      contactInfo: null,
      messageList: [],
      pendingMessages: new Map(),
    })
  },
}))
