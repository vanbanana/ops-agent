import { authFetch } from '../lib/auth'
import { type FC, useState } from 'react'
import type { PermissionRequestData } from '../types/api'

interface PermissionBannerProps {
  permission: PermissionRequestData
  onRespond: (requestId: string, action: 'allow' | 'allow_session' | 'deny') => void
}

export const PermissionBanner: FC<PermissionBannerProps> = ({ permission, onRespond }) => {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleClick = async (action: 'allow' | 'allow_session' | 'deny') => {
    setLoading(true)
    setError(null)
    try {
      const res = await authFetch('/api/v1/permission/respond', {
        method: 'POST',
        body: JSON.stringify({ request_id: permission.request_id, action }),
      })
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`)
      }
      onRespond(permission.request_id, action)
    } catch (e) {
      setError('发送失败，请重试')
      setLoading(false)
      // Auto-retry once after 2s
      setTimeout(async () => {
        try {
          const res = await authFetch('/api/v1/permission/respond', {
            method: 'POST',
            body: JSON.stringify({ request_id: permission.request_id, action }),
          })
          if (res.ok) {
            onRespond(permission.request_id, action)
            setError(null)
          }
        } catch {
          // Keep showing error
        }
      }, 2000)
    }
  }

  // Truncate long commands
  const displayCommand = permission.command.length > 60
    ? permission.command.slice(0, 60) + '...'
    : permission.command

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        height: 40,
        padding: '0 12px',
        background: 'var(--ops-bg-surface)',
        borderBottom: '1px solid var(--ops-border-subtle)',
        transition: 'max-height 150ms ease, opacity 150ms ease',
      }}
    >
      {/* Left: icon + command text */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0 }}>
        <span className="material-symbols-outlined" style={{ fontSize: 18, color: 'var(--ops-status-warn)' }}>
          warning
        </span>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-secondary)', whiteSpace: 'nowrap' }}>
          Agent 请求执行:
        </span>
        <span
          style={{
            fontFamily: 'var(--ops-font-mono)',
            fontSize: 12,
            color: 'var(--ops-fg-primary)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
          title={permission.command}
        >
          {displayCommand}
        </span>
      </div>

      {/* Right: buttons */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
        {error && (
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-status-danger)' }}>
            {error}
          </span>
        )}
        <button
          onClick={() => handleClick('allow')}
          disabled={loading}
          style={{
            width: 64,
            height: 28,
            borderRadius: 6,
            border: 'none',
            background: 'var(--ops-status-ok)',
            color: '#fff',
            fontFamily: 'var(--ops-font-ui)',
            fontSize: 12,
            fontWeight: 500,
            cursor: loading ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.6 : 1,
          }}
        >
          同意
        </button>
        <button
          onClick={() => handleClick('deny')}
          disabled={loading}
          style={{
            width: 64,
            height: 28,
            borderRadius: 6,
            border: '1px solid var(--ops-border-default)',
            background: 'transparent',
            color: 'var(--ops-fg-secondary)',
            fontFamily: 'var(--ops-font-ui)',
            fontSize: 12,
            fontWeight: 500,
            cursor: loading ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.6 : 1,
          }}
        >
          拒绝
        </button>
      </div>
    </div>
  )
}
