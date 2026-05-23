// Data: ResourceData (from polling)
import { type FC } from 'react'
import type { ResourceData } from '../../../types/api'

interface MonitorAppProps {
  resources: ResourceData
  connected: boolean
}

export const MonitorApp: FC<MonitorAppProps> = ({ resources, connected }) => {
  return (
    <div style={{ height: '100%', overflowY: 'auto', padding: 16, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
      <MetricCard
        title="系统负载"
        value={resources.load > 0 ? resources.load.toFixed(2) : '--'}
        unit="load avg"
        percent={Math.min(resources.load * 25, 100)}
        color={resources.load > 4 ? 'var(--ops-status-danger)' : resources.load > 2 ? 'var(--ops-status-warn)' : 'var(--ops-status-ok)'}
      />
      <MetricCard
        title="内存"
        value={resources.memory.percent > 0 ? `${resources.memory.percent}%` : '--'}
        unit={resources.memory.used && resources.memory.total ? `${resources.memory.used} / ${resources.memory.total}` : ''}
        percent={resources.memory.percent}
        color={resources.memory.percent > 90 ? 'var(--ops-status-danger)' : resources.memory.percent > 70 ? 'var(--ops-status-warn)' : 'var(--ops-status-ok)'}
      />
      <div style={{ gridColumn: '1 / -1' }}>
        <div style={{ fontSize: 12, color: 'var(--ops-fg-secondary)', fontWeight: 500, marginBottom: 8 }}>磁盘</div>
        {resources.disk.length > 0 ? resources.disk.map(d => (
          <DiskBar key={d.mount} mount={d.mount} percent={d.percent} used={d.used} total={d.total} />
        )) : (
          <span style={{ fontSize: 12, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)' }}>
            {connected ? '加载中...' : '--'}
          </span>
        )}
      </div>
      <MetricCard title="进程" value={resources.processes > 0 ? String(resources.processes) : '--'} unit="活跃" percent={0} color="var(--ops-fg-secondary)" showBar={false} />
      <MetricCard title="端口" value={resources.ports > 0 ? String(resources.ports) : '--'} unit="监听" percent={0} color="var(--ops-fg-secondary)" showBar={false} />
    </div>
  )
}

const MetricCard: FC<{ title: string; value: string; unit: string; percent: number; color: string; showBar?: boolean }> = ({ title, value, unit, percent, color, showBar = true }) => (
  <div style={{ padding: 12, border: '1px solid var(--ops-border-subtle)', borderRadius: 4 }}>
    <div style={{ fontSize: 11, color: 'var(--ops-fg-muted)', marginBottom: 6 }}>{title}</div>
    <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
      <span style={{ fontSize: 22, fontWeight: 600, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-primary)' }}>{value}</span>
      <span style={{ fontSize: 11, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)' }}>{unit}</span>
    </div>
    {showBar && <div style={{ height: 4, background: 'var(--ops-bg-canvas)', borderRadius: 2, marginTop: 8, overflow: 'hidden' }}><div style={{ height: '100%', width: `${Math.min(percent, 100)}%`, background: color, borderRadius: 2 }} /></div>}
  </div>
)

const DiskBar: FC<{ mount: string; percent: number; used: string; total: string }> = ({ mount, percent, used, total }) => {
  const color = percent > 95 ? 'var(--ops-status-danger)' : percent > 80 ? 'var(--ops-status-warn)' : 'var(--ops-status-ok)'
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
      <span style={{ width: 56, fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-secondary)' }}>{mount}</span>
      <div style={{ flex: 1, height: 6, background: 'var(--ops-bg-canvas)', borderRadius: 3, overflow: 'hidden' }}>
        <div style={{ height: '100%', width: `${percent}%`, background: color, borderRadius: 3 }} />
      </div>
      <span style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)', width: 32, textAlign: 'right' }}>{percent}%</span>
      <span style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)' }}>{used}/{total}</span>
    </div>
  )
}
