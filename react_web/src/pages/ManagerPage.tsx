import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Modal } from 'antd'
import { useAuthStore } from '../stores/useAuthStore'
import { getUserInfoList, ableUsers, disableUsers, deleteUsers, setAdmin } from '../api/user'
import { getGroupInfoList, deleteGroups, setGroupsStatus } from '../api/group'
import { showToast } from '../utils/toast'
import type { UserInfo } from '../types/user'
import type { GroupInfo } from '../types/group'

function Checkbox({ checked, onChange }: { checked: boolean; onChange: () => void }) {
  return (
    <span
      onClick={(e) => { e.stopPropagation(); onChange() }}
      style={{
        display: 'inline-block',
        width: 16, height: 16,
        border: `2px solid ${checked ? '#07C160' : '#ccc'}`,
        borderRadius: 3,
        cursor: 'pointer',
        backgroundColor: checked ? '#07C160' : 'transparent',
        position: 'relative',
        verticalAlign: 'middle',
      }}
    >
      {checked && (
        <svg width="10" height="8" viewBox="0 0 10 8" fill="none" style={{ position: 'absolute', top: 1, left: 1 }}>
          <path d="M1 4L3.5 6.5L9 1" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )}
    </span>
  )
}

type MenuKey = 'disable-user' | 'delete-user' | 'set-admin' | 'disable-group' | 'delete-group'

const menuItems = [
  { key: 'disable-user' as MenuKey, icon: '👤', label: '启用/禁用用户', group: '用户管理' },
  { key: 'delete-user' as MenuKey, icon: '🗑️', label: '删除用户', group: '用户管理' },
  { key: 'set-admin' as MenuKey, icon: '🛡️', label: '设置管理员', group: '用户管理' },
  { key: 'disable-group' as MenuKey, icon: '👥', label: '启用/禁用群组', group: '群组管理' },
  { key: 'delete-group' as MenuKey, icon: '🗑️', label: '删除/解散群组', group: '群组管理' },
]

export default function ManagerPage() {
  const navigate = useNavigate()
  const userInfo = useAuthStore(state => state.userInfo)
  const [activeMenu, setActiveMenu] = useState<MenuKey>('disable-user')
  const [users, setUsers] = useState<UserInfo[]>([])
  const [groups, setGroups] = useState<GroupInfo[]>([])
  const [selectedUsers, setSelectedUsers] = useState<string[]>([])
  const [selectedGroups, setSelectedGroups] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [operating, setOperating] = useState(false)

  useEffect(() => {
    const currentUser = useAuthStore.getState().userInfo
    if (!currentUser || currentUser.isAdmin !== 1) {
      navigate('/chat')
      return
    }
    loadData()
  }, [activeMenu])

  const loadData = async () => {
    const currentUser = useAuthStore.getState().userInfo
    if (!currentUser) return
    setLoading(true)
    try {
      if (activeMenu === 'disable-group' || activeMenu === 'delete-group') {
        const res = await getGroupInfoList()
        if (res.code === 200 && res.data) {
          setGroups(res.data)
        }
      } else {
        const res = await getUserInfoList(currentUser.uuid)
        if (res.code === 200 && res.data) {
          setUsers(res.data)
        }
      }
    } catch (e) {
      console.error('Load data error:', e)
    } finally {
      setLoading(false)
    }
  }

  const toggleUser = (uuid: string) => {
    setSelectedUsers(prev => prev.includes(uuid) ? prev.filter(u => u !== uuid) : [...prev, uuid])
  }

  const toggleGroup = (uuid: string) => {
    setSelectedGroups(prev => prev.includes(uuid) ? prev.filter(g => g !== uuid) : [...prev, uuid])
  }

  const toggleAllUsers = () => {
    setSelectedUsers(prev => prev.length === users.length ? [] : users.map(u => u.uuid))
  }

  const toggleAllGroups = () => {
    setSelectedGroups(prev => prev.length === groups.length ? [] : groups.map(g => g.uuid))
  }

  const handleDisableUsers = async () => {
    if (selectedUsers.includes(userInfo!.uuid)) { showToast('不能禁用自己', 'error'); return }
    setOperating(true)
    try {
      const res = await disableUsers({ uuid_list: selectedUsers, is_admin: 0 })
      if (res.code === 200) {
        showToast('禁用成功', 'success')
        setSelectedUsers([])
        loadData()
      } else {
        showToast(res.message || '操作失败', 'error')
      }
    } finally { setOperating(false) }
  }

  const handleEnableUsers = async () => {
    setOperating(true)
    try {
      const res = await ableUsers({ uuid_list: selectedUsers, is_admin: 0 })
      if (res.code === 200) {
        showToast('启用成功', 'success')
        setSelectedUsers([])
        loadData()
      } else {
        showToast(res.message || '操作失败', 'error')
      }
    } finally { setOperating(false) }
  }

  const handleDeleteUsers = () => {
    if (selectedUsers.includes(userInfo!.uuid)) { showToast('不能删除自己', 'error'); return }
    Modal.confirm({
      title: '确认删除用户',
      content: `确定要删除选中的 ${selectedUsers.length} 个用户吗？此操作不可恢复。`,
      okType: 'danger',
      onOk: async () => {
        const res = await deleteUsers({ uuid_list: selectedUsers, is_admin: 0 })
        if (res.code === 200) {
          showToast('删除成功', 'success')
          setSelectedUsers([])
          loadData()
        } else {
          showToast(res.message || '操作失败', 'error')
        }
      },
    })
  }

  const handleSetAdmin = async () => {
    setOperating(true)
    try {
      const res = await setAdmin({ uuid_list: selectedUsers, is_admin: 1 })
      if (res.code === 200) {
        showToast('设置成功', 'success')
        setSelectedUsers([])
        loadData()
      } else {
        showToast(res.message || '操作失败', 'error')
      }
    } finally { setOperating(false) }
  }

  const handleCancelAdmin = async () => {
    if (selectedUsers.includes(userInfo!.uuid)) { showToast('不能取消自己的管理员', 'error'); return }
    setOperating(true)
    try {
      const res = await setAdmin({ uuid_list: selectedUsers, is_admin: 0 })
      if (res.code === 200) {
        showToast('取消成功', 'success')
        setSelectedUsers([])
        loadData()
      } else {
        showToast(res.message || '操作失败', 'error')
      }
    } finally { setOperating(false) }
  }

  const handleDisableGroups = async () => {
    setOperating(true)
    try {
      const res = await setGroupsStatus({ uuid_list: selectedGroups, status: 1 })
      if (res.code === 200) {
        showToast('禁用成功', 'success')
        setSelectedGroups([])
        loadData()
      } else {
        showToast(res.message || '操作失败', 'error')
      }
    } finally { setOperating(false) }
  }

  const handleEnableGroups = async () => {
    setOperating(true)
    try {
      const res = await setGroupsStatus({ uuid_list: selectedGroups, status: 0 })
      if (res.code === 200) {
        showToast('启用成功', 'success')
        setSelectedGroups([])
        loadData()
      } else {
        showToast(res.message || '操作失败', 'error')
      }
    } finally { setOperating(false) }
  }

  const handleDeleteGroups = () => {
    Modal.confirm({
      title: '确认解散群组',
      content: `确定要解散选中的 ${selectedGroups.length} 个群组吗？此操作不可恢复。`,
      okType: 'danger',
      onOk: async () => {
        const res = await deleteGroups(selectedGroups)
        if (res.code === 200) {
          showToast('解散成功', 'success')
          setSelectedGroups([])
          loadData()
        } else {
          showToast(res.message || '操作失败', 'error')
        }
      },
    })
  }

  // Group menu items by group name
  const groups_map: Record<string, typeof menuItems> = {}
  menuItems.forEach(item => {
    if (!groups_map[item.group]) groups_map[item.group] = []
    groups_map[item.group].push(item)
  })

  const allUsersSelected = selectedUsers.length === users.length && users.length > 0
  const allGroupsSelected = selectedGroups.length === groups.length && groups.length > 0

  const renderContent = () => {
    if (loading) {
      return <div style={{ textAlign: 'center', padding: '40px', color: 'var(--text-secondary)' }}>加载中...</div>
    }

    if (activeMenu === 'disable-user') {
      return (
        <>
          <h2>👤 启用/禁用用户</h2>
          <table className="admin-table">
            <thead>
              <tr>
                <th><Checkbox checked={allUsersSelected} onChange={toggleAllUsers} /></th>
                <th>UUID</th><th>昵称</th><th>邮箱</th><th>管理员</th><th>状态</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.uuid}>
                  <td><Checkbox checked={selectedUsers.includes(u.uuid)} onChange={() => toggleUser(u.uuid)} /></td>
                  <td style={{fontFamily:'monospace',fontSize:13}}>{u.uuid}</td>
                  <td>{u.nickname}</td>
                  <td>{u.email}</td>
                  <td><span className={`tag ${u.isAdmin ? 'tag-success' : 'tag-default'}`}>{u.isAdmin ? '是' : '否'}</span></td>
                  <td><span className={`tag ${u.status === 0 ? 'tag-success' : 'tag-danger'}`}>{u.status === 0 ? '正常' : '禁用'}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="admin-actions">
            <button className="btn-action danger" onClick={handleDisableUsers} disabled={selectedUsers.length===0||operating}>禁用</button>
            <button className="btn-action primary" onClick={handleEnableUsers} disabled={selectedUsers.length===0||operating}>启用</button>
          </div>
        </>
      )
    }

    if (activeMenu === 'delete-user') {
      return (
        <>
          <h2>🗑️ 删除用户</h2>
          <table className="admin-table">
            <thead>
              <tr>
                <th><Checkbox checked={allUsersSelected} onChange={toggleAllUsers} /></th>
                <th>UUID</th><th>昵称</th><th>邮箱</th><th>管理员</th><th>状态</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.uuid}>
                  <td><Checkbox checked={selectedUsers.includes(u.uuid)} onChange={() => toggleUser(u.uuid)} /></td>
                  <td style={{fontFamily:'monospace',fontSize:13}}>{u.uuid}</td>
                  <td>{u.nickname}</td>
                  <td>{u.email}</td>
                  <td><span className={`tag ${u.isAdmin ? 'tag-success' : 'tag-default'}`}>{u.isAdmin ? '是' : '否'}</span></td>
                  <td><span className={`tag ${u.status === 0 ? 'tag-success' : 'tag-danger'}`}>{u.status === 0 ? '正常' : '禁用'}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="admin-actions">
            <button className="btn-action danger" onClick={handleDeleteUsers} disabled={selectedUsers.length===0||operating}>删除选中</button>
          </div>
        </>
      )
    }

    if (activeMenu === 'set-admin') {
      return (
        <>
          <h2>🛡️ 设置管理员</h2>
          <table className="admin-table">
            <thead>
              <tr>
                <th><Checkbox checked={allUsersSelected} onChange={toggleAllUsers} /></th>
                <th>UUID</th><th>昵称</th><th>邮箱</th><th>管理员</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.uuid}>
                  <td><Checkbox checked={selectedUsers.includes(u.uuid)} onChange={() => toggleUser(u.uuid)} /></td>
                  <td style={{fontFamily:'monospace',fontSize:13}}>{u.uuid}</td>
                  <td>{u.nickname}</td>
                  <td>{u.email}</td>
                  <td><span className={`tag ${u.isAdmin ? 'tag-success' : 'tag-default'}`}>{u.isAdmin ? '是' : '否'}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="admin-actions">
            <button className="btn-action primary" onClick={handleSetAdmin} disabled={selectedUsers.length===0||operating}>设为管理员</button>
            <button className="btn-action" onClick={handleCancelAdmin} disabled={selectedUsers.length===0||operating}>取消管理员</button>
          </div>
        </>
      )
    }

    if (activeMenu === 'disable-group') {
      return (
        <>
          <h2>👥 启用/禁用群组</h2>
          <table className="admin-table">
            <thead>
              <tr>
                <th><Checkbox checked={allGroupsSelected} onChange={toggleAllGroups} /></th>
                <th>UUID</th><th>群名称</th><th>群主ID</th><th>成员数</th><th>状态</th>
              </tr>
            </thead>
            <tbody>
              {groups.map(g => (
                <tr key={g.uuid}>
                  <td><Checkbox checked={selectedGroups.includes(g.uuid)} onChange={() => toggleGroup(g.uuid)} /></td>
                  <td style={{fontFamily:'monospace',fontSize:13}}>{g.uuid}</td>
                  <td>{g.name}</td>
                  <td style={{fontFamily:'monospace',fontSize:13}}>{g.owner_id}</td>
                  <td>{g.member_cnt}</td>
                  <td><span className={`tag ${g.status === 0 ? 'tag-success' : 'tag-danger'}`}>{g.status === 0 ? '正常' : '禁用'}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="admin-actions">
            <button className="btn-action danger" onClick={handleDisableGroups} disabled={selectedGroups.length===0||operating}>禁用</button>
            <button className="btn-action primary" onClick={handleEnableGroups} disabled={selectedGroups.length===0||operating}>启用</button>
          </div>
        </>
      )
    }

    if (activeMenu === 'delete-group') {
      return (
        <>
          <h2>🗑️ 删除/解散群组</h2>
          <table className="admin-table">
            <thead>
              <tr>
                <th><Checkbox checked={allGroupsSelected} onChange={toggleAllGroups} /></th>
                <th>UUID</th><th>群名称</th><th>群主ID</th><th>成员数</th><th>状态</th>
              </tr>
            </thead>
            <tbody>
              {groups.map(g => (
                <tr key={g.uuid}>
                  <td><Checkbox checked={selectedGroups.includes(g.uuid)} onChange={() => toggleGroup(g.uuid)} /></td>
                  <td style={{fontFamily:'monospace',fontSize:13}}>{g.uuid}</td>
                  <td>{g.name}</td>
                  <td style={{fontFamily:'monospace',fontSize:13}}>{g.owner_id}</td>
                  <td>{g.member_cnt}</td>
                  <td><span className={`tag ${g.status === 0 ? 'tag-success' : 'tag-danger'}`}>{g.status === 0 ? '正常' : '禁用'}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="admin-actions">
            <button className="btn-action danger" onClick={handleDeleteGroups} disabled={selectedGroups.length===0||operating}>解散群组</button>
          </div>
        </>
      )
    }
  }

  return (
    <div className="admin-page">
      <div className="admin-header">
        <div className="admin-header-left">
          <div className="admin-header-icon">🛡️</div>
          <div className="admin-header-title">Admin</div>
        </div>
        <button className="btn-back-chat" onClick={() => navigate('/chat')}>返回聊天</button>
      </div>
      <div className="admin-body">
        <div className="admin-sidebar">
          {Object.entries(groups_map).map(([groupName, items]) => (
            <div key={groupName} className="admin-menu-group">
              <div className="admin-menu-label">{groupName}</div>
              {items.map(item => (
                <button
                  key={item.key}
                  className={`admin-menu-item ${activeMenu === item.key ? 'active' : ''}`}
                  onClick={() => { setActiveMenu(item.key); setSelectedUsers([]); setSelectedGroups([]) }}
                >
                  <span className="menu-icon">{item.icon}</span>
                  {item.label}
                </button>
              ))}
            </div>
          ))}
        </div>
        <div className="admin-content">
          {renderContent()}
        </div>
      </div>
    </div>
  )
}
