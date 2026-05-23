// Data: SSE events with agent_role (planner/executor/verifier)
// 多Agent协作群聊视图 — 显示 Planner/Executor/Verifier 像群聊一样通信
import { type FC, useRef, useEffect, useState } from 'react'

export type AgentRole = 'planner' | 'executor' | 'verifier' | 'coordinator'

export interface AgentMessage {
  id: string
  role: AgentRole
  content: string
  timestamp: string
  round: number
  tokensUsed?: number
  toolName?: string
  expandedPrompt?: string
}

interface MultiAgentChatProps {
  messages: AgentMessage[]
  currentRound: number
  activeRole: AgentRole | null
  status: string // "协作中 · 第1轮 · Planner正在规划..."
}

const ROLE_CONFIG: Record<string, { label: string; icon: string; color: string }> = {
  planner:     { label: 'Planner',     icon: 'architecture', color: '#8B949E' },
  executor:    { label: 'Executor',    icon: 'play_arrow',   color: '#388BFD' },
  verifier:    { label: 'Verifier',    icon: 'verified',     color: '#3FB950' },
  coordinator: { label: 'Coordinator', icon: 'hub',          color: '#D29922' },
}

const DEFAULT_ROLE_CONFIG = { label: 'Agent', icon: 'smart_toy', color: '#8B949E' }

export const MultiAgentChat: FC<MultiAgentChatProps> = ({ messages, currentRound, activeRole, status }) => {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Status bar */}
      <div style={{ height: 32, display: 'flex', alignItems: 'center', padding: '0 12px', borderBottom: '1px solid var(--ops-border-subtle)', flexShrink: 0, gap: 8 }}>
        <span className="material-symbols-outlined" style={{ fontSize: 14, color: activeRole ? 'var(--ops-status-info)' : 'var(--ops-fg-muted)', animation: activeRole ? 'pulse 1.5s infinite' : 'none' }}>
          group
        </span>
        <span style={{ fontSize: 11, color: 'var(--ops-fg-secondary)', fontFamily: 'var(--ops-font-mono)' }}>
          {status}
        </span>
        <div style={{ flex: 1 }} />
        <span style={{ fontSize: 11, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)' }}>
          第 {currentRound} 轮
        </span>
      </div>

      {/* Messages */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
        {messages.map(msg => (
          <AgentMessageBubble key={msg.id} message={msg} isTyping={activeRole === msg.role && msg === messages[messages.length - 1]} />
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}

const AgentMessageBubble: FC<{ message: AgentMessage; isTyping: boolean }> = ({ message, isTyping }) => {
  const config = ROLE_CONFIG[message.role] || DEFAULT_ROLE_CONFIG
  const [expanded, setExpanded] = useState(false)

  return (
    <div style={{ display: 'flex', gap: 8 }}>
      {/* Avatar */}
      <div style={{ width: 28, height: 28, borderRadius: '50%', background: 'var(--ops-bg-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, border: `1px solid ${config.color}33` }}>
        <span className="material-symbols-outlined" style={{ fontSize: 14, color: config.color }}>{config.icon}</span>
      </div>

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {/* Header: role + timestamp */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
          <span style={{ fontSize: 11, fontWeight: 600, color: config.color }}>{config.label}</span>
          <span style={{ fontSize: 10, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)' }}>{message.timestamp}</span>
          {message.tokensUsed && (
            <span style={{ fontSize: 10, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)' }}>{message.tokensUsed} tokens</span>
          )}
        </div>

        {/* Message body */}
        <div style={{ fontSize: 12, color: 'var(--ops-fg-primary)', lineHeight: '18px', whiteSpace: 'pre-wrap', borderLeft: `2px solid ${config.color}33`, paddingLeft: 8 }}>
          {message.content}
          {isTyping && <span style={{ animation: 'blink 1s infinite' }}>▊</span>}
        </div>

        {/* Tool call indicator for executor */}
        {message.toolName && (
          <div style={{ marginTop: 4, fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)' }}>
            ▸ 调用: {message.toolName}
          </div>
        )}

        {/* Expand raw prompt (for technical reviewers) */}
        {message.expandedPrompt && (
          <div style={{ marginTop: 4 }}>
            <button
              onClick={() => setExpanded(!expanded)}
              style={{ fontSize: 10, color: 'var(--ops-fg-muted)', background: 'none', border: 'none', cursor: 'pointer', textDecoration: 'underline', padding: 0 }}
            >
              {expanded ? '收起原始 prompt' : '展开原始 prompt'}
            </button>
            {expanded && (
              <pre style={{ fontSize: 10, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)', background: 'var(--ops-bg-canvas)', padding: 8, borderRadius: 4, marginTop: 4, maxHeight: 200, overflowY: 'auto', whiteSpace: 'pre-wrap' }}>
                {message.expandedPrompt}
              </pre>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// Need useState import for the bubble's expanded state — already imported at top
