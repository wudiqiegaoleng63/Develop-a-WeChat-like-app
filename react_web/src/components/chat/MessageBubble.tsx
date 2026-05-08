import React from 'react'
import type { ChatMessage } from '../../types/message'
import { MessageType } from '../../types/message'
import { useAuthStore } from '../../stores/useAuthStore'

interface Props {
  message: ChatMessage
}

export default function MessageBubble({ message }: Props) {
  const userInfo = useAuthStore(state => state.userInfo)
  const isSent = message.send_id === userInfo?.uuid

  const renderContent = () => {
    switch (message.type) {
      case MessageType.FILE:
        return (
          <div className="file-message">
            <span className="file-icon">📄</span>
            <div className="file-info">
              <div className="file-name">{message.file_name || '文件'}</div>
              <div className="file-size">{message.file_size || '未知大小'}</div>
            </div>
            {message.url && (
              <a href={message.url} target="_blank" rel="noopener noreferrer" className="file-download">↓</a>
            )}
          </div>
        )
      case MessageType.AV:
        return (
          <div className="message-bubble">
            📞 {message.content || '通话'}
          </div>
        )
      default:
        return <div className="message-bubble">{message.content}</div>
    }
  }

  return (
    <div className={`message-item ${isSent ? 'sent' : 'received'}`}>
      <img src={message.send_avatar} alt={message.send_name} className="message-avatar" />
      <div className="message-content">
        {renderContent()}
        <div className="message-time">
          {message.created_at}
        </div>
      </div>
    </div>
  )
}
