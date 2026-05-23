// Data: GET /health (连接状态), 终端尺寸 (Rows/Cols)
// 对标: 阿里云 Workbench 底部状态栏 22px
import { type FC } from 'react'
import type { HealthResponse } from '../types/api'

interface StatusBarProps {
  health: HealthResponse | null
  connected: boolean
}

export const StatusBar: FC<StatusBarProps> = ({ connected }) => {

  return (
    <footer
      style={{
        height: 22,
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        padding: '0 10px',
        background: 'var(--ops-bg-sidebar)',
        borderTop: '1px solid var(--ops-border-subtle)',
        flexShrink: 0,
        gap: 12,
        fontFamily: 'var(--ops-font-ui)',
        fontSize: 11,
        color: 'var(--ops-fg-muted)',
        userSelect: 'none',
      }}
    >
      {/* Connection status */}
      <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <span style={{ width: 6, height: 6, borderRadius: '50%', background: connected ? 'var(--ops-status-ok)' : 'var(--ops-status-danger)' }} />
        <span>{connected ? '已连接' : '未连接'}</span>
      </span>

      <Separator />

      {/* Instance info */}
      <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11 }}>
        ops-agent@localhost
      </span>

      <Separator />

      {/* Version */}
      <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11 }}>
        v0.1.0
      </span>

      {/* Spacer */}
      <div style={{ flex: 1 }} />

      {/* Right side info */}
      <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11 }}>
        UTF-8
      </span>
    </footer>
  )
}

const Separator: FC = () => (
  <span style={{ color: 'var(--ops-border-default)' }}>|</span>
)
