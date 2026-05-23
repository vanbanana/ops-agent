import type { FC } from 'react'

interface SuggestedPromptsProps {
  onSelect: (prompt: string) => void
}

const PROMPTS = [
  { icon: 'monitor_heart', text: '系统健康检查', description: '全面扫描 CPU、内存、磁盘、网络' },
  { icon: 'storage', text: '磁盘空间分析', description: '找出占用空间最大的目录' },
  { icon: 'memory', text: '高内存进程排查', description: '列出内存占用 Top 10 进程' },
  { icon: 'security', text: '安全审计快检', description: '检查异常登录和可疑进程' },
] as const

export const SuggestedPrompts: FC<SuggestedPromptsProps> = ({ onSelect }) => (
  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, maxWidth: 480, width: '100%' }}>
    {PROMPTS.map((p) => (
      <div
        key={p.text}
        onClick={() => onSelect(p.text)}
        style={{
          border: '1px solid var(--ops-border-subtle)',
          borderRadius: 8,
          padding: '12px 16px',
          cursor: 'pointer',
          transition: 'border-color 150ms, background 150ms',
          background: 'transparent',
        }}
        onMouseEnter={(e) => { e.currentTarget.style.borderColor = 'var(--ops-border-default)'; e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
        onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'var(--ops-border-subtle)'; e.currentTarget.style.background = 'transparent' }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
          <span className="material-symbols-outlined" style={{ fontSize: 16, color: 'var(--ops-status-info)' }}>
            {p.icon}
          </span>
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, fontWeight: 500, color: 'var(--ops-fg-primary)' }}>
            {p.text}
          </span>
        </div>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', lineHeight: 1.4 }}>
          {p.description}
        </span>
      </div>
    ))}
  </div>
)
