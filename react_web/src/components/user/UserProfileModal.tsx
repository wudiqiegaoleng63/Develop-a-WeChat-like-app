import React, { useState, useRef, useEffect } from 'react'
import { useAuthStore } from '../../stores/useAuthStore'
import { showToast } from '../../utils/toast'

interface Props {
  visible: boolean
  onClose: () => void
}

export default function UserProfileModal({ visible, onClose }: Props) {
  const userInfo = useAuthStore(state => state.userInfo)
  const updateProfile = useAuthStore(state => state.updateProfile)
  const uploadAndSetAvatar = useAuthStore(state => state.uploadAndSetAvatar)
  const [nickname, setNickname] = useState(userInfo?.nickname || '')
  const [gender, setGender] = useState(userInfo?.gender ?? -1)
  const [signature, setSignature] = useState(userInfo?.signature || '')
  const [birthday, setBirthday] = useState(userInfo?.birthday || '')
  const [avatarUrl, setAvatarUrl] = useState<string | null>(null)
  const [avatarFile, setAvatarFile] = useState<File | null>(null)
  const [loading, setLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const avatarUrlRef = useRef<string | null>(null)

  // Revoke blob URL on unmount
  useEffect(() => {
    return () => {
      if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current)
    }
  }, [])

  if (!visible || !userInfo) return null

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
    if (!nickname.trim()) { showToast('昵称不能为空', 'error'); return }
    setLoading(true)
    try {
      if (avatarFile) {
        await uploadAndSetAvatar(avatarFile)
      }
      const ok = await updateProfile({
        nickname: nickname.trim(),
        gender: gender === -1 ? undefined : gender,
        signature,
        birthday,
      })
      if (ok) {
        showToast('资料已更新', 'success')
        onClose()
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()} style={{ width: 400 }}>
        <div className="info-modal-header">
          <h3>编辑资料</h3>
          <button className="info-close-btn" onClick={onClose}>×</button>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="info-modal-body">
            <div className="avatar-upload" style={{ marginBottom: 16 }}>
              <div className="avatar-preview" onClick={() => fileInputRef.current?.click()} style={{ width: 64, height: 64 }}>
                {avatarUrl ? <img src={avatarUrl} alt="头像" /> : <img src={userInfo.avatar} alt="头像" />}
              </div>
              <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>点击更换头像</span>
              <input ref={fileInputRef} type="file" accept="image/*" style={{ display: 'none' }} onChange={handleFileChange} />
            </div>
            <div className="form-group">
              <input className="form-input" placeholder="昵称" value={nickname} onChange={e => setNickname(e.target.value)} />
            </div>
            <div className="form-group" style={{ display: 'flex', gap: 16, alignItems: 'center', color: '#333' }}>
              <span style={{ fontSize: 14, color: '#666' }}>性别:</span>
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 14, cursor: 'pointer', color: '#333' }}>
                <input type="radio" name="gender" checked={gender === 0} onChange={() => setGender(0)} /> 男
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 14, cursor: 'pointer', color: '#333' }}>
                <input type="radio" name="gender" checked={gender === 1} onChange={() => setGender(1)} /> 女
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 14, cursor: 'pointer', color: '#333' }}>
                <input type="radio" name="gender" checked={gender === -1} onChange={() => setGender(-1)} /> 未设置
              </label>
            </div>
            <div className="form-group">
              <textarea className="info-textarea" placeholder="个性签名" value={signature} onChange={e => setSignature(e.target.value)} />
            </div>
            <div className="form-group">
              <input className="form-input" type="date" placeholder="生日" value={birthday} onChange={e => setBirthday(e.target.value)} />
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
