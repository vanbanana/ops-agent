import { type FC, useState } from 'react'
import { PermissionSettings } from './PermissionSettings'

type SettingsTab = 'permission' | 'general' | 'llm' | 'about'

const TABS: { id: SettingsTab; icon: string; label: string }[] = [
  { id: 'permission', icon: 'shield', label: '权限' },
  { id: 'general', icon: 'tune', label: '通用' },
  { id: 'llm', icon: 'smart_toy', label: '模型' },
  { id: 'about', icon: 'info', label: '关于' },
]

export const SettingsPage: FC = () => {
  const [activeTab, setActiveTab] = useState<SettingsTab>('permission')

  return (
    <div style={{ flex: 1, display: 'flex', overflow: 'hidden', background: 'var(--ops-bg-canvas)' }}>
      {/* Settings sidebar */}
      <nav
        style={{
          width: 160,
          borderRight: '1px solid var(--ops-border-subtle)',
          background: 'var(--ops-bg-surface)',
          padding: '16px 0',
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
          flexShrink: 0,
        }}
      >
        <div style={{ padding: '0 16px 12px', fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 600, color: 'var(--ops-fg-muted)', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
          设置
        </div>
        {TABS.map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '8px 16px',
              background: activeTab === tab.id ? 'var(--ops-bg-elevated)' : 'transparent',
              border: 'none',
              borderLeft: activeTab === tab.id ? '2px solid var(--ops-fg-primary)' : '2px solid transparent',
              cursor: 'pointer',
              width: '100%',
              textAlign: 'left',
            }}
          >
            <span
              className="material-symbols-outlined"
              style={{ fontSize: 16, color: activeTab === tab.id ? 'var(--ops-fg-primary)' : 'var(--ops-fg-muted)' }}
            >
              {tab.icon}
            </span>
            <span
              style={{
                fontFamily: 'var(--ops-font-ui)',
                fontSize: 12,
                color: activeTab === tab.id ? 'var(--ops-fg-primary)' : 'var(--ops-fg-secondary)',
                fontWeight: activeTab === tab.id ? 500 : 400,
              }}
            >
              {tab.label}
            </span>
          </button>
        ))}
      </nav>

      {/* Settings content */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {activeTab === 'permission' && <PermissionSettings />}
        {activeTab === 'general' && <SettingsPlaceholder icon="tune" title="通用设置" description="主题、语言、快捷键等通用配置" />}
        {activeTab === 'llm' && <SettingsPlaceholder icon="smart_toy" title="模型设置" description="LLM 模型选择、API 配置、Token 限制等" />}
        {activeTab === 'about' && <SettingsPlaceholder icon="info" title="关于" description="版本信息、开源协议、更新检查" />}
      </div>
    </div>
  )
}

const SettingsPlaceholder: FC<{ icon: string; title: string; description: string }> = ({ icon, title, description }) => (
  <div
    style={{
      flex: 1,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      gap: 12,
      padding: 48,
      minHeight: 400,
    }}
  >
    <span
      className="material-symbols-outlined"
      style={{ fontSize: 36, color: 'var(--ops-fg-muted)', opacity: 0.5 }}
    >
      {icon}
    </span>
    <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 14, fontWeight: 500, color: 'var(--ops-fg-secondary)' }}>
      {title}
    </span>
    <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>
      {description}
    </span>
  </div>
)
