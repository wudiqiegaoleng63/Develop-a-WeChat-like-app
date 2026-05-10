import React, { useState, useRef, useEffect } from 'react'
import { useChatStore } from '../../stores/useChatStore'
import { useAuthStore } from '../../stores/useAuthStore'
import { MessageType } from '../../types/message'
import { uploadFile } from '../../api/message'
import { BACKEND_URL } from '../../utils/constants'
import { showToast } from '../../utils/toast'
import EmojiPicker from './EmojiPicker'

// Common image extensions as fallback when file.type is empty
const IMAGE_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg', '.heic', '.avif']

function isImageFile(file: File): boolean {
  if (file.type && file.type.startsWith('image/')) return true
  const name = file.name.toLowerCase()
  return IMAGE_EXTENSIONS.some(ext => name.endsWith(ext))
}

export default function ChatInput() {
  const [inputValue, setInputValue] = useState('')
  const [emojiPickerVisible, setEmojiPickerVisible] = useState(false)
  const userInfo = useAuthStore(state => state.userInfo)
  const sendMessage = useChatStore(state => state.sendMessage)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const emojiRef = useRef<HTMLDivElement>(null)

  // Close emoji picker on outside click
  useEffect(() => {
    if (!emojiPickerVisible) return
    const handler = (e: MouseEvent) => {
      if (emojiRef.current && !emojiRef.current.contains(e.target as Node)) {
        setEmojiPickerVisible(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [emojiPickerVisible])

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
      const fileUrl = BACKEND_URL + '/static/files/' + encodeURIComponent(file.name)
      sendMessage(file.name, MessageType.FILE, {
        uuid: userInfo.uuid,
        nickname: userInfo.nickname,
        avatar: userInfo.avatar || '',
      }, {
        url: fileUrl,
        file_name: file.name,
        file_size: formatSize(file.size),
        file_type: file.type || 'application/octet-stream',
      })
    } else {
      showToast('文件上传失败', 'error')
    }
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !userInfo) return

    const res = await uploadFile(file)
    if (res.code === 200) {
      const fileUrl = BACKEND_URL + '/static/files/' + encodeURIComponent(file.name)
      // Use file.type if available; fallback to image/png heuristic
      const fileType = file.type || (isImageFile(file) ? 'image/png' : 'application/octet-stream')
      sendMessage(file.name, MessageType.FILE, {
        uuid: userInfo.uuid,
        nickname: userInfo.nickname,
        avatar: userInfo.avatar || '',
      }, {
        url: fileUrl,
        file_name: file.name,
        file_size: formatSize(file.size),
        file_type: fileType,
      })
    } else {
      showToast('图片上传失败', 'error')
    }
    if (imageInputRef.current) imageInputRef.current.value = ''
  }

  const handleEmojiSelect = (emoji: string) => {
    setInputValue(prev => prev + emoji)
    setEmojiPickerVisible(false)
  }

  return (
    <div className="chat-input-area">
      <div className="input-toolbar" style={{ position: 'relative' }}>
        <div ref={emojiRef} style={{ display: 'contents' }}>
          <button className="toolbar-btn" title="表情" onClick={() => setEmojiPickerVisible(!emojiPickerVisible)}>😊</button>
          {emojiPickerVisible && <EmojiPicker onSelect={handleEmojiSelect} />}
        </div>
        <button className="toolbar-btn" title="文件" onClick={() => fileInputRef.current?.click()}>📎</button>
        <button className="toolbar-btn" title="图片" onClick={() => imageInputRef.current?.click()}>🖼</button>
        <input
          ref={fileInputRef}
          type="file"
          style={{ display: 'none' }}
          onChange={handleFileUpload}
        />
        <input
          ref={imageInputRef}
          type="file"
          accept="image/*"
          style={{ display: 'none' }}
          onChange={handleImageUpload}
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
