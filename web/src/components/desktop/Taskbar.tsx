// Data: GET /health (健康灯)
import { type FC, useState, useEffect } from 'react'
import type { WindowState } from '../../hooks/useWindowManager'
import type { HealthResponse } from '../../types/api'

interface TaskbarProps {
  windows: WindowState[]
  focusedId: string | null
  health: HealthResponse | null
  connected: boolean
  onToggleWindow: (id: string) => void
}

export const Taskbar: FC<TaskbarProps> = ({ windows, focusedId, health, connected, onToggleWindow }) => {
  const [time, setTime] = useState(formatTime())

  useEffect(() => {
    const interval = setInterval(() => setTime(formatTime()), 1000)
    return () => clearInterval(interval)
  }, [])

  const statusColor = !connected
    ? 'var(--ops-status-danger)'
    : health?.status === 'healthy'
    ? 'var(--ops-status-ok)'
    : health?.status === 'degraded'
    ? 'var(--ops-status-warn)'
    : 'var(--ops-status-danger)'

  return (
    <div
      style={{
        height: 40,
        display: 'flex',
        alignItems: 'center',
        padding: '0 8px',
        background: 'var(--ops-bg-surface)',
        borderTop: '1px solid var(--ops-border-subtle)',
        gap: 2,
        flexShrink: 0,
      }}
    >
      {/* Running window tabs */}
      {windows.map(win => (
        <button
          key={win.id}
          onClick={() => onToggleWindow(win.id)}
          style={{
            height: 30,
            display: 'flex',
            alignItems: 'center',
            gap: 4,
            padding: '0 10px',
            borderRadius: 4,
            border: 'none',
            cursor: 'pointer',
            fontSize: 11,
            fontFamily: 'var(--ops-font-ui)',
            color: focusedId === win.id ? 'var(--ops-fg-primary)' : 'var(--ops-fg-secondary)',
            background: focusedId === win.id ? 'var(--ops-bg-elevated)' : 'transparent',
            opacity: win.minimized ? 0.6 : 1,
            maxWidth: 160,
            overflow: 'hidden',
          }}
        >
          <span className="material-symbols-outlined" style={{ fontSize: 14 }}>{win.icon}</span>
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{win.title}</span>
        </button>
      ))}

      <div style={{ flex: 1 }} />

      {/* System tray */}
      <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <span style={{ width: 6, height: 6, borderRadius: '50%', background: statusColor }} />
        <span style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)' }}>
          {connected ? (health?.status ?? '--') : '断开'}
        </span>
      </span>

      <div style={{ width: 1, height: 16, background: 'var(--ops-border-subtle)', margin: '0 6px' }} />

      <span style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-secondary)' }}>
        {time}
      </span>
    </div>
  )
}

function formatTime(): string {
  const now = new Date()
  return now.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
}
