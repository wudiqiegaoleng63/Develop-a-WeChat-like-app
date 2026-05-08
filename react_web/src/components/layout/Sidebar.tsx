import React from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../../stores/useAuthStore'
import { useUIStore } from '../../stores/useUIStore'

export default function Sidebar() {
  const navigate = useNavigate()
  const userInfo = useAuthStore(state => state.userInfo)
  const logout = useAuthStore(state => state.logout)
  const { sidebarCollapsed, toggleSidebar } = useUIStore()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  if (!userInfo) return null

  return (
    <div className={`sidebar-left ${sidebarCollapsed ? 'collapsed' : ''}`}>
      <div className="sidebar-user">
        <div className="user-avatar-wrapper">
          <img src={userInfo.avatar} alt={userInfo.nickname} className="user-avatar" />
          <div className="status-dot online" />
        </div>
        <div className="user-name">{userInfo.nickname}</div>
        <div className="user-status-text">在线</div>
      </div>

      <div className="sidebar-actions">
        <button className="sidebar-btn">
          <span className="btn-icon">🔍</span>
          <span className="btn-text">搜索好友</span>
        </button>
        <button className="sidebar-btn" onClick={() => navigate('/chat/contactlist')}>
          <span className="btn-icon">➕</span>
          <span className="btn-text">添加好友</span>
        </button>
        <button className="sidebar-btn">
          <span className="btn-icon">👥</span>
          <span className="btn-text">创建群组</span>
        </button>
      </div>

      <div className="sidebar-settings">
        {userInfo.is_admin === 1 && (
          <button className="sidebar-btn" onClick={() => navigate('/manager')}>
            <span className="btn-icon">🛡️</span>
            <span className="btn-text">管理</span>
          </button>
        )}
        <button className="sidebar-btn" onClick={handleLogout}>
          <span className="btn-icon">🚪</span>
          <span className="btn-text">退出</span>
        </button>
      </div>

      <button
        className="collapse-btn"
        onClick={toggleSidebar}
        title={sidebarCollapsed ? '展开侧栏' : '收起侧栏'}
      >
        {sidebarCollapsed ? '›' : '‹'}
      </button>
    </div>
  )
}
