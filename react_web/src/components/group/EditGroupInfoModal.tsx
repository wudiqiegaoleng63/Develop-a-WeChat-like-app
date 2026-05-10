import React, { useState, useEffect, useRef } from 'react'
import { useAuthStore } from '../../stores/useAuthStore'
import { useChatStore } from '../../stores/useChatStore'
import { getGroupInfo, updateGroupInfo } from '../../api/group'
import { uploadAvatar } from '../../api/message'
import { showToast } from '../../utils/toast'
import { normalizeAvatarUrl } from '../../utils/avatar'

interface Props {
  visible: boolean
  onClose: () => void
}

export default function EditGroupInfoModal({ visible, onClose }: Props) {
  const userInfo = useAuthStore(state => state.userInfo)
  const contactInfo = useChatStore(state => state.contactInfo)
  const [name, setName] = useState('')
  const [notice, setNotice] = useState('')
  const [addMode, setAddMode] = useState(0)
  const [avatar, setAvatar] = useState('')
  const [avatarFile, setAvatarFile] = useState<File | null>(null)
  const [avatarUrl, setAvatarUrl] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const avatarUrlRef = useRef<string | null>(null)

  // Revoke blob URL on unmount
  useEffect(() => {
    return () => {
      if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current)
    }
  }, [])

  useEffect(() => {
    if (visible && contactInfo) {
      getGroupInfo(contactInfo.contact_id).then(res => {
        if (res.code === 200 && res.data) {
          setName(res.data.name)
          setNotice(res.data.notice || '')
          setAddMode(res.data.add_mode || 0)
          setAvatar(normalizeAvatarUrl(res.data.avatar || ''))
        }
      })
      setAvatarFile(null)
      if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current)
      avatarUrlRef.current = null
      setAvatarUrl(null)
    }
  }, [visible, contactInfo])

  if (!visible || !contactInfo || !userInfo) return null

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setAvatarFile(file)
      if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current)
      const url = URL.createObjectURL(file)
      avatarUrlRef.current = url
      setAvatarUrl(url)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) { showToast('群名称不能为空', 'error'); return }
    setLoading(true)
    try {
      let finalAvatar = avatar
      if (avatarFile) {
        const uploadRes = await uploadAvatar(avatarFile)
        if (uploadRes.code === 200) {
          finalAvatar = '/static/avatars/' + avatarFile.name
        }
      }
      const res = await updateGroupInfo({
        owner_id: userInfo.uuid,
        uuid: contactInfo.contact_id,
        name: name.trim(),
        notice,
        add_mode: addMode,
        avatar: finalAvatar,
      })
      if (res.code === 200) {
        showToast('群信息已更新', 'success')
        onClose()
      } else {
        showToast(res.message || '更新失败', 'error')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()} style={{ width: 400 }}>
        <div className="info-modal-header">
          <h3>修改群聊信息</h3>
          <button className="info-close-btn" onClick={onClose}>×</button>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="info-modal-body">
            <div className="avatar-upload" style={{ marginBottom: 16 }}>
              <div className="avatar-preview" onClick={() => fileInputRef.current?.click()} style={{ width: 64, height: 64 }}>
                {avatarUrl ? <img src={avatarUrl} alt="群头像" /> : avatar ? <img src={avatar} alt="群头像" /> : <span>📷</span>}
              </div>
              <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>点击更换群头像</span>
              <input ref={fileInputRef} type="file" accept="image/*" style={{ display: 'none' }} onChange={handleFileChange} />
            </div>
            <div className="form-group">
              <input className="form-input" placeholder="群名称" value={name} onChange={e => setName(e.target.value)} />
            </div>
            <div className="form-group">
              <textarea className="info-textarea" placeholder="群公告" value={notice} onChange={e => setNotice(e.target.value)} />
            </div>
            <div className="form-group" style={{ display: 'flex', gap: 16, alignItems: 'center', color: '#333' }}>
              <span style={{ fontSize: 14, color: '#666' }}>加群方式:</span>
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 14, cursor: 'pointer', color: '#333' }}>
                <input type="radio" name="editAddMode" checked={addMode === 0} onChange={() => setAddMode(0)} /> 直接加入
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 14, cursor: 'pointer', color: '#333' }}>
                <input type="radio" name="editAddMode" checked={addMode === 1} onChange={() => setAddMode(1)} /> 需审核
              </label>
            </div>
          </div>
          <div style={{ padding: '12px 24px 20px', display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="btn-action" onClick={onClose}>取消</button>
            <button type="submit" className="btn-action primary" disabled={loading}>{loading ? '保存中...' : '保存'}</button>
          </div>
        </form>
      </div>
    </div>
  )
}
