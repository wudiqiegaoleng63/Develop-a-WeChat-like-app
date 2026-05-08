import React, { useState, useRef } from 'react'
import { useChatStore } from '../../stores/useChatStore'
import { useAuthStore } from '../../stores/useAuthStore'
import { MessageType } from '../../types/message'
import { uploadFile } from '../../api/message'
import { BACKEND_URL } from '../../utils/constants'
import { showToast } from '../../utils/toast'

export default function ChatInput() {
  const [inputValue, setInputValue] = useState('')
  const userInfo = useAuthStore(state => state.userInfo)
  const sendMessage = useChatStore(state => state.sendMessage)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleSend = () => {
    if (!inputValue.trim() || !userInfo) return
    sendMessage(inputValue.trim(), MessageType.TEXT, {
      uuid: userInfo.uuid,
      nickname: userInfo.nickname,
      avatar: userInfo.avatar || '',
    })
    setInputValue('')
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !userInfo) return

    const res = await uploadFile(file)
    if (res.code === 200) {
      const fileUrl = BACKEND_URL + '/static/files/' + file.name
      sendMessage(file.name, MessageType.FILE, {
        uuid: userInfo.uuid,
        nickname: userInfo.nickname,
        avatar: userInfo.avatar || '',
      }, {
        url: fileUrl,
        file_name: file.name,
        file_size: formatSize(file.size),
        file_type: file.type,
      })
    } else {
      showToast('文件上传失败', 'error')
    }
    // Reset input
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  return (
    <div className="chat-input-area">
      <div className="input-toolbar">
        <button className="toolbar-btn" title="表情">😊</button>
        <button className="toolbar-btn" title="文件" onClick={() => fileInputRef.current?.click()}>📎</button>
        <button className="toolbar-btn" title="图片">🖼</button>
        <button className="toolbar-btn" title="截图">✂</button>
        <input
          ref={fileInputRef}
          type="file"
          style={{ display: 'none' }}
          onChange={handleFileUpload}
        />
      </div>
      <div className="input-wrapper">
        <textarea
          className="message-input"
          placeholder="输入消息..."
          value={inputValue}
          onChange={e => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          rows={1}
        />
        <button
          className="send-btn"
          onClick={handleSend}
          disabled={!inputValue.trim()}
        >
          发送
        </button>
      </div>
    </div>
  )
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + units[i]
}
