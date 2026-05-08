import React, { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/useAuthStore'
import { sendEmailCode } from '../api/auth'
import { showToast } from '../utils/toast'

export default function RegisterPage() {
  const navigate = useNavigate()
  const register = useAuthStore(state => state.register)

  const [form, setForm] = useState({ nickname: '', password: '', confirm: '', email: '', emailCode: '' })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [avatarUrl, setAvatarUrl] = useState<string | null>(null)
  const [codeCooldown, setCodeCooldown] = useState(0)
  const [loading, setLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const avatarUrlRef = useRef<string | null>(null)

  useEffect(() => {
    return () => {
      if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current)
    }
  }, [])

  const update = (key: string, value: string) => setForm(prev => ({ ...prev, [key]: value }))

  const handleAvatarClick = () => fileInputRef.current?.click()

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current)
      const url = URL.createObjectURL(file)
      avatarUrlRef.current = url
      setAvatarUrl(url)
    }
  }

  const handleSendCode = async () => {
    if (!form.email.trim()) {
      setErrors({ email: '请输入邮箱' })
      return
    }
    const res = await sendEmailCode({ email: form.email })
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

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    const errs: Record<string, string> = {}
    if (!form.nickname.trim()) errs.nickname = '请输入用户名'
    if (!form.password.trim()) errs.password = '请输入密码'
    else if (form.password.length < 6) errs.password = '密码至少6位'
    if (form.password !== form.confirm) errs.confirm = '两次密码不一致'
    if (!form.email.trim()) errs.email = '请输入邮箱'
    if (!form.emailCode.trim()) errs.emailCode = '请输入验证码'
    setErrors(errs)
    if (Object.keys(errs).length > 0) return

    setLoading(true)
    const success = await register({
      nickname: form.nickname,
      email: form.email,
      password: form.password,
      emailCode: form.emailCode,
    })
    setLoading(false)
    if (success) {
      showToast('注册成功，请登录', 'success')
      navigate('/login')
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-logo">
          <div className="icon">💬</div>
          <h1>创建账号</h1>
          <p>加入我们，开始聊天</p>
        </div>

        <div className="avatar-upload">
          <div className="avatar-preview" onClick={handleAvatarClick}>
            {avatarUrl ? <img src={avatarUrl} alt="头像" /> : <span>📷</span>}
          </div>
          <span>点击上传头像（可选）</span>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            style={{ display: 'none' }}
            onChange={handleAvatarChange}
          />
        </div>

        <form onSubmit={handleRegister}>
          <div className="form-group">
            <input
              className={`form-input ${errors.nickname ? 'error' : ''}`}
              placeholder="用户名"
              value={form.nickname}
              onChange={e => update('nickname', e.target.value)}
            />
            {errors.nickname && <div className="form-error">{errors.nickname}</div>}
          </div>
          <div className="form-group">
            <input
              className={`form-input ${errors.password ? 'error' : ''}`}
              type="password"
              placeholder="密码（至少6位）"
              value={form.password}
              onChange={e => update('password', e.target.value)}
            />
            {errors.password && <div className="form-error">{errors.password}</div>}
          </div>
          <div className="form-group">
            <input
              className={`form-input ${errors.confirm ? 'error' : ''}`}
              type="password"
              placeholder="确认密码"
              value={form.confirm}
              onChange={e => update('confirm', e.target.value)}
            />
            {errors.confirm && <div className="form-error">{errors.confirm}</div>}
          </div>
          <div className="form-group">
            <input
              className={`form-input ${errors.email ? 'error' : ''}`}
              type="email"
              placeholder="邮箱地址"
              value={form.email}
              onChange={e => update('email', e.target.value)}
            />
            {errors.email && <div className="form-error">{errors.email}</div>}
          </div>
          <div className="form-group">
            <div className="form-row">
              <input
                className={`form-input ${errors.emailCode ? 'error' : ''}`}
                placeholder="验证码"
                value={form.emailCode}
                onChange={e => update('emailCode', e.target.value)}
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
            {errors.emailCode && <div className="form-error">{errors.emailCode}</div>}
          </div>
          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? '注册中...' : '注册'}
          </button>
        </form>

        <div className="auth-footer">
          已有账号？<span className="link" onClick={() => navigate('/login')}>返回登录</span>
        </div>
      </div>
    </div>
  )
}
