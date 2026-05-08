import React, { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import Sidebar from '../components/layout/Sidebar'
import SessionList from '../components/layout/SessionList'
import ChatWindow from '../components/layout/ChatWindow'
import { useAuthStore } from '../stores/useAuthStore'
import { useChatStore } from '../stores/useChatStore'
import { useWebSocket } from '../hooks/useWebSocket'

export default function ChatPage() {
  const { id: contactId } = useParams()
  const userInfo = useAuthStore(state => state.userInfo)
  const setActiveChat = useChatStore(state => state.setActiveChat)
  const clearChat = useChatStore(state => state.clearChat)

  // Subscribe to WebSocket messages
  useWebSocket()

  useEffect(() => {
    if (contactId && userInfo) {
      setActiveChat(contactId, userInfo.uuid)
    } else {
      clearChat()
    }
  }, [contactId, userInfo, setActiveChat, clearChat])

  return (
    <div className="chat-page">
      <Sidebar />
      <SessionList />
      <ChatWindow />
    </div>
  )
}
