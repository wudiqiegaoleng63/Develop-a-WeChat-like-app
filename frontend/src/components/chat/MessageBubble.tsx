import React, { useState, useEffect } from 'react'
import type { ChatMessage } from '../../types/message'
import { MessageType } from '../../types/message'
import { useAuthStore } from '../../stores/useAuthStore'
import { BACKEND_URL } from '../../utils/constants'

async function downloadFile(url: string, fileName: string) {
  try {
    const token = localStorage.getItem('token')
    const headers: Record<string, string> = {}
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
    const rsp = await fetch(url.startsWith('http') ? url : BACKEND_URL + url, { headers })
    const blob = await rsp.blob()
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = fileName
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(link.href)
  } catch (e) {
    console.error('Download failed:', e)
  }
}

interface Props {
  message: ChatMessage
}

export default function MessageBubble({ message }: Props) {
  const userInfo = useAuthStore(state => state.userInfo)
  const isSent = message.send_id === userInfo?.uuid
  const [imageSrc, setImageSrc] = useState<string | undefined>(undefined)

  // 带认证加载图片
  useEffect(() => {
    if (message.type === MessageType.FILE && message.file_type?.startsWith('image/') && message.url) {
      const token = localStorage.getItem('token')
      const headers: Record<string, string> = {}
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }
      const url = message.url.startsWith('http') ? message.url : BACKEND_URL + message.url
      fetch(url, { headers })
        .then(rsp => rsp.blob())
        .then(blob => {
          const objectUrl = URL.createObjectURL(blob)
          setImageSrc(objectUrl)
          return () => URL.revokeObjectURL(objectUrl)
        })
        .catch(() => {
          setImageSrc(url) // fallback to direct URL
        })
    }
  }, [message.url, message.type, message.file_type])

  const renderContent = () => {
    switch (message.type) {
      case MessageType.FILE:
        if (message.file_type?.startsWith('image/') && message.url) {
          return (
            <div className="message-bubble" style={{ padding: 4, background: 'transparent' }}>
              {imageSrc && <img src={imageSrc} alt={message.file_name || '图片'} className="chat-image" />}
            </div>
          )
        }
        return (
          <div className="message-bubble">
            <div className="file-message">
              <span className="file-icon">📄</span>
              <div className="file-info">
                <div className="file-name">{message.file_name || '文件'}</div>
                <div className="file-size">{message.file_size || '未知大小'}</div>
              </div>
              {message.url && (
                <button className="file-download" onClick={() => downloadFile(message.url!, message.file_name || '文件')}>↓</button>
              )}
            </div>
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
