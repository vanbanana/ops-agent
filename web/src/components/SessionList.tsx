// Data: GET /api/v1/sessions (未就绪，本地状态管理)
import { type FC } from 'react'
import type { Session } from '../types/api'

interface SessionListProps {
  sessions: Session[]
  activeId: string | null
  onSelect: (id: string) => void
  onNew: () => void
}

export const SessionList: FC<SessionListProps> = ({
  sessions,
  activeId,
  onSelect,
  onNew,
}) => {
  return (
    <aside
      style={{
        width: 200,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--ops-bg-surface)',
        borderRight: '1px solid var(--ops-border-subtle)',
        flexShrink: 0,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          height: 36,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 12px',
          borderBottom: '1px solid var(--ops-border-subtle)',
          flexShrink: 0,
        }}
      >
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, fontWeight: 500, color: 'var(--ops-fg-secondary)' }}>
          会话
        </span>
        <button
          onClick={onNew}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 22,
            height: 22,
            borderRadius: 3,
            border: 'none',
            background: 'transparent',
            cursor: 'pointer',
            color: 'var(--ops-fg-muted)',
          }}
          title="新建会话"
        >
          <span className="material-symbols-outlined" style={{ fontSize: 16 }}>add</span>
        </button>
      </div>

      {/* Session list */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '4px 0' }}>
        {sessions.map((session) => {
          const isActive = activeId === session.id
          return (
            <button
              key={session.id}
              onClick={() => onSelect(session.id)}
              style={{
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
                gap: 2,
                padding: '8px 12px',
                border: 'none',
                textAlign: 'left',
                cursor: 'pointer',
                background: isActive ? 'var(--ops-bg-elevated)' : 'transparent',
                borderLeft: isActive ? '2px solid var(--ops-status-info)' : '2px solid transparent',
              }}
            >
              <span
                style={{
                  fontFamily: 'var(--ops-font-ui)',
                  fontSize: 12,
                  color: 'var(--ops-fg-primary)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {session.title}
              </span>
              {session.last_message && (
                <span
                  style={{
                    fontFamily: 'var(--ops-font-mono)',
                    fontSize: 10,
                    color: 'var(--ops-fg-muted)',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {session.last_message.slice(0, 28)}
                </span>
              )}
            </button>
          )
        })}

        {sessions.length === 0 && (
          <div style={{ padding: '24px 12px', textAlign: 'center', fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
            暂无会话
          </div>
        )}
      </div>
    </aside>
  )
}
