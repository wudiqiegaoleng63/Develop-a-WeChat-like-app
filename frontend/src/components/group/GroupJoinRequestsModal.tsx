import React, { useState, useEffect } from 'react'
import { Modal } from 'antd'
import { useAuthStore } from '../../stores/useAuthStore'
import { useChatStore } from '../../stores/useChatStore'
import { getAddGroupList } from '../../api/contact'
import { passContactApply, refuseContactApply, blackApply } from '../../api/contact'
import { showToast } from '../../utils/toast'
import { normalizeAvatarUrl } from '../../utils/avatar'
import type { ContactRequest } from '../../types/user'

interface Props {
  visible: boolean
  onClose: () => void
}

export default function GroupJoinRequestsModal({ visible, onClose }: Props) {
  const userInfo = useAuthStore(state => state.userInfo)
  const contactInfo = useChatStore(state => state.contactInfo)
  const [requests, setRequests] = useState<ContactRequest[]>([])
  const [loading, setLoading] = useState(false)

  const loadRequests = () => {
    if (!contactInfo) return
    setLoading(true)
    getAddGroupList(contactInfo.contact_id).then(res => {
      if (res.code === 200 && res.data) setRequests(res.data)
    }).finally(() => setLoading(false))
  }

  useEffect(() => {
    if (visible) loadRequests()
  }, [visible])

  if (!visible || !contactInfo || !userInfo) return null

  const handlePass = async (contactId: string) => {
    const res = await passContactApply({ owner_id: contactInfo.contact_id, contact_id: contactId })
    if (res.code === 200) {
      showToast('已通过', 'success')
      loadRequests()
    } else {
      showToast(res.message || '操作失败', 'error')
    }
  }

  const handleRefuse = async (contactId: string) => {
    const res = await refuseContactApply({ owner_id: contactInfo.contact_id, contact_id: contactId })
    if (res.code === 200) {
      showToast('已拒绝', 'success')
      loadRequests()
    } else {
      showToast(res.message || '操作失败', 'error')
    }
  }

  const handleBlack = (contactId: string, name: string) => {
    Modal.confirm({
      title: '确认拉黑',
      content: `确定要拉黑申请人 ${name} 吗？`,
      okType: 'danger',
      onOk: async () => {
        const res = await blackApply({ owner_id: contactInfo.contact_id, contact_id: contactId })
        if (res.code === 200) {
          showToast('已拉黑', 'success')
          loadRequests()
        } else {
          showToast(res.message || '操作失败', 'error')
        }
      },
    })
  }

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()} style={{ width: 420 }}>
        <div className="info-modal-header">
          <h3>加群申请</h3>
          <button className="info-close-btn" onClick={onClose}>×</button>
        </div>
        <div className="info-modal-body" style={{ maxHeight: 400, overflowY: 'auto' }}>
          {loading && <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>加载中...</div>}
          {!loading && requests.length === 0 && (
            <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>暂无加群申请</div>
          )}
          {requests.map(r => (
            <div key={r.contact_id} className="contact-user-item" style={{ padding: '8px 0' }}>
              <img src={normalizeAvatarUrl(r.contact_avatar)} alt={r.contact_name} className="session-avatar" style={{ width: 36, height: 36 }} />
              <div className="contact-user-info">
                <span className="contact-user-name">{r.contact_name}</span>
                {r.message && <span className="contact-user-sub">{r.message}</span>}
              </div>
              <div className="contact-actions">
                <button className="btn-action primary" style={{ fontSize: 12, padding: '4px 10px' }} onClick={() => handlePass(r.contact_id)}>通过</button>
                <button className="btn-action danger" style={{ fontSize: 12, padding: '4px 10px' }} onClick={() => handleRefuse(r.contact_id)}>拒绝</button>
                <button className="btn-action" style={{ fontSize: 12, padding: '4px 10px' }} onClick={() => handleBlack(r.contact_id, r.contact_name)}>拉黑</button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
