import React from 'react'
import { useChatStore } from '../../stores/useChatStore'
import { useAutoScroll } from '../../hooks/useAutoScroll'
import MessageBubble from './MessageBubble'

export default function MessageList() {
  const messageList = useChatStore(state => state.messageList)
  const scrollRef = useAutoScroll([messageList.length])

  return (
    <div className="message-list" ref={scrollRef}>
      {messageList.map((msg, index) => (
        <MessageBubble key={`msg-${msg.send_id}-${msg.created_at}-${index}`} message={msg} />
      ))}
    </div>
  )
}
