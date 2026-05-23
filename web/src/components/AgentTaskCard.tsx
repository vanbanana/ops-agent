// 多Agent并行任务卡片 — 每个子Agent独立卡片，显示角色+状态+进度
import { type FC, useState } from 'react'
import type { AgentChatMessage } from '../types/api'

interface AgentTaskCardProps {
  messages: AgentChatMessage[]
  currentRound: number
  activeRole: string | null
  status: string
}

const ROLE_STYLES: Record<string, { label: string; icon: string; color: string; bgColor: string }> = {
  planner:     { label: 'Planner',     icon: 'architecture', color: '#8B949E', bgColor: 'rgba(139,148,158,0.08)' },
  coordinator: { label: 'Coordinator', icon: 'hub',          color: '#D29922', bgColor: 'rgba(210,153,34,0.08)' },
  executor:    { label: 'Executor',    icon: 'play_arrow',   color: '#388BFD', bgColor: 'rgba(56,139,253,0.08)' },
  verifier:    { label: 'Verifier',    icon: 'verified',     color: '#3FB950', bgColor: 'rgba(63,185,80,0.08)' },
}

const DEFAULT_STYLE = { label: 'Agent', icon: 'smart_toy', color: '#8B949E', bgColor: 'rgba(139,148,158,0.08)' }

export const AgentTaskCard: FC<AgentTaskCardProps> = ({ messages, currentRound, activeRole, status }) => {
  // Group messages by role
  const roleGroups = new Map<string, AgentChatMessage[]>()
  for (const msg of messages) {
    const arr = roleGroups.get(msg.role) || []
    arr.push(msg)
    roleGroups.set(msg.role, arr)
  }

  return (
    <div style={{ margin: '8px 0', border: '1px solid var(--ops-border-subtle)', borderRadius: 6, overflow: 'hidden' }}>
      {/* Header bar */}
      <div style={{ height: 32, display: 'flex', alignItems: 'center', padding: '0 12px', gap: 8, background: 'var(--ops-bg-elevated)', borderBottom: '1px solid var(--ops-border-subtle)' }}>
        <span className="material-symbols-outlined" style={{ fontSize: 14, color: activeRole ? 'var(--ops-status-info)' : 'var(--ops-fg-muted)', animation: activeRole ? 'spin 2s linear infinite' : 'none' }}>
          group
        </span>
        <span style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-secondary)' }}>
          {status || `多Agent协作 · 第${currentRound}轮`}
        </span>
        <div style={{ flex: 1 }} />
        {/* Phase indicators */}
        {['planner', 'executor', 'verifier'].map(role => {
          const style = ROLE_STYLES[role] || DEFAULT_STYLE
          const hasMessages = roleGroups.has(role)
          const isActive = activeRole === role
          return (
            <span key={role} style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
              <span style={{ width: 6, height: 6, borderRadius: '50%', background: isActive ? style.color : hasMessages ? style.color : 'var(--ops-fg-muted)', opacity: hasMessages ? 1 : 0.3, animation: isActive ? 'pulse 1.5s infinite' : 'none' }} />
              <span style={{ fontSize: 10, color: isActive ? style.color : 'var(--ops-fg-muted)' }}>{style.label}</span>
            </span>
          )
        })}
      </div>

      {/* Agent cards — each role in its own section */}
      <div style={{ padding: 8, display: 'flex', flexDirection: 'column', gap: 6 }}>
        {Array.from(roleGroups.entries()).map(([role, msgs]) => (
          <RoleSection key={role} role={role} messages={msgs} isActive={activeRole === role} />
        ))}
      </div>
    </div>
  )
}

const RoleSection: FC<{ role: string; messages: AgentChatMessage[]; isActive: boolean }> = ({ role, messages, isActive }) => {
  const [expanded, setExpanded] = useState(true)
  const style = ROLE_STYLES[role] || DEFAULT_STYLE

  return (
    <div style={{ borderLeft: `3px solid ${style.color}`, borderRadius: 2, background: style.bgColor }}>
      {/* Role header */}
      <button
        onClick={() => setExpanded(!expanded)}
        style={{ width: '100%', display: 'flex', alignItems: 'center', gap: 6, padding: '4px 8px', border: 'none', background: 'none', cursor: 'pointer' }}
      >
        <span className="material-symbols-outlined" style={{ fontSize: 14, color: style.color, animation: isActive ? 'spin 1.5s linear infinite' : 'none' }}>
          {isActive ? 'progress_activity' : style.icon}
        </span>
        <span style={{ fontSize: 11, fontWeight: 600, color: style.color }}>{style.label}</span>
        <span style={{ fontSize: 10, color: 'var(--ops-fg-muted)' }}>({messages.length} 条)</span>
        <div style={{ flex: 1 }} />
        <span className="material-symbols-outlined" style={{ fontSize: 12, color: 'var(--ops-fg-muted)', transform: expanded ? 'rotate(180deg)' : 'none' }}>expand_more</span>
      </button>

      {/* Messages */}
      {expanded && (
        <div style={{ padding: '2px 8px 6px 24px', display: 'flex', flexDirection: 'column', gap: 3 }}>
          {messages.map(msg => (
            <div key={msg.id} style={{ fontSize: 12, color: 'var(--ops-fg-primary)', lineHeight: '18px' }}>
              <span style={{ fontSize: 10, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)', marginRight: 6 }}>{msg.timestamp}</span>
              {msg.content}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
