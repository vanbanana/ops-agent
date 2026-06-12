// Data: POST /api/v1/auth/login
import { type FC, useState, useCallback } from 'react'
import { setAuthToken } from '../lib/auth'

interface LoginPageProps {
  onLoginSuccess: (token: string) => void
}

export const LoginPage: FC<LoginPageProps> = ({ onLoginSuccess }) => {
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [lockRemaining, setLockRemaining] = useState<number | null>(null)

  const handleSubmit = useCallback(async (e?: React.FormEvent) => {
    e?.preventDefault()
    if (!username.trim() || !password.trim() || loading) return

    setError(null)
    setLoading(true)
    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const json = await res.json()

      if (res.ok && json.data?.token) {
        setAuthToken(json.data.token)
        onLoginSuccess(json.data.token)
      } else if (res.status === 429 || json.error_code === 'AUTH_RATE_001' || json.error_code === 'AUTH_LOCKED_001') {
        const remaining = json.remaining_seconds ?? json.lock_remaining_seconds ?? 180
        setLockRemaining(remaining)
        setError(`IP 已锁定，剩余 ${formatLockTime(remaining)}`)
        startLockCountdown(remaining)
      } else {
        setError(json.message ?? json.error ?? '用户名或密码错误')
      }
    } catch {
      setError('无法连接服务器')
    } finally {
      setLoading(false)
    }
  }, [username, password, loading, onLoginSuccess])

  const startLockCountdown = (seconds: number) => {
    const interval = setInterval(() => {
      seconds--
      if (seconds <= 0) {
        clearInterval(interval)
        setLockRemaining(null)
        setError(null)
      } else {
        setLockRemaining(seconds)
        setError(`IP 已锁定，剩余 ${formatLockTime(seconds)}`)
      }
    }, 1000)
  }

  const isLocked = lockRemaining !== null && lockRemaining > 0

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', background: 'var(--ops-bg-canvas)' }}>
      <form
        onSubmit={handleSubmit}
        style={{ width: 320, display: 'flex', flexDirection: 'column', gap: 16 }}
      >
        {/* Branding */}
        <div style={{ textAlign: 'center', marginBottom: 8 }}>
          <div style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 18, fontWeight: 600, color: 'var(--ops-fg-primary)', letterSpacing: 1 }}>
            OPS·AGENT
          </div>
          <div style={{ fontSize: 12, color: 'var(--ops-fg-muted)', marginTop: 4 }}>
            Linux 运维智能体
          </div>
        </div>

        {/* Username */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          <label style={{ fontSize: 12, color: 'var(--ops-fg-secondary)' }}>账号</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoFocus
            style={{
              height: 36, padding: '0 12px', fontSize: 13, fontFamily: 'var(--ops-font-mono)',
              color: 'var(--ops-fg-primary)', background: 'var(--ops-bg-surface)',
              border: `1px solid ${error ? 'var(--ops-status-danger)' : 'var(--ops-border-default)'}`,
              borderRadius: 4, outline: 'none',
            }}
          />
        </div>

        {/* Password */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          <label style={{ fontSize: 12, color: 'var(--ops-fg-secondary)' }}>密码</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={{
              height: 36, padding: '0 12px', fontSize: 13,
              color: 'var(--ops-fg-primary)', background: 'var(--ops-bg-surface)',
              border: `1px solid ${error ? 'var(--ops-status-danger)' : 'var(--ops-border-default)'}`,
              borderRadius: 4, outline: 'none',
            }}
          />
        </div>

        {/* Error */}
        {error && (
          <div style={{ fontSize: 12, color: 'var(--ops-status-danger)', fontFamily: 'var(--ops-font-mono)' }}>
            {error}
          </div>
        )}

        {/* Submit */}
        <button
          type="submit"
          disabled={loading || isLocked}
          style={{
            height: 36, fontSize: 13, fontWeight: 500,
            color: '#fff', background: isLocked ? 'var(--ops-fg-muted)' : 'var(--ops-status-info)',
            border: 'none', borderRadius: 4,
            cursor: loading || isLocked ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.7 : 1,
          }}
        >
          {loading ? '验证中...' : isLocked ? `已锁定 ${formatLockTime(lockRemaining!)}` : '登录'}
        </button>

        {/* Warning */}
        <div style={{ fontSize: 11, color: 'var(--ops-fg-muted)', textAlign: 'center' }}>
          5 次错误尝试将锁定 IP 3 分钟
        </div>
      </form>

      {/* Footer */}
      <div style={{ position: 'absolute', bottom: 16, fontSize: 11, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)' }}>
        OPS·AGENT v0.1.0
      </div>
    </div>
  )
}

function formatLockTime(seconds: number): string {
  const min = Math.floor(seconds / 60)
  const sec = seconds % 60
  return `${min}:${String(sec).padStart(2, '0')}`
}
