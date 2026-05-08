import React, { useState, useEffect } from 'react'
import { Modal } from 'antd'
import { useChatStore } from '../../stores/useChatStore'
import { useAuthStore } from '../../stores/useAuthStore'
import { deleteSession } from '../../api/session'
import { deleteContact, blackContact } from '../../api/contact'
import { leaveGroup, dismissGroup } from '../../api/group'
import { showToast } from '../../utils/toast'
import AVCallModal from './AVCallModal'
import ContactInfoModal from './ContactInfoModal'

export default function ChatHeader() {
  const contactInfo = useChatStore(state => state.contactInfo)
  const activeSessionId = useChatStore(state => state.activeSessionId)
  const clearChat = useChatStore(state => state.clearChat)
  const userInfo = useAuthStore(state => state.userInfo)

  const [dropdownOpen, setDropdownOpen] = useState(false)
  const [avModalVisible, setAvModalVisible] = useState(false)
  const [infoModalVisible, setInfoModalVisible] = useState(false)

  // Close dropdown on outside click
  useEffect(() => {
    if (!dropdownOpen) return
    const handler = (e: MouseEvent) => {
      if (!(e.target as HTMLElement).closest('.dropdown-menu-wrapper')) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [dropdownOpen])

  if (!contactInfo || !userInfo) return null

  const isGroup = contactInfo.contact_id.startsWith('G')
  const isOwner = isGroup && contactInfo.contact_owner_id === userInfo.uuid

  const handleDeleteSession = () => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除该会话吗？',
      onOk: async () => {
        if (!activeSessionId) return
        const res = await deleteSession({ owner_id: userInfo.uuid, session_id: activeSessionId })
        if (res.code === 200) {
          showToast('会话已删除', 'success')
          clearChat()
        } else {
          showToast(res.message || '删除失败', 'error')
        }
      },
    })
  }

  const handleDeleteContact = () => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除联系人 ${contactInfo.contact_name} 吗？`,
      onOk: async () => {
        const res = await deleteContact({ owner_id: userInfo.uuid, contact_id: contactInfo.contact_id })
        if (res.code === 200) {
          showToast('联系人已删除', 'success')
          clearChat()
        } else {
          showToast(res.message || '删除失败', 'error')
        }
      },
    })
  }

  const handleBlackContact = () => {
    Modal.confirm({
      title: '确认拉黑',
      content: `确定要拉黑联系人 ${contactInfo.contact_name} 吗？`,
      onOk: async () => {
        const res = await blackContact({ owner_id: userInfo.uuid, contact_id: contactInfo.contact_id })
        if (res.code === 200) {
          showToast('已拉黑', 'success')
          clearChat()
        } else {
          showToast(res.message || '操作失败', 'error')
        }
      },
    })
  }

  const handleLeaveGroup = () => {
    Modal.confirm({
      title: '确认退出',
      content: '确定要退出该群聊吗？',
      onOk: async () => {
        const res = await leaveGroup({ user_id: userInfo.uuid, group_id: contactInfo.contact_id })
        if (res.code === 200) {
          showToast('已退出群聊', 'success')
          clearChat()
        } else {
          showToast(res.message || '操作失败', 'error')
        }
      },
    })
  }

  const handleDismissGroup = () => {
    Modal.confirm({
      title: '确认解散',
      content: '确定要解散该群聊吗？此操作不可恢复！',
      okType: 'danger',
      onOk: async () => {
        const res = await dismissGroup({ owner_id: userInfo.uuid, group_id: contactInfo.contact_id })
        if (res.code === 200) {
          showToast('群聊已解散', 'success')
          clearChat()
        } else {
          showToast(res.message || '操作失败', 'error')
        }
      },
    })
  }

  // Build dropdown menu items
  const menuItems = isGroup
    ? [
        { icon: '👥', label: '群聊信息', onClick: () => setInfoModalVisible(true) },
        ...(isOwner
          ? [
              { icon: '✏️', label: '修改群聊信息', onClick: () => showToast('修改群聊信息功能开发中', 'info') },
              { icon: '🚫', label: '移除群组人员', onClick: () => showToast('移除群组人员功能开发中', 'info') },
              { icon: '📋', label: '加群申请', onClick: () => showToast('加群申请功能开发中', 'info') },
            ]
          : []),
        { icon: '🗑️', label: '删除该会话', onClick: handleDeleteSession },
        ...(isOwner
          ? [{ icon: '💥', label: '解散群聊', onClick: handleDismissGroup, danger: true }]
          : [{ icon: '🚪', label: '退出群聊', onClick: handleLeaveGroup, danger: true }]),
      ]
    : [
        { icon: '👤', label: '个人信息', onClick: () => setInfoModalVisible(true) },
        { icon: '🗑️', label: '删除该会话', onClick: handleDeleteSession },
        { icon: '❌', label: '删除联系人', onClick: handleDeleteContact, danger: true },
        { icon: '🚫', label: '拉黑联系人', onClick: handleBlackContact, danger: true },
      ]

  return (
    <>
      <div className="chat-header">
        <div className="chat-header-left">
          <img src={contactInfo.contact_avatar} alt={contactInfo.contact_name} className="chat-header-avatar" />
          <div className="chat-header-info">
            <h3>{contactInfo.contact_name}</h3>
            <div className="status-text">
              {isGroup ? `${contactInfo.contact_member_cnt || 0}人` : (contactInfo.contact_signature || '')}
            </div>
          </div>
        </div>
        <div className="chat-header-actions">
          <button className="chat-header-btn" title="音视频通话" onClick={() => setAvModalVisible(true)}>📞</button>
          <div className="dropdown-menu-wrapper">
            <button className="chat-header-btn" title="更多" onClick={() => setDropdownOpen(!dropdownOpen)}>⋯</button>
            {dropdownOpen && (
              <div className="dropdown-menu">
                {menuItems.map((item, i) => (
                  <button
                    key={i}
                    className={`dropdown-item${item.danger ? ' danger' : ''}`}
                    onClick={() => { item.onClick(); setDropdownOpen(false) }}
                  >
                    <span className="item-icon">{item.icon}</span>
                    {item.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <AVCallModal
        visible={avModalVisible}
        onClose={() => setAvModalVisible(false)}
      />

      <ContactInfoModal
        visible={infoModalVisible}
        onClose={() => setInfoModalVisible(false)}
        contactInfo={contactInfo}
        isGroup={isGroup}
      />
    </>
  )
}
