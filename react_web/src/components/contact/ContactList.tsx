import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../../stores/useAuthStore'
import { useContactStore } from '../../stores/useContactStore'
import { enterGroupDirectly, checkGroupAddMode, loadMyGroup } from '../../api/group'
import type { MyGroupItem } from '../../api/group'
import { blackApply } from '../../api/contact'
import { showToast } from '../../utils/toast'
import { normalizeAvatarUrl } from '../../utils/avatar'

type ContactTab = 'search' | 'requests' | 'groups' | 'join'

export default function ContactList() {
  const navigate = useNavigate()
  const userInfo = useAuthStore(state => state.userInfo)
  const {
    searchUsers, friendRequests, myGroups,
    loadingSearch, loadingRequests, loadingGroups,
    fetchSearchUsers, fetchFriendRequests, fetchMyGroups,
    applyFriend, passRequest, refuseRequest,
  } = useContactStore()

  const [activeTab, setActiveTab] = useState<ContactTab>('search')
  const [searchQuery, setSearchQuery] = useState('')
  const [allGroups, setAllGroups] = useState<MyGroupItem[]>([])
  const [groupAddModes, setGroupAddModes] = useState<Record<string, number>>({})
  const [loadingAllGroups, setLoadingAllGroups] = useState(false)

  useEffect(() => {
    if (!userInfo) return
    if (activeTab === 'search') fetchSearchUsers(userInfo.uuid)
    else if (activeTab === 'requests') fetchFriendRequests(userInfo.uuid)
    else if (activeTab === 'groups') fetchMyGroups(userInfo.uuid)
    else if (activeTab === 'join') {
      setLoadingAllGroups(true)
      loadMyGroup(userInfo.uuid).then(async res => {
        if (res.code === 200 && res.data) {
          setAllGroups(res.data)
          // Fetch add_mode for each group
          const modes: Record<string, number> = {}
          await Promise.all(res.data.map(async g => {
            const modeRes = await checkGroupAddMode(g.group_id)
            if (modeRes.code === 200 && modeRes.data !== undefined) modes[g.group_id] = modeRes.data
          }))
          setGroupAddModes(modes)
        }
      }).finally(() => setLoadingAllGroups(false))
    }
  }, [activeTab, userInfo])

  const handleApplyFriend = async (contactId: string) => {
    if (!userInfo) return
    const ok = await applyFriend(userInfo.uuid, contactId)
    showToast(ok ? '申请已发送' : '申请失败', ok ? 'success' : 'error')
  }

  const handlePassRequest = async (contactId: string) => {
    if (!userInfo) return
    const ok = await passRequest(userInfo.uuid, contactId)
    showToast(ok ? '已通过' : '操作失败', ok ? 'success' : 'error')
  }

  const handleRefuseRequest = async (contactId: string) => {
    if (!userInfo) return
    const ok = await refuseRequest(userInfo.uuid, contactId)
    showToast(ok ? '已拒绝' : '操作失败', ok ? 'success' : 'error')
  }

  const handleBlackApply = async (contactId: string) => {
    if (!userInfo) return
    const res = await blackApply({ owner_id: userInfo.uuid, contact_id: contactId })
    if (res.code === 200) {
      showToast('已拉黑', 'success')
      fetchFriendRequests(userInfo.uuid)
    } else {
      showToast(res.message || '操作失败', 'error')
    }
  }

  const handleJoinGroup = async (groupId: string) => {
    if (!userInfo) return
    const modeRes = await checkGroupAddMode(groupId)
    if (modeRes.code !== 200) { showToast('获取群信息失败', 'error'); return }
    if (modeRes.data === 0) {
      const res = await enterGroupDirectly({ owner_id: groupId, contact_id: userInfo.uuid })
      if (res.code === 200) {
        showToast('加入成功', 'success')
        fetchMyGroups(userInfo.uuid)
      } else {
        showToast(res.message || '加入失败', 'error')
      }
    } else {
      const ok = await applyFriend(userInfo.uuid, groupId)
      showToast(ok ? '申请已发送' : '申请失败', ok ? 'success' : 'error')
    }
  }

  const loading = loadingSearch || loadingRequests || loadingGroups || loadingAllGroups

  // getUserList returns { user_id, user_name, avatar }
  const filteredUsers = searchQuery
    ? searchUsers.filter(u => u.user_name.toLowerCase().includes(searchQuery.toLowerCase()))
    : searchUsers

  const tabs: { key: ContactTab; label: string }[] = [
    { key: 'search', label: '搜索好友' },
    { key: 'requests', label: '好友申请' },
    { key: 'groups', label: '我的群组' },
    { key: 'join', label: '加群' },
  ]

  return (
    <div className="contact-list-page">
      <div className="contact-list-header">
        <h2>联系人管理</h2>
        <button className="btn-back-chat" onClick={() => navigate('/chat')}>返回聊天</button>
      </div>

      <div className="session-tabs">
        {tabs.map(t => (
          <button
            key={t.key}
            className={`session-tab ${activeTab === t.key ? 'active' : ''}`}
            onClick={() => setActiveTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="contact-list-content">
        {loading && <div style={{ textAlign: 'center', padding: 20, color: 'var(--text-secondary)' }}>加载中...</div>}

        {activeTab === 'search' && (
          <>
            <div className="search-box">
              <input
                className="search-input"
                placeholder="搜索用户昵称"
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
              />
            </div>
            {filteredUsers.length === 0 && !loading && (
              <div className="contact-empty">暂无用户</div>
            )}
            {filteredUsers.map(u => (
              <div key={u.user_id} className="contact-user-item">
                <img src={normalizeAvatarUrl(u.avatar)} alt={u.user_name} className="session-avatar" />
                <div className="contact-user-info">
                  <span className="contact-user-name">{u.user_name}</span>
                </div>
                <button className="btn-action primary" onClick={() => handleApplyFriend(u.user_id)}>添加好友</button>
              </div>
            ))}
          </>
        )}

        {activeTab === 'requests' && (
          <>
            {friendRequests.length === 0 && !loading && (
              <div className="contact-empty">暂无好友申请</div>
            )}
            {friendRequests.map(r => (
              <div key={r.contact_id} className="contact-user-item">
                <img src={normalizeAvatarUrl(r.contact_avatar)} alt={r.contact_name} className="session-avatar" />
                <div className="contact-user-info">
                  <span className="contact-user-name">{r.contact_name}</span>
                  {r.message && <span className="contact-user-sub">{r.message}</span>}
                </div>
                <div className="contact-actions">
                  <button className="btn-action primary" onClick={() => handlePassRequest(r.contact_id)}>通过</button>
                  <button className="btn-action danger" onClick={() => handleRefuseRequest(r.contact_id)}>拒绝</button>
                  <button className="btn-action" onClick={() => handleBlackApply(r.contact_id)}>拉黑</button>
                </div>
              </div>
            ))}
          </>
        )}

        {activeTab === 'groups' && (
          <>
            {myGroups.length === 0 && !loading && (
              <div className="contact-empty">暂无群组</div>
            )}
            {myGroups.map(g => (
              <div
                key={g.group_id}
                className="contact-user-item"
                style={{ cursor: 'pointer' }}
                onClick={() => navigate(`/chat/${g.group_id}`)}
              >
                <img src={normalizeAvatarUrl(g.avatar)} alt={g.group_name} className="session-avatar" />
                <div className="contact-user-info">
                  <span className="contact-user-name">{g.group_name}</span>
                </div>
                <button className="btn-action primary">进入</button>
              </div>
            ))}
          </>
        )}

        {activeTab === 'join' && (
          <>
            {allGroups.length === 0 && !loading && (
              <div className="contact-empty">暂无可加入的群组</div>
            )}
            {allGroups.map(g => (
              <div key={g.group_id} className="contact-user-item">
                <img src={normalizeAvatarUrl(g.avatar || '')} alt={g.group_name} className="session-avatar" />
                <div className="contact-user-info">
                  <span className="contact-user-name">{g.group_name}</span>
                  <span className="contact-user-sub">{groupAddModes[g.group_id] === 1 ? '需审核' : '直接加入'}</span>
                </div>
                <button className="btn-action primary" onClick={() => handleJoinGroup(g.group_id)}>
                  {groupAddModes[g.group_id] === 1 ? '申请' : '加入'}
                </button>
              </div>
            ))}
          </>
        )}
      </div>
    </div>
  )
}
