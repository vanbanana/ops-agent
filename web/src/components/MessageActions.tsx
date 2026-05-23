import { type FC, useState, useCallback } from 'react'

interface MessageActionsProps {
  content: string
  isStreaming: boolean
  onRetry: () => void
  onStop: () => void
}

export const MessageActions: FC<MessageActionsProps> = ({ content, isStreaming, onRetry, onStop }) => {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(() => {
    if (!content) return
    navigator.clipboard.writeText(content).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }, [content])

  if (isStreaming) {
    return (
      <div style={{ display: 'flex', gap: 6, alignItems: 'center', height: 26 }}>
        <ActionBtn icon="stop_circle" label="停止生成" onClick={onStop} color="var(--ops-status-warn)" />
      </div>
    )
  }

  if (!content) return null

  return (
    <div style={{ display: 'flex', gap: 6, alignItems: 'center', height: 26, opacity: 0, transition: 'opacity 150ms' }} className="msg-actions">
      <ActionBtn icon={copied ? 'check' : 'content_copy'} label="复制" onClick={handleCopy} color={copied ? 'var(--ops-status-ok)' : undefined} />
      <ActionBtn icon="refresh" label="重新生成" onClick={onRetry} />
    </div>
  )
}

const ActionBtn: FC<{ icon: string; label: string; onClick: () => void; color?: string }> = ({ icon, label, onClick, color }) => (
  <div
    onClick={onClick}
    style={{
      display: 'flex',
      alignItems: 'center',
      gap: 4,
      height: 26,
      padding: '0 8px',
      borderRadius: 6,
      cursor: 'pointer',
    }}
  >
    <span className="material-symbols-outlined" style={{ fontSize: 13, color: color || 'var(--ops-fg-muted)' }}>
      {icon}
    </span>
    <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, color: color || 'var(--ops-fg-muted)' }}>
      {label}
    </span>
  </div>
)
