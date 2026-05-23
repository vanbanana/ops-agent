// Data: GET /health (每30s轮询)
// 端点状态: ✅ 已就绪
import { type FC } from 'react'
import type { HealthResponse } from '../types/api'

interface HealthPanelProps {
  health: HealthResponse | null
  loading: boolean
}

export const HealthPanel: FC<HealthPanelProps> = ({ health, loading }) => {
  if (loading) {
    return (
      <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>加载中...</span>
      </div>
    )
  }

  if (!health) {
    return (
      <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-status-danger)' }}>无法连接后端</span>
      </div>
    )
  }

  const statusColor = health.status === 'healthy' ? 'var(--ops-status-ok)' : health.status === 'degraded' ? 'var(--ops-status-warn)' : 'var(--ops-status-danger)'
  const statusText = health.status === 'healthy' ? '健康' : health.status === 'degraded' ? '降级' : '异常'

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Overall */}
      <div>
        <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 500, color: 'var(--ops-fg-muted)', marginBottom: 6, textTransform: 'uppercase' }}>
          总体状态
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ width: 8, height: 8, borderRadius: '50%', background: statusColor }} />
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 13, fontWeight: 500, color: 'var(--ops-fg-primary)' }}>{statusText}</span>
        </div>
      </div>

      {/* Components */}
      <div>
        <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 500, color: 'var(--ops-fg-muted)', marginBottom: 6, textTransform: 'uppercase' }}>
          子组件
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {Object.entries(health.components).map(([name, comp]) => {
            const dotColor = comp.status === 'up' ? 'var(--ops-status-ok)' : comp.status === 'down' ? 'var(--ops-status-danger)' : 'var(--ops-status-warn)'
            const label = comp.status === 'up' ? '正常' : comp.status === 'down' ? '异常' : '降级'
            return (
              <div
                key={name}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '6px 8px',
                  borderRadius: 3,
                  background: 'var(--ops-bg-elevated)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ width: 6, height: 6, borderRadius: '50%', background: dotColor }} />
                  <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-primary)' }}>{name}</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  {comp.latency_ms !== undefined && (
                    <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>{comp.latency_ms}ms</span>
                  )}
                  <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, color: 'var(--ops-fg-secondary)' }}>{label}</span>
                </div>
              </div>
            )
          })}

          {Object.keys(health.components).length === 0 && (
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>无组件信息</span>
          )}
        </div>
      </div>
    </div>
  )
}
