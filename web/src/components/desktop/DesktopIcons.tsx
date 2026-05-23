// Data: ResourceData (动态副标题)
import { type FC, useState } from 'react'
import type { ResourceData } from '../../types/api'
import type { DesktopApp } from '../../hooks/useWindowManager'

interface DesktopIconsProps {
  resources: ResourceData
  onOpenApp: (app: DesktopApp) => void
  onSwitchToChat: () => void
}

interface IconDef {
  app: DesktopApp | 'chat'
  icon: string
  label: string
  getSubtitle: (r: ResourceData) => string
}

const icons: IconDef[] = [
  { app: 'files',    icon: 'folder',           label: '文件管理器', getSubtitle: (r) => { const d = r.disk.find(x => x.mount === '/'); return d ? `/ ${d.percent}%` : '--' } },
  { app: 'trash',    icon: 'delete',           label: '回收站',     getSubtitle: () => '可清理文件' },
  { app: 'monitor',  icon: 'monitoring',       label: '系统监控',   getSubtitle: (r) => r.load > 0 ? `load ${r.load.toFixed(1)}` : '--' },
  { app: 'process',  icon: 'memory',           label: '进程管理',   getSubtitle: (r) => r.processes > 0 ? `${r.processes} 进程` : '--' },
  { app: 'network',  icon: 'lan',              label: '网络',       getSubtitle: (r) => r.ports > 0 ? `${r.ports} 端口` : '--' },
  { app: 'logs',     icon: 'article',          label: '日志',       getSubtitle: () => '系统日志' },
  { app: 'services', icon: 'settings_suggest', label: '服务管理',   getSubtitle: () => 'systemctl' },
  { app: 'terminal', icon: 'terminal',         label: '终端',       getSubtitle: () => 'shell' },
  { app: 'security', icon: 'shield',           label: '安全中心',   getSubtitle: () => '安全演练' },
  { app: 'chat',     icon: 'smart_toy',        label: '对话助手',   getSubtitle: () => 'AI 对话' },
]

export const DesktopIcons: FC<DesktopIconsProps> = ({ resources, onOpenApp, onSwitchToChat }) => {
  return (
    <div style={{ position: 'absolute', top: 16, left: 16, display: 'grid', gridTemplateRows: 'repeat(5, 88px)', gridTemplateColumns: 'repeat(2, 88px)', gap: 8 }}>
      {icons.map((def) => (
        <DesktopIcon
          key={def.app}
          icon={def.icon}
          label={def.label}
          subtitle={def.getSubtitle(resources)}
          onDoubleClick={() => {
            if (def.app === 'chat') onSwitchToChat()
            else onOpenApp(def.app as DesktopApp)
          }}
        />
      ))}
    </div>
  )
}

const DesktopIcon: FC<{
  icon: string
  label: string
  subtitle: string
  onDoubleClick: () => void
}> = ({ icon, label, subtitle, onDoubleClick }) => {
  const [selected, setSelected] = useState(false)

  return (
    <div
      onDoubleClick={onDoubleClick}
      onClick={() => setSelected(true)}
      onBlur={() => setSelected(false)}
      tabIndex={0}
      style={{
        width: 88,
        height: 88,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 4,
        borderRadius: 4,
        cursor: 'pointer',
        background: selected ? 'rgba(55,148,253,0.12)' : 'transparent',
        border: selected ? '1px solid rgba(55,148,253,0.4)' : '1px solid transparent',
        outline: 'none',
        userSelect: 'none',
      }}
    >
      <span className="material-symbols-outlined" style={{ fontSize: 32, color: 'var(--ops-fg-primary)' }}>
        {icon}
      </span>
      <span style={{ fontSize: 11, color: 'var(--ops-fg-primary)', fontFamily: 'var(--ops-font-ui)', textAlign: 'center', lineHeight: '14px', maxWidth: 80, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {label}
      </span>
      <span style={{ fontSize: 10, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)', lineHeight: '12px' }}>
        {subtitle}
      </span>
    </div>
  )
}
