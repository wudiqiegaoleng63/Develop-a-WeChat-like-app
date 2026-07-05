import React, { useState, useEffect } from 'react'
import { Modal } from 'antd'
import { useAuthStore } from '../../stores/useAuthStore'
import { useChatStore } from '../../stores/useChatStore'
import { getGroupMemberList, removeGroupMembers } from '../../api/group'
import { showToast } from '../../utils/toast'
import { normalizeAvatarUrl } from '../../utils/avatar'
import type { GroupMember } from '../../types/group'

interface Props {
  visible: boolean
  onClose: () => void
}

export default function RemoveGroupMembersModal({ visible, onClose }: Props) {
  const userInfo = useAuthStore(state => state.userInfo)
  const contactInfo = useChatStore(state => state.contactInfo)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (visible && contactInfo) {
      setLoading(true)
      getGroupMemberList(contactInfo.contact_id).then(res => {
        if (res.code === 200 && res.data) setMembers(res.data)
      }).finally(() => setLoading(false))
    }
  }, [visible, contactInfo])

  if (!visible || !contactInfo || !userInfo) return null

  const handleRemove = (member: GroupMember) => {
    Modal.confirm({
      title: '确认移除成员',
      content: `确定要移除 ${member.nickname} 吗？`,
      okType: 'danger',
      onOk: async () => {
        const res = await removeGroupMembers({
          group_id: contactInfo.contact_id,
          owner_id: contactInfo.contact_owner_id || userInfo.uuid,
          uuid_list: [member.user_id],
        })
        if (res.code === 200) {
          showToast('已移除', 'success')
          setMembers(prev => prev.filter(m => m.user_id !== member.user_id))
        } else {
          showToast(res.message || '移除失败', 'error')
        }
      },
    })
  }

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()} style={{ width: 420 }}>
        <div className="info-modal-header">
          <h3>群成员管理</h3>
          <button className="info-close-btn" onClick={onClose}>×</button>
        </div>
        <div className="info-modal-body" style={{ maxHeight: 400, overflowY: 'auto' }}>
          {loading && <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>加载中...</div>}
          {!loading && members.length === 0 && (
            <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>暂无成员</div>
          )}
          {members.map(m => {
            const isOwner = contactInfo.contact_owner_id && m.user_id === contactInfo.contact_owner_id
            return (
              <div key={m.user_id} className="contact-user-item" style={{ padding: '8px 0' }}>
                <img src={normalizeAvatarUrl(m.avatar)} alt={m.nickname} className="session-avatar" style={{ width: 36, height: 36 }} />
                <div className="contact-user-info">
                  <span className="contact-user-name">{m.nickname}</span>
                </div>
                {isOwner ? (
                  <span className="tag tag-success" style={{ fontSize: 12 }}>群主</span>
                ) : (
                  <button className="btn-action danger" style={{ fontSize: 12, padding: '4px 12px' }} onClick={() => handleRemove(m)}>移除</button>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
