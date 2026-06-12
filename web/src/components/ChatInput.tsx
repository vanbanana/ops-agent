import { authFetch } from '../lib/auth'
// Data: POST /api/v1/chat/stream (sends user message)
import { type FC, useState, useRef, useEffect, useCallback } from 'react'
import { PermissionBanner } from './PermissionBanner'
import type { PermissionRequestData } from '../types/api'

interface ModelOption {
  id: string
  name: string
  provider: string
  is_active: boolean
}

interface ChatInputProps {
  onSend: (message: string) => void
  disabled: boolean
  pendingPermission?: PermissionRequestData | null
  onPermissionRespond?: (requestId: string, action: 'allow' | 'allow_session' | 'deny') => void
  permissionMode?: 'default' | 'auto_approve' | 'plan'
  onPermissionModeChange?: (mode: 'default' | 'auto_approve' | 'plan') => void
  contextUsage?: number
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

export const ChatInput: FC<ChatInputProps> = ({ onSend, disabled, pendingPermission, onPermissionRespond, permissionMode = 'default', onPermissionModeChange, contextUsage = 0 }) => {
  const [text, setText] = useState('')
  const [showCommands, setShowCommands] = useState(false)
  const [filteredCommands, setFilteredCommands] = useState(COMMANDS)
  const [showModeConfirm, setShowModeConfirm] = useState(false)
  const [models, setModels] = useState<ModelOption[]>([])
  const [activeModel, setActiveModel] = useState<string>('')
  const [showModelPicker, setShowModelPicker] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const modelPickerRef = useRef<HTMLDivElement>(null)

  // Load models from pool
  useEffect(() => {
    authFetch('/api/v1/models/pool')
      .then(r => r.json())
      .then(d => {
        if (d?.data?.providers) {
          const list = d.data.providers.map((p: { id: string; name: string; provider: string; is_active: boolean }) => ({
            id: p.id, name: p.name, provider: p.provider, is_active: p.is_active,
          }))
          setModels(list)
          const active = list.find((m: ModelOption) => m.is_active)
          if (active) setActiveModel(active.name)
        }
      })
      .catch(() => {})
  }, [])

  // Close model picker on click outside
  useEffect(() => {
    if (!showModelPicker) return
    const handler = (e: MouseEvent) => {
      if (modelPickerRef.current && !modelPickerRef.current.contains(e.target as Node)) {
        setShowModelPicker(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showModelPicker])

  const [switchToast, setSwitchToast] = useState<string | null>(null)

  const handleModelSwitch = async (id: string, name: string) => {
    setShowModelPicker(false)
    try {
      const res = await authFetch('/api/v1/models/switch', {
        method: 'POST',
        body: JSON.stringify({ id }),
      })
      if (res.ok) {
        setActiveModel(name)
        setModels(prev => prev.map(m => ({ ...m, is_active: m.id === id })))
        setSwitchToast(`已切换: ${name}`)
        setTimeout(() => setSwitchToast(null), 2000)
      }
    } catch { /* ignore */ }
  }

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

  const isDisabledByPermission = !!pendingPermission && pendingPermission.status === 'pending'
  const effectiveDisabled = disabled || isDisabledByPermission

  return (
    <div
      style={{
        padding: '10px 24px',
        background: 'var(--ops-bg-canvas)',
        flexShrink: 0,
        position: 'relative',
      }}
    >
      {/* Model switch toast */}
      {switchToast && (
        <div style={{ position: 'absolute', top: -32, left: '50%', transform: 'translateX(-50%)', padding: '4px 12px', borderRadius: 4, background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-default)', fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: '#34c759', whiteSpace: 'nowrap', zIndex: 100 }}>
          {switchToast}
        </div>
      )}

      {/* Permission banner — appears above input when pending */}
      {pendingPermission && pendingPermission.status === 'pending' && onPermissionRespond && (
        <PermissionBanner permission={pendingPermission} onRespond={onPermissionRespond} />
      )}

      {/* Command palette dropdown */}
      {showCommands && (
        <div
          ref={dropdownRef}
          style={{
            position: 'absolute',
            bottom: '100%',
            left: 24,
            right: 24,
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

      {/* Input area */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          borderRadius: 8,
          background: 'var(--ops-bg-input)',
          border: '1px solid var(--ops-border-default)',
        }}
      >
        <textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => setText(e.target.value.slice(0, MAX_CHARS))}
          onKeyDown={handleKeyDown}
          placeholder={isDisabledByPermission ? '等待确认操作...' : '描述你的运维诉求，例如：看看 /var 还剩多少空间'}
          disabled={effectiveDisabled}
          rows={2}
          style={{
            flex: 1,
            background: 'transparent',
            border: 'none',
            outline: 'none',
            resize: 'none',
            fontFamily: 'var(--ops-font-ui)',
            fontSize: 13,
            lineHeight: 1.6,
            color: 'var(--ops-fg-primary)',
            minHeight: 56,
            maxHeight: 200,
            padding: '12px 14px 4px',
          }}
        />
        {/* Bottom toolbar inside input box */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '4px 10px 8px',
          }}
        >
          {/* Left: action icons */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <ToolbarBtn icon="tag" title="命令面板 (/)" onClick={() => { setText('/'); textareaRef.current?.focus() }} />
            <ToolbarBtn icon="attach_file" title="附件" onClick={() => {}} />
            <ContextRing percent={contextUsage} />
          </div>

          {/* Right: char count + model selector + mode toggle + send */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: charColor }}>
              {text.length}/{MAX_CHARS}
            </span>
            {/* Model selector */}
            <div style={{ position: 'relative' }} ref={modelPickerRef}>
              <button
                onClick={() => setShowModelPicker(!showModelPicker)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 3,
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: '2px 6px',
                  borderRadius: 4,
                }}
                title="切换模型"
              >
                <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-secondary)', maxWidth: 120, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {activeModel || 'loading...'}
                </span>
                <span className="material-symbols-outlined" style={{ fontSize: 14, color: 'var(--ops-fg-muted)' }}>
                  {showModelPicker ? 'expand_less' : 'expand_more'}
                </span>
              </button>
              {showModelPicker && models.length > 0 && (
                <div style={{ position: 'absolute', bottom: '100%', right: 0, marginBottom: 4, background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-default)', borderRadius: 6, padding: '4px 0', zIndex: 200, minWidth: 180, boxShadow: '0 4px 12px rgba(0,0,0,0.15)' }}>
                  {models.map(m => (
                    <button
                      key={m.id}
                      onClick={() => handleModelSwitch(m.id, m.name)}
                      style={{
                        width: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        padding: '6px 12px',
                        border: 'none',
                        background: m.is_active ? 'var(--ops-bg-canvas)' : 'transparent',
                        cursor: 'pointer',
                        textAlign: 'left',
                      }}
                    >
                      <span style={{ width: 6, height: 6, borderRadius: '50%', background: m.is_active ? '#34c759' : 'transparent', flexShrink: 0 }} />
                      <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-primary)', flex: 1 }}>{m.name}</span>
                      <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>{m.provider}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
            {/* Permission mode toggle (3 states: default → plan → auto_approve) */}
            <button
              onClick={() => {
                if (permissionMode === 'default') {
                  onPermissionModeChange?.('plan')
                } else if (permissionMode === 'plan') {
                  setShowModeConfirm(true)
                } else {
                  onPermissionModeChange?.('default')
                }
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 3,
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: '2px 4px',
                borderRadius: 4,
              }}
              title={
                permissionMode === 'auto_approve' ? '全权限模式（点击切回标准）' :
                permissionMode === 'plan' ? '计划模式（只读探查，点击切换全权限）' :
                '标准模式（点击切换计划模式）'
              }
            >
              <span
                className="material-symbols-outlined"
                style={{
                  fontSize: 14,
                  color: permissionMode === 'auto_approve' ? 'var(--ops-status-warn)' :
                         permissionMode === 'plan' ? 'var(--ops-status-info, #3b82f6)' :
                         'var(--ops-fg-muted)',
                }}
              >
                {permissionMode === 'auto_approve' ? 'lock_open' :
                 permissionMode === 'plan' ? 'assignment' : 'lock'}
              </span>
              <span
                style={{
                  fontSize: 11,
                  fontFamily: 'var(--ops-font-ui)',
                  color: permissionMode === 'auto_approve' ? 'var(--ops-status-warn)' :
                         permissionMode === 'plan' ? 'var(--ops-status-info, #3b82f6)' :
                         'var(--ops-fg-muted)',
                }}
              >
                {permissionMode === 'auto_approve' ? '全权限' :
                 permissionMode === 'plan' ? '计划模式' : '标准模式'}
              </span>
            </button>
            {/* Send button */}
            <button
              onClick={handleSend}
              disabled={effectiveDisabled || !text.trim()}
              style={{
                width: 28,
                height: 28,
                borderRadius: 6,
                border: 'none',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: text.trim() && !effectiveDisabled ? 'pointer' : 'not-allowed',
                background: text.trim() && !effectiveDisabled ? 'var(--ops-fg-primary)' : 'var(--ops-border-default)',
                opacity: text.trim() && !effectiveDisabled ? 1 : 0.5,
              }}
            >
              <span className="material-symbols-outlined" style={{ fontSize: 16, color: 'var(--ops-bg-canvas)' }}>
                arrow_upward
              </span>
            </button>
          </div>
        </div>
      </div>

      {/* Confirmation popover for permission mode */}
      {showModeConfirm && (
        <div
          style={{
            position: 'absolute',
            bottom: 60,
            right: 24,
            padding: '8px 12px',
            background: 'var(--ops-bg-elevated)',
            border: '1px solid var(--ops-border-default)',
            borderRadius: 6,
            zIndex: 200,
            width: 280,
          }}
        >
          <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-secondary)', marginBottom: 8 }}>
            全权限模式下所有操作将自动执行，不再弹出确认。
          </div>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button
              onClick={() => setShowModeConfirm(false)}
              style={{
                padding: '4px 12px',
                borderRadius: 4,
                border: '1px solid var(--ops-border-default)',
                background: 'transparent',
                color: 'var(--ops-fg-secondary)',
                fontFamily: 'var(--ops-font-ui)',
                fontSize: 11,
                cursor: 'pointer',
              }}
            >
              取消
            </button>
            <button
              onClick={() => {
                onPermissionModeChange?.('auto_approve')
                setShowModeConfirm(false)
              }}
              style={{
                padding: '4px 12px',
                borderRadius: 4,
                border: 'none',
                background: 'var(--ops-status-warn)',
                color: '#fff',
                fontFamily: 'var(--ops-font-ui)',
                fontSize: 11,
                cursor: 'pointer',
              }}
            >
              确认
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

// Small icon button for toolbar
const ToolbarBtn: FC<{ icon: string; title: string; onClick: () => void }> = ({ icon, title, onClick }) => (
  <button
    onClick={onClick}
    title={title}
    style={{
      width: 24,
      height: 24,
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

// Context usage ring indicator
const ContextRing: FC<{ percent: number }> = ({ percent }) => {
  const size = 18
  const stroke = 2.5
  const radius = (size - stroke) / 2
  const circumference = 2 * Math.PI * radius
  const filled = (percent / 100) * circumference
  const color = percent >= 90 ? 'var(--ops-status-danger)' : percent >= 70 ? 'var(--ops-status-warn)' : percent > 0 ? 'var(--ops-status-ok)' : 'var(--ops-fg-muted)'

  const budget = 19660
  const used = Math.round((percent / 100) * budget)

  return (
    <div
      title={`上下文窗口: ${used.toLocaleString()}/${budget.toLocaleString()} tokens (${percent}%)\n超过 90% 时会触发自动压缩`}
      style={{ display: 'flex', alignItems: 'center', gap: 3, cursor: 'default' }}
    >
      <svg width={size} height={size} style={{ transform: 'rotate(-90deg)' }}>
        <circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke="var(--ops-border-subtle)" strokeWidth={stroke} />
        {percent > 0 && (
          <circle
            cx={size / 2} cy={size / 2} r={radius} fill="none"
            stroke={color} strokeWidth={stroke}
            strokeDasharray={`${filled} ${circumference - filled}`}
            strokeLinecap="round"
          />
        )}
      </svg>
      <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 9, color, lineHeight: 1 }}>
        {percent > 0 ? `${percent}%` : 'ctx'}
      </span>
    </div>
  )
}
