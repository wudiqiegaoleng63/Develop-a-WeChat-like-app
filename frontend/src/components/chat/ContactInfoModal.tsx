import React, { useState, useEffect } from 'react'
import type { ContactInfo } from '../../types/user'
import { getUserInfo } from '../../api/user'
import { normalizeAvatarUrl } from '../../utils/avatar'

interface ContactInfoModalProps {
  visible: boolean
  onClose: () => void
  contactInfo: ContactInfo
  isGroup: boolean
}

export default function ContactInfoModal({ visible, onClose, contactInfo, isGroup }: ContactInfoModalProps) {
  const [detailedInfo, setDetailedInfo] = useState<ContactInfo | null>(null)

  // Fetch fresh user info via getUserInfo for personal contacts
  useEffect(() => {
    if (visible && !isGroup && contactInfo.contact_id) {
      getUserInfo(contactInfo.contact_id).then(res => {
        if (res.code === 200 && res.data) {
          setDetailedInfo({
            contact_id: res.data.uuid,
            contact_name: res.data.nickname,
            contact_avatar: normalizeAvatarUrl(res.data.avatar || ''),
            contact_phone: res.data.telephone,
            contact_email: res.data.email,
            contact_gender: res.data.gender,
            contact_signature: res.data.signature,
            contact_birthday: res.data.birthday,
          })
        }
      })
    } else {
      setDetailedInfo(null)
    }
  }, [visible, isGroup, contactInfo.contact_id])

  if (!visible) return null

  const info = detailedInfo || contactInfo

  if (isGroup) {
    return (
      <div className="info-modal-overlay" onClick={onClose}>
        <div className="info-modal" onClick={e => e.stopPropagation()}>
          <div className="info-modal-header">
            <h3>群聊信息</h3>
            <button className="info-close-btn" onClick={onClose}>✕</button>
          </div>
          <div className="info-modal-body">
            <div className="info-avatar-section">
              <img src={contactInfo.contact_avatar} className="info-avatar" alt={contactInfo.contact_name} />
              <div className="info-name">{contactInfo.contact_name}</div>
              <div className="info-id">ID: {contactInfo.contact_id}</div>
            </div>
            <table className="info-table">
              <tbody>
                <tr><td className="info-label">群主ID</td><td className="info-value" style={{fontFamily:'monospace',fontSize:13}}>{contactInfo.contact_owner_id || '未知'}</td></tr>
                <tr><td className="info-label">成员数量</td><td className="info-value">{contactInfo.contact_member_cnt ?? 0} 人</td></tr>
                <tr><td className="info-label">加群方式</td><td className="info-value">{contactInfo.contact_add_mode === 0 ? '直接加入' : contactInfo.contact_add_mode === 1 ? '需审核' : '未知'}</td></tr>
                <tr>
                  <td className="info-label">群公告</td>
                  <td className="info-value"><div className="info-textarea">{contactInfo.contact_notice || '暂无公告'}</div></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()}>
        <div className="info-modal-header">
          <h3>个人信息</h3>
          <button className="info-close-btn" onClick={onClose}>✕</button>
        </div>
        <div className="info-modal-body">
          <div className="info-avatar-section">
            <img src={info.contact_avatar} className="info-avatar" alt={info.contact_name} />
            <div className="info-name">{info.contact_name}</div>
            <div className="info-id">ID: {info.contact_id}</div>
          </div>
          <table className="info-table">
            <tbody>
              <tr><td className="info-label">性别</td><td className="info-value">{info.contact_gender === 0 ? '男' : info.contact_gender === 1 ? '女' : '未设置'}</td></tr>
              <tr><td className="info-label">手机</td><td className="info-value">{info.contact_phone || '未设置'}</td></tr>
              <tr><td className="info-label">邮箱</td><td className="info-value">{info.contact_email || '未设置'}</td></tr>
              <tr><td className="info-label">生日</td><td className="info-value">{info.contact_birthday || '未设置'}</td></tr>
              <tr>
                <td className="info-label">个性签名</td>
                <td className="info-value"><div className="info-textarea">{info.contact_signature || '暂无签名'}</div></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
