// Data: GET /health (连接状态)
import { type FC } from 'react'
import type { HealthResponse } from '../types/api'

interface AppHeaderProps {
  health: HealthResponse | null
  mode: 'chat' | 'desktop'
  onModeChange: (mode: 'chat' | 'desktop') => void
  connected: boolean
  onLogout?: () => void
}

export const AppHeader: FC<AppHeaderProps> = ({ health, mode, onModeChange, connected, onLogout }) => {

  return (
    <header
      style={{
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
        flexShrink: 0,
      }}
    >
      {/* Top row — breadcrumb + actions (40px) */}
      <div
        style={{
          height: 40,
          display: 'flex',
          alignItems: 'center',
          padding: '0 12px',
          background: 'var(--ops-bg-surface)',
          borderBottom: '1px solid var(--ops-border-subtle)',
          gap: 8,
        }}
      >
        {/* Logo */}
        <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 13, fontWeight: 600, color: 'var(--ops-fg-primary)', letterSpacing: '0.5px' }}>
          OPS
        </span>

        {/* Breadcrumb separator */}
        <span className="material-symbols-outlined" style={{ fontSize: 14, color: 'var(--ops-fg-muted)' }}>chevron_right</span>

        {/* Current context */}
        <span style={{ fontSize: 12, color: 'var(--ops-fg-secondary)' }}>
          admin@ops-agent
        </span>

        {/* Spacer */}
        <div style={{ flex: 1 }} />

        {/* Right actions */}
        <HeaderBtn icon="notifications" />
        <HeaderBtn icon="splitscreen" />
        <HeaderBtn icon="fullscreen" />
        {onLogout && <HeaderBtn icon="logout" onClick={onLogout} />}
      </div>

      {/* Second row — tabs + info (36px) */}
      <div
        style={{
          height: 36,
          display: 'flex',
          alignItems: 'center',
          padding: '0 12px',
          background: 'var(--ops-bg-surface)',
          borderBottom: '1px solid var(--ops-border-subtle)',
          gap: 4,
        }}
      >
        {/* Tab: 对话 — filled style like Alibaba's Shell tab */}
        <TabButton label="对话" icon="chat" active={mode === 'chat'} onClick={() => onModeChange('chat')} />
        <TabButton label="桌面" icon="grid_view" active={mode === 'desktop'} onClick={() => onModeChange('desktop')} />

        {/* Separator */}
        <div style={{ width: 1, height: 16, background: 'var(--ops-border-default)', margin: '0 8px' }} />

        {/* Connection indicator — like "82 ms" in Alibaba */}
        <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          <span className="material-symbols-outlined" style={{ fontSize: 13, color: connected ? 'var(--ops-status-ok)' : 'var(--ops-status-danger)' }}>
            {connected ? 'check_circle' : 'error'}
          </span>
          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
            {connected ? (health?.components?.llm?.latency_ms ? `${health.components.llm.latency_ms}ms` : 'ok') : '断开'}
          </span>
        </span>

        <div style={{ flex: 1 }} />
      </div>
    </header>
  )
}

const TabButton: FC<{ label: string; icon: string; active: boolean; onClick: () => void }> = ({ label, icon, active, onClick }) => (
  <button
    onClick={onClick}
    style={{
      height: 26,
      display: 'flex',
      alignItems: 'center',
      gap: 4,
      padding: '0 10px',
      borderRadius: 4,
      border: 'none',
      cursor: 'pointer',
      fontSize: 12,
      fontFamily: 'var(--ops-font-ui)',
      fontWeight: active ? 500 : 400,
      color: active ? 'var(--ops-fg-primary)' : 'var(--ops-fg-secondary)',
      background: active ? 'var(--ops-bg-elevated)' : 'transparent',
    }}
  >
    <span className="material-symbols-outlined" style={{ fontSize: 14 }}>{icon}</span>
    {label}
  </button>
)

const HeaderBtn: FC<{ icon: string; onClick?: () => void }> = ({ icon, onClick }) => (
  <button
    onClick={onClick}
    style={{
      width: 28,
      height: 28,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      borderRadius: 4,
      border: 'none',
      background: 'transparent',
      cursor: 'pointer',
      color: 'var(--ops-fg-muted)',
    }}
  >
    <span className="material-symbols-outlined" style={{ fontSize: 16 }}>{icon}</span>
  </button>
)
