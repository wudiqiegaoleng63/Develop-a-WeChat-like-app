import React from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ChatPage from './pages/ChatPage'
import ManagerPage from './pages/ManagerPage'
import { useAuthStore } from './stores/useAuthStore'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const userInfo = useAuthStore(state => state.userInfo)
  if (!userInfo) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const userInfo = useAuthStore(state => state.userInfo)
  if (!userInfo) {
    return <Navigate to="/login" replace />
  }
  if (userInfo.is_admin !== 1) {
    return <Navigate to="/chat" replace />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#07C160',
          borderRadius: 8,
        },
      }}
    >
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Navigate to="/login" replace />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route
            path="/chat"
            element={
              <ProtectedRoute>
                <ChatPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/chat/:id"
            element={
              <ProtectedRoute>
                <ChatPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/chat/contactlist"
            element={
              <ProtectedRoute>
                <ChatPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/manager"
            element={
              <AdminRoute>
                <ManagerPage />
              </AdminRoute>
            }
          />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}
