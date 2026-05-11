import React, { useState, useEffect } from 'react'
import { Modal } from 'antd'
import { useChatStore } from '../../stores/useChatStore'
import { useAuthStore } from '../../stores/useAuthStore'
import { deleteSession } from '../../api/session'
import { deleteContact, blackContact, cancelBlackContact } from '../../api/contact'
import { getCurContactListInChatRoom } from '../../api/chatroom'
import { getGroupMemberList } from '../../api/group'
import { leaveGroup, dismissGroup } from '../../api/group'
import { showToast } from '../../utils/toast'
import { normalizeAvatarUrl } from '../../utils/avatar'
import AVCallModal from './AVCallModal'
import ContactInfoModal from './ContactInfoModal'
import EditGroupInfoModal from '../group/EditGroupInfoModal'
import RemoveGroupMembersModal from '../group/RemoveGroupMembersModal'
import GroupJoinRequestsModal from '../group/GroupJoinRequestsModal'

export default function ChatHeader() {
  const contactInfo = useChatStore(state => state.contactInfo)
  const activeSessionId = useChatStore(state => state.activeSessionId)
  const clearChat = useChatStore(state => state.clearChat)
  const userInfo = useAuthStore(state => state.userInfo)

  const [dropdownOpen, setDropdownOpen] = useState(false)
  const [avModalVisible, setAvModalVisible] = useState(false)
  const [infoModalVisible, setInfoModalVisible] = useState(false)
  const [editGroupVisible, setEditGroupVisible] = useState(false)
  const [removeMembersVisible, setRemoveMembersVisible] = useState(false)
  const [joinRequestsVisible, setJoinRequestsVisible] = useState(false)
  const [onlineMembersVisible, setOnlineMembersVisible] = useState(false)
  const [groupMembersVisible, setGroupMembersVisible] = useState(false)

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
  const isOwner = isGroup && !!contactInfo.contact_owner_id && contactInfo.contact_owner_id === userInfo.uuid

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

  const handleCancelBlackContact = () => {
    Modal.confirm({
      title: '取消拉黑',
      content: `确定要取消拉黑 ${contactInfo.contact_name} 吗？`,
      onOk: async () => {
        const res = await cancelBlackContact({ owner_id: userInfo.uuid, contact_id: contactInfo.contact_id })
        if (res.code === 200) {
          showToast('已取消拉黑', 'success')
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
        { icon: '📋', label: '群成员', onClick: () => setGroupMembersVisible(true) },
        { icon: '🟢', label: '在线成员', onClick: () => setOnlineMembersVisible(true) },
        ...(isOwner
          ? [
              { icon: '✏️', label: '修改群聊信息', onClick: () => setEditGroupVisible(true) },
              { icon: '🚫', label: '移除群组人员', onClick: () => setRemoveMembersVisible(true) },
              { icon: '📋', label: '加群申请', onClick: () => setJoinRequestsVisible(true) },
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
        { icon: '✅', label: '取消拉黑', onClick: handleCancelBlackContact },
      ]

  return (
    <>
      <div className="chat-header">
        <div className="chat-header-left">
          <img src={contactInfo.contact_avatar} alt={contactInfo.contact_name} className="chat-header-avatar" />
          <div className="chat-header-info">
            <h3>{contactInfo.contact_name}</h3>
            <div className="status-text">
              {isGroup ? `${contactInfo.contact_member_cnt ?? 0}人` : (contactInfo.contact_signature || '')}
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

      <AVCallModal visible={avModalVisible} onClose={() => setAvModalVisible(false)} />
      <ContactInfoModal visible={infoModalVisible} onClose={() => setInfoModalVisible(false)} contactInfo={contactInfo} isGroup={isGroup} />
      {isGroup && (
        <>
          <GroupMembersModal visible={groupMembersVisible} onClose={() => setGroupMembersVisible(false)} groupId={contactInfo.contact_id} ownerId={contactInfo.contact_owner_id || ''} currentUserId={userInfo.uuid} />
          <OnlineMembersModal visible={onlineMembersVisible} onClose={() => setOnlineMembersVisible(false)} groupId={contactInfo.contact_id} userId={userInfo.uuid} />
          {isOwner && (
            <>
              <EditGroupInfoModal visible={editGroupVisible} onClose={() => setEditGroupVisible(false)} />
              <RemoveGroupMembersModal visible={removeMembersVisible} onClose={() => setRemoveMembersVisible(false)} />
              <GroupJoinRequestsModal visible={joinRequestsVisible} onClose={() => setJoinRequestsVisible(false)} />
            </>
          )}
        </>
      )}
    </>
  )
}

function OnlineMembersModal({ visible, onClose, groupId, userId }: { visible: boolean; onClose: () => void; groupId: string; userId: string }) {
  const [members, setMembers] = useState<string[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (visible) {
      setLoading(true)
      getCurContactListInChatRoom({ owner_id: userId, contact_id: groupId }).then(res => {
        if (res.code === 200 && res.data) {
          setMembers(res.data.map(m => m.contact_id))
        }
      }).finally(() => setLoading(false))
    }
  }, [visible, groupId, userId])

  if (!visible) return null

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()} style={{ width: 320 }}>
        <div className="info-modal-header">
          <h3>在线成员</h3>
          <button className="info-close-btn" onClick={onClose}>×</button>
        </div>
        <div className="info-modal-body" style={{ maxHeight: 300, overflowY: 'auto' }}>
          {loading && <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>加载中...</div>}
          {!loading && members.length === 0 && (
            <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>暂无在线成员</div>
          )}
          {members.map(id => (
            <div key={id} className="contact-user-item" style={{ padding: '6px 0' }}>
              <span className="status-indicator online" />
              <span style={{ marginLeft: 8, fontFamily: 'monospace', fontSize: 13 }}>{id}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function GroupMembersModal({ visible, onClose, groupId, ownerId, currentUserId }: { visible: boolean; onClose: () => void; groupId: string; ownerId: string; currentUserId: string }) {
  const [members, setMembers] = useState<{ user_id: string; nickname: string; avatar: string }[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (visible) {
      setLoading(true)
      getGroupMemberList(groupId).then(res => {
        if (res.code === 200 && res.data) setMembers(res.data)
      }).finally(() => setLoading(false))
    }
  }, [visible, groupId])

  if (!visible) return null

  const owner = members.find(m => m.user_id === ownerId)

  return (
    <div className="info-modal-overlay" onClick={onClose}>
      <div className="info-modal" onClick={e => e.stopPropagation()} style={{ width: 380 }}>
        <div className="info-modal-header">
          <h3>群成员 ({members.length})</h3>
          <button className="info-close-btn" onClick={onClose}>×</button>
        </div>
        <div className="info-modal-body" style={{ maxHeight: 400, overflowY: 'auto' }}>
          {loading && <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>加载中...</div>}
          {!loading && members.length === 0 && (
            <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>暂无成员</div>
          )}
          {owner && (
            <div className="contact-user-item" style={{ padding: '8px 0' }}>
              <img src={normalizeAvatarUrl(owner.avatar)} alt={owner.nickname} className="session-avatar" style={{ width: 36, height: 36 }} />
              <div className="contact-user-info">
                <span className="contact-user-name">{owner.nickname}</span>
                <span style={{ fontSize: 11, color: 'var(--primary)', marginLeft: 6 }}>群主</span>
              </div>
            </div>
          )}
          {members.filter(m => m.user_id !== ownerId).map(m => (
            <div key={m.user_id} className="contact-user-item" style={{ padding: '8px 0' }}>
              <img src={normalizeAvatarUrl(m.avatar)} alt={m.nickname} className="session-avatar" style={{ width: 36, height: 36 }} />
              <div className="contact-user-info">
                <span className="contact-user-name">{m.nickname}</span>
                {m.user_id === currentUserId && <span style={{ fontSize: 11, color: 'var(--text-secondary)', marginLeft: 6 }}>我</span>}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
