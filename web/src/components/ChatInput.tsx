// Data: POST /api/v1/chat/stream (sends user message)
import { type FC, useState, useRef, useEffect, useCallback } from 'react'

interface ChatInputProps {
  onSend: (message: string) => void
  disabled: boolean
}

const MAX_CHARS = 4000

const COMMANDS = [
  { cmd: '/disk', label: '查看磁盘', message: '看下磁盘使用情况' },
  { cmd: '/load', label: '查看负载', message: '看下系统负载' },
  { cmd: '/mem', label: '查看内存', message: '看下内存使用情况' },
  { cmd: '/proc', label: '查看进程', message: '看下进程状态' },
  { cmd: '/net', label: '查看网络', message: '看下网络状态' },
  { cmd: '/health', label: '健康检查', message: '做一次健康检查' },
  { cmd: '/clear', label: '清空对话', message: '清空对话' },
]

export const ChatInput: FC<ChatInputProps> = ({ onSend, disabled }) => {
  const [text, setText] = useState('')
  const [showCommands, setShowCommands] = useState(false)
  const [filteredCommands, setFilteredCommands] = useState(COMMANDS)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const adjustHeight = useCallback(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`
  }, [])

  useEffect(() => {
    adjustHeight()
  }, [text, adjustHeight])

  useEffect(() => {
    if (text.startsWith('/')) {
      const query = text.toLowerCase()
      const matched = COMMANDS.filter(c => c.cmd.startsWith(query))
      setFilteredCommands(matched)
      setShowCommands(matched.length > 0)
    } else {
      setShowCommands(false)
    }
  }, [text])

  const handleSend = () => {
    const trimmed = text.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    setText('')
    setShowCommands(false)
    if (textareaRef.current) textareaRef.current.style.height = 'auto'
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
    if (e.key === 'Escape') {
      setShowCommands(false)
    }
  }

  const handleCommandClick = (message: string) => {
    setText(message)
    setShowCommands(false)
    textareaRef.current?.focus()
  }

  const charColor = text.length > MAX_CHARS * 0.9 ? 'var(--ops-status-warn)' : 'var(--ops-fg-muted)'

  return (
    <div
      style={{
        padding: '10px 48px',
        background: 'var(--ops-bg-canvas)',
        flexShrink: 0,
        position: 'relative',
      }}
    >
      {/* Command palette dropdown */}
      {showCommands && (
        <div
          ref={dropdownRef}
          style={{
            position: 'absolute',
            bottom: '100%',
            left: 48,
            right: 48,
            background: 'var(--ops-bg-elevated)',
            border: '1px solid var(--ops-border-default)',
            borderRadius: 0,
            zIndex: 100,
            maxHeight: 240,
            overflowY: 'auto',
          }}
        >
          {filteredCommands.map((c) => (
            <div
              key={c.cmd}
              onClick={() => handleCommandClick(c.message)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '6px 12px',
                cursor: 'pointer',
                borderBottom: '1px solid var(--ops-border-default)',
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLDivElement).style.background = 'var(--ops-bg-input)'
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLDivElement).style.background = 'transparent'
              }}
            >
              <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 12, color: 'var(--ops-fg-primary)' }}>
                {c.cmd}
              </span>
              <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>
                {c.label}
              </span>
            </div>
          ))}
        </div>
      )}

      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          gap: 8,
          padding: '8px 12px',
          borderRadius: 4,
          background: 'var(--ops-bg-input)',
          border: '1px solid var(--ops-border-default)',
        }}
      >
        <textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => setText(e.target.value.slice(0, MAX_CHARS))}
          onKeyDown={handleKeyDown}
          placeholder="描述你的运维诉求，例如：看看 /var 还剩多少空间"
          disabled={disabled}
          rows={1}
          style={{
            flex: 1,
            background: 'transparent',
            border: 'none',
            outline: 'none',
            resize: 'none',
            fontFamily: 'var(--ops-font-ui)',
            fontSize: 13,
            lineHeight: 1.5,
            color: 'var(--ops-fg-primary)',
            minHeight: 20,
            maxHeight: 200,
          }}
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: charColor }}>
            {text.length}/{MAX_CHARS}
          </span>
          <button
            onClick={handleSend}
            disabled={disabled || !text.trim()}
            style={{
              width: 26,
              height: 26,
              borderRadius: 4,
              border: 'none',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: text.trim() && !disabled ? 'pointer' : 'not-allowed',
              background: text.trim() && !disabled ? 'var(--ops-fg-primary)' : 'var(--ops-border-default)',
              opacity: text.trim() && !disabled ? 1 : 0.5,
            }}
          >
            <span className="material-symbols-outlined" style={{ fontSize: 16, color: 'var(--ops-bg-canvas)' }}>
              arrow_upward
            </span>
          </button>
        </div>
      </div>
    </div>
  )
}
