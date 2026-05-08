import React from 'react'
import type { ContactInfo } from '../../types/user'

interface ContactInfoModalProps {
  visible: boolean
  onClose: () => void
  contactInfo: ContactInfo
  isGroup: boolean
}

export default function ContactInfoModal({ visible, onClose, contactInfo, isGroup }: ContactInfoModalProps) {
  if (!visible) return null

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
                <tr><td className="info-label">群主ID</td><td className="info-value" style={{fontFamily:'monospace',fontSize:13}}>{contactInfo.contact_owner_id}</td></tr>
                <tr><td className="info-label">成员数量</td><td className="info-value">{contactInfo.contact_member_cnt} 人</td></tr>
                <tr><td className="info-label">加群方式</td><td className="info-value">{contactInfo.contact_add_mode === 0 ? '直接加入' : '需审核'}</td></tr>
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
            <img src={contactInfo.contact_avatar} className="info-avatar" alt={contactInfo.contact_name} />
            <div className="info-name">{contactInfo.contact_name}</div>
            <div className="info-id">ID: {contactInfo.contact_id}</div>
          </div>
          <table className="info-table">
            <tbody>
              <tr><td className="info-label">性别</td><td className="info-value">{contactInfo.contact_gender === 0 ? '男' : '女'}</td></tr>
              <tr><td className="info-label">手机</td><td className="info-value">{contactInfo.contact_phone || '未设置'}</td></tr>
              <tr><td className="info-label">邮箱</td><td className="info-value">{contactInfo.contact_email || '未设置'}</td></tr>
              <tr><td className="info-label">生日</td><td className="info-value">{contactInfo.contact_birthday || '未设置'}</td></tr>
              <tr>
                <td className="info-label">个性签名</td>
                <td className="info-value"><div className="info-textarea">{contactInfo.contact_signature || '暂无签名'}</div></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
