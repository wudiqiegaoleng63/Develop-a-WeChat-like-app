import React from 'react'
import ChatHeader from '../chat/ChatHeader'
import MessageList from '../chat/MessageList'
import ChatInput from '../chat/ChatInput'
import { useChatStore } from '../../stores/useChatStore'

export default function ChatWindow() {
  const activeContactId = useChatStore(state => state.activeContactId)

  if (!activeContactId) {
    return (
      <div className="chat-main">
        <div className="empty-chat">
          选择一个会话开始聊天
        </div>
      </div>
    )
  }

  return (
    <div className="chat-main">
      <ChatHeader />
      <MessageList />
      <ChatInput />
    </div>
  )
}
