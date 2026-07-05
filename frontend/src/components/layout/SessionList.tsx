import React, { useEffect, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAuthStore } from '../../stores/useAuthStore'
import { useChatStore } from '../../stores/useChatStore'
import { useUIStore } from '../../stores/useUIStore'
import { formatTime } from '../../utils/format'

export default function SessionList() {
  const navigate = useNavigate()
  const { id: activeId } = useParams()
  const userInfo = useAuthStore(state => state.userInfo)
  const userSessionList = useChatStore(state => state.userSessionList)
  const groupSessionList = useChatStore(state => state.groupSessionList)
  const fetchUserSessionList = useChatStore(state => state.fetchUserSessionList)
  const fetchGroupSessionList = useChatStore(state => state.fetchGroupSessionList)
  const { activeTab, setActiveTab, searchQuery, setSearchQuery } = useUIStore()

  useEffect(() => {
    if (userInfo) {
      fetchUserSessionList(userInfo.uuid)
      fetchGroupSessionList(userInfo.uuid)
    }
  }, [userInfo, fetchUserSessionList, fetchGroupSessionList])

  const allSessions = useMemo(() => {
    const userSessions = userSessionList.map(s => ({
      id: s.user_id,
      name: s.user_name,
      avatar: s.avatar,
      status: 0,
      lastMessage: '',
      lastTime: '',
      unread: 0,
      isGroup: false,
    }))
    const groupSessions = groupSessionList.map(s => ({
      id: s.group_id,
      name: s.group_name,
      avatar: s.avatar,
      status: 0,
      lastMessage: '',
      lastTime: '',
      unread: 0,
      isGroup: true,
    }))
    return [...userSessions, ...groupSessions]
  }, [userSessionList, groupSessionList])

  const filteredSessions = useMemo(() => {
    let sessions = allSessions
    if (activeTab === 'contacts') sessions = sessions.filter(s => !s.isGroup)
    if (activeTab === 'groups') sessions = sessions.filter(s => s.isGroup)
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      sessions = sessions.filter(s => s.name.toLowerCase().includes(q))
    }
    return sessions
  }, [allSessions, activeTab, searchQuery])

  return (
    <div className="sidebar-middle">
      <div className="search-box">
        <input
          className="search-input"
          placeholder="搜索联系人/群组"
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
        />
      </div>

      <div className="session-tabs">
        <button
          className={`session-tab ${activeTab === 'all' ? 'active' : ''}`}
          onClick={() => setActiveTab('all')}
        >
          全部
        </button>
        <button
          className={`session-tab ${activeTab === 'contacts' ? 'active' : ''}`}
          onClick={() => setActiveTab('contacts')}
        >
          联系人
        </button>
        <button
          className={`session-tab ${activeTab === 'groups' ? 'active' : ''}`}
          onClick={() => setActiveTab('groups')}
        >
          群组
        </button>
      </div>

      <div className="session-list">
        {filteredSessions.map(session => (
          <div
            key={session.id}
            className={`session-item ${activeId === session.id ? 'active' : ''}`}
            onClick={() => navigate(`/chat/${session.id}`)}
          >
            <img src={session.avatar} alt={session.name} className="session-avatar" />
            <div className="session-info">
              <div className="session-name">
                {session.name}
                {!session.isGroup && (
                  <span className={`status-indicator ${session.status === 0 ? 'online' : 'offline'}`} />
                )}
              </div>
              <div className="session-preview">{session.lastMessage}</div>
            </div>
            <div className="session-meta">
              <span className="session-time">{formatTime(session.lastTime)}</span>
              {session.unread > 0 && (
                <span className="unread-badge">
                  {session.unread > 99 ? '99+' : session.unread}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
