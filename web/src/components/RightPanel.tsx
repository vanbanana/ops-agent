// Data: SSE events (推理链路) + POST /desktop/probe/* (系统监控)
// 右侧面板: 系统监控(默认) | 推理链路
import { type FC, useState } from 'react'
import { ReasoningPanel } from './ReasoningPanel'
import { MonitorPanel } from './MonitorPanel'
import type { ReasoningStep, ResourceData, HealthResponse } from '../types/api'

interface RightPanelProps {
  reasoning: ReasoningStep[]
  resources: ResourceData
  health: HealthResponse | null
  healthLoading: boolean
  isStreaming: boolean
  onClose: () => void
}

type Tab = 'monitor' | 'reasoning'

export const RightPanel: FC<RightPanelProps> = ({
  reasoning,
  resources,
  isStreaming,
  onClose,
}) => {
  const [activeTab, setActiveTab] = useState<Tab>('monitor')

  const tabs: Array<{ key: Tab; label: string }> = [
    { key: 'monitor', label: '系统监控' },
    { key: 'reasoning', label: '推理链路' },
  ]

  return (
    <aside
      style={{
        width: 256,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--ops-bg-surface)',
        borderLeft: '1px solid var(--ops-border-subtle)',
        flexShrink: 0,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          height: 36,
          padding: '0 10px',
          borderBottom: '1px solid var(--ops-border-subtle)',
          flexShrink: 0,
          gap: 2,
        }}
      >
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            style={{
              height: 24,
              padding: '0 10px',
              borderRadius: 4,
              border: 'none',
              cursor: 'pointer',
              fontFamily: 'var(--ops-font-ui)',
              fontSize: 11,
              fontWeight: activeTab === tab.key ? 500 : 400,
              color: activeTab === tab.key ? 'var(--ops-fg-primary)' : 'var(--ops-fg-muted)',
              background: activeTab === tab.key ? 'var(--ops-bg-elevated)' : 'transparent',
            }}
          >
            {tab.label}
          </button>
        ))}
        <div style={{ flex: 1 }} />
        <button style={{ width: 20, height: 20, display: 'flex', alignItems: 'center', justifyContent: 'center', border: 'none', background: 'transparent', cursor: 'pointer', borderRadius: 3, color: 'var(--ops-fg-muted)' }}>
          <span className="material-symbols-outlined" style={{ fontSize: 14 }}>refresh</span>
        </button>
        <button onClick={onClose} style={{ width: 20, height: 20, display: 'flex', alignItems: 'center', justifyContent: 'center', border: 'none', background: 'transparent', cursor: 'pointer', borderRadius: 3, color: 'var(--ops-fg-muted)' }}>
          <span className="material-symbols-outlined" style={{ fontSize: 14 }}>close</span>
        </button>
      </div>

      {/* Content */}
      {activeTab === 'monitor' && (
        <div style={{ flex: 1, overflow: 'auto', padding: '0 12px' }}>
          <MonitorPanel resources={resources} />
        </div>
      )}

      {activeTab === 'reasoning' && (
        <div style={{ flex: 1, overflow: 'auto', padding: '10px 12px' }}>
          <ReasoningPanel steps={reasoning} isStreaming={isStreaming} />
        </div>
      )}
    </aside>
  )
}
