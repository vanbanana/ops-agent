// Data: None (static navigation)
import { type FC } from 'react'

interface SideNavProps {
  active: string
  onNavigate: (page: string) => void
}

const topItems = [
  { key: 'agent', icon: 'smart_toy', label: 'Agent' },
  { key: 'terminal', icon: 'terminal', label: '终端' },
  { key: 'files', icon: 'folder', label: '文件' },
  { key: 'audit', icon: 'description', label: '审计' },
]

const bottomItems = [
  { key: 'theme', icon: 'dark_mode', label: '主题' },
  { key: 'account', icon: 'account_circle', label: '账号' },
  { key: 'settings', icon: 'settings', label: '设置' },
]

export const SideNav: FC<SideNavProps> = ({ active, onNavigate }) => {
  return (
    <nav
      style={{
        width: 44,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        padding: '8px 0',
        background: 'var(--ops-bg-sidebar)',
        borderRight: '1px solid var(--ops-border-subtle)',
        flexShrink: 0,
        gap: 2,
      }}
    >
      {topItems.map((item) => (
        <NavBtn
          key={item.key}
          icon={item.icon}
          active={active === item.key}
          onClick={() => onNavigate(item.key)}
          title={item.label}
        />
      ))}

      <div style={{ flex: 1 }} />

      {bottomItems.map((item) => (
        <NavBtn
          key={item.key}
          icon={item.icon}
          active={active === item.key}
          onClick={() => onNavigate(item.key)}
          title={item.label}
        />
      ))}
    </nav>
  )
}

const NavBtn: FC<{ icon: string; active: boolean; onClick: () => void; title: string }> = ({ icon, active, onClick, title }) => (
  <button
    onClick={onClick}
    title={title}
    style={{
      width: 36,
      height: 36,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      borderRadius: 4,
      border: 'none',
      cursor: 'pointer',
      background: active ? 'var(--ops-bg-elevated)' : 'transparent',
      color: active ? 'var(--ops-fg-primary)' : 'var(--ops-fg-muted)',
    }}
  >
    <span className="material-symbols-outlined" style={{ fontSize: 20 }}>{icon}</span>
  </button>
)
