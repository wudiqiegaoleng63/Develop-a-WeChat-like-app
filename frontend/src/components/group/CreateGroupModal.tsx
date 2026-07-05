import React, { useState, useRef, useEffect } from 'react'
import { useAuthStore } from '../../stores/useAuthStore'
import { createGroup, loadMyGroup } from '../../api/group'
import { openSession } from '../../api/session'
import { uploadAvatar } from '../../api/message'
import { showToast } from '../../utils/toast'

interface Props {
  visible: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function CreateGroupModal({ visible, onClose, onSuccess }: Props) {
  const userInfo = useAuthStore(state => state.userInfo)
  const [name, setName] = useState('')
  const [notice, setNotice] = useState('')
  const [addMode, setAddMode] = useState(0)
  const [avatarUrl, setAvatarUrl] = useState<string | null>(null)
  const [avatarFile, setAvatarFile] = useState<File | null>(null)
  const [loading, setLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Revoke blob URL on unmount
  useEffect(() => {
    return () => {
      if (avatarUrl) URL.revokeObjectURL(avatarUrl)
    }
  }, [avatarUrl])

  if (!visible) return null

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setAvatarFile(file)
      if (avatarUrl) URL.revokeObjectURL(avatarUrl)
      setAvatarUrl(URL.createObjectURL(file))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) { showToast('请输入群名称', 'error'); return }
    if (!userInfo) return
    setLoading(true)
    try {
      let avatar = ''
      if (avatarFile) {
        console.log('[CreateGroup] Uploading avatar:', avatarFile.name)
        const uploadRes = await uploadAvatar(avatarFile)
        console.log('[CreateGroup] Upload response:', uploadRes)
        if (uploadRes.code === 200) {
          avatar = '/static/avatars/' + avatarFile.name
          console.log('[CreateGroup] Avatar URL:', avatar)
        } else {
          showToast('头像上传失败: ' + (uploadRes.message || ''), 'error')
          return
        }
      }
      console.log('[CreateGroup] Creating group with:', { name: name.trim(), owner_id: userInfo.uuid, notice, add_mode: addMode, avatar })
      const res = await createGroup({
        name: name.trim(),
        owner_id: userInfo.uuid,
        notice,
        add_mode: addMode,
        avatar,
      })
      console.log('[CreateGroup] Create response:', res)
      if (res.code === 200) {
        showToast('群组创建成功', 'success')
        // createGroup doesn't return group ID, fetch my groups to find it
        if (userInfo) {
          const myGroupsRes = await loadMyGroup(userInfo.uuid)
          if (myGroupsRes.code === 200 && myGroupsRes.data && myGroupsRes.data.length > 0) {
            const newestGroup = myGroupsRes.data[0]
            await openSession({ send_id: userInfo.uuid, receive_id: newestGroup.group_id })
          }
        }
        onSuccess()
        onClose()
        setName(''); setNotice(''); setAddMode(0)
        if (avatarUrl) URL.revokeObjectURL(avatarUrl)
        setAvatarUrl(null); setAvatarFile(null)
      } else {
        showToast(res.message || '创建失败', 'error')
      }
    } catch (err) {
      console.error('[CreateGroup] Error:', err)
      showToast('操作失败: ' + (err instanceof Error ? err.message : String(err)), 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()} style={{ width: 400 }}>
        <div className="info-modal-header">
          <h3>创建群组</h3>
          <button className="info-close-btn" onClick={onClose}>×</button>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="info-modal-body">
            <div className="avatar-upload" style={{ marginBottom: 16 }}>
              <div className="avatar-preview" onClick={() => fileInputRef.current?.click()} style={{ width: 64, height: 64 }}>
                {avatarUrl ? <img src={avatarUrl} alt="群头像" /> : <span>📷</span>}
              </div>
              <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>点击上传群头像</span>
              <input ref={fileInputRef} type="file" accept="image/*" style={{ display: 'none' }} onChange={handleFileChange} />
            </div>
            <div className="form-group">
              <input className="form-input" placeholder="群名称（必填）" value={name} onChange={e => setName(e.target.value)} />
            </div>
            <div className="form-group">
              <textarea className="info-textarea" placeholder="群公告（可选）" value={notice} onChange={e => setNotice(e.target.value)} />
            </div>
            <div className="form-group" style={{ display: 'flex', gap: 16, alignItems: 'center', color: '#333' }}>
              <span style={{ fontSize: 14, color: '#666' }}>加群方式:</span>
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 14, cursor: 'pointer', color: '#333' }}>
                <input type="radio" name="addMode" checked={addMode === 0} onChange={() => setAddMode(0)} /> 直接加入
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 14, cursor: 'pointer', color: '#333' }}>
                <input type="radio" name="addMode" checked={addMode === 1} onChange={() => setAddMode(1)} /> 需审核
              </label>
            </div>
          </div>
          <div style={{ padding: '12px 24px 20px', display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="btn-action" onClick={onClose}>取消</button>
            <button type="submit" className="btn-action primary" disabled={loading}>{loading ? '创建中...' : '创建'}</button>
          </div>
        </form>
      </div>
    </div>
  )
}
