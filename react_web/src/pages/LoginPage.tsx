import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/useAuthStore'
import { sendEmailCode } from '../api/auth'
import { showToast } from '../utils/toast'

export default function LoginPage() {
  const navigate = useNavigate()
  const login = useAuthStore(state => state.login)
  const emailLogin = useAuthStore(state => state.emailLogin)

  const [tab, setTab] = useState<'account' | 'email'>('account')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [remember, setRemember] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [codeCooldown, setCodeCooldown] = useState(0)
  const [loading, setLoading] = useState(false)

  const handleSendCode = async () => {
    if (!email.trim()) {
      setErrors({ email: '请输入邮箱' })
      return
    }
    const res = await sendEmailCode({ email })
    if (res.code === 200) {
      showToast('验证码已发送', 'success')
      setCodeCooldown(60)
      const timer = setInterval(() => {
        setCodeCooldown(prev => {
          if (prev <= 1) { clearInterval(timer); return 0 }
          return prev - 1
        })
      }, 1000)
    } else {
      showToast(res.message || '验证码发送失败', 'error')
    }
  }

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    const errs: Record<string, string> = {}

    if (tab === 'account') {
      if (!username.trim()) errs.username = '请输入邮箱'
      if (!password.trim()) errs.password = '请输入密码'
      setErrors(errs)
      if (Object.keys(errs).length > 0) return

      setLoading(true)
      const success = await login({ email: username, password })
      setLoading(false)
      if (success) {
        showToast('登录成功', 'success')
        navigate('/chat')
      }
    } else {
      if (!email.trim()) errs.email = '请输入邮箱'
      if (!code.trim()) errs.code = '请输入验证码'
      setErrors(errs)
      if (Object.keys(errs).length > 0) return

      setLoading(true)
      const success = await emailLogin({ email, emailCode: code })
      setLoading(false)
      if (success) {
        showToast('登录成功', 'success')
        navigate('/chat')
      }
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-logo">
          <div className="icon">💬</div>
          <h1>即时通讯</h1>
          <p>随时随地，畅快沟通</p>
        </div>

        <div className="auth-tabs">
          <button
            className={`auth-tab ${tab === 'account' ? 'active' : ''}`}
            onClick={() => { setTab('account'); setErrors({}) }}
          >
            账号密码登录
          </button>
          <button
            className={`auth-tab ${tab === 'email' ? 'active' : ''}`}
            onClick={() => { setTab('email'); setErrors({}) }}
          >
            邮箱验证码登录
          </button>
        </div>

        <form onSubmit={handleLogin}>
          {tab === 'account' ? (
            <>
              <div className="form-group">
                <input
                  className={`form-input ${errors.username ? 'error' : ''}`}
                  placeholder="邮箱"
                  value={username}
                  onChange={e => setUsername(e.target.value)}
                />
                {errors.username && <div className="form-error">{errors.username}</div>}
              </div>
              <div className="form-group">
                <input
                  className={`form-input ${errors.password ? 'error' : ''}`}
                  type="password"
                  placeholder="密码"
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                />
                {errors.password && <div className="form-error">{errors.password}</div>}
              </div>
              <div className="form-options">
                <label className="checkbox-label">
                  <input type="checkbox" checked={remember} onChange={e => setRemember(e.target.checked)} />
                  记住密码
                </label>
                <span className="link">忘记密码？</span>
              </div>
            </>
          ) : (
            <>
              <div className="form-group">
                <input
                  className={`form-input ${errors.email ? 'error' : ''}`}
                  type="email"
                  placeholder="邮箱地址"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                />
                {errors.email && <div className="form-error">{errors.email}</div>}
              </div>
              <div className="form-group">
                <div className="form-row">
                  <input
                    className={`form-input ${errors.code ? 'error' : ''}`}
                    placeholder="验证码"
                    value={code}
                    onChange={e => setCode(e.target.value)}
                  />
                  <button
                    type="button"
                    className="btn-secondary"
                    disabled={codeCooldown > 0}
                    onClick={handleSendCode}
                  >
                    {codeCooldown > 0 ? `${codeCooldown}s` : '获取验证码'}
                  </button>
                </div>
                {errors.code && <div className="form-error">{errors.code}</div>}
              </div>
            </>
          )}
          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>

        <div className="auth-footer">
          还没有账号？<span className="link" onClick={() => navigate('/register')}>立即注册</span>
        </div>
      </div>
    </div>
  )
}
