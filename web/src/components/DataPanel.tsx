// Data: SSE execute_done events (probe_disk, probe_memory, probe_top, probe_process)
// 规则: Agent未调用过探针时显示空态"暂无数据"，调用后用最近一次真实结果渲染
import { type FC } from 'react'
import type { ResourceData } from '../types/api'

interface DataPanelProps {
  resources: ResourceData
}

export const DataPanel: FC<DataPanelProps> = ({ resources }) => {
  const hasDisk = resources.disk.length > 0
  const hasMemory = resources.memory.percent >= 0
  const hasLoad = resources.load >= 0

  if (!hasDisk && !hasMemory && !hasLoad) {
    return (
      <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
          暂无数据，发起对话后自动填充
        </span>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Disk */}
      {hasDisk && (
        <Section title="磁盘">
          {resources.disk.map((d) => {
            const color = d.percent >= 95 ? 'var(--ops-status-danger)' : d.percent >= 80 ? 'var(--ops-status-warn)' : 'var(--ops-status-ok)'
            return (
              <div key={d.mount} style={{ marginBottom: 8 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                  <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-secondary)' }}>{d.mount}</span>
                  <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color }}>{d.percent}%</span>
                </div>
                <div style={{ height: 4, borderRadius: 2, background: 'var(--ops-bg-canvas)', overflow: 'hidden' }}>
                  <div style={{ height: '100%', width: `${d.percent}%`, background: color, borderRadius: 2 }} />
                </div>
                <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>{d.used} / {d.total}</span>
              </div>
            )
          })}
        </Section>
      )}

      {/* Memory */}
      {hasMemory && (
        <Section title="内存">
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
            <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 18, fontWeight: 600, color: 'var(--ops-fg-primary)' }}>{resources.memory.percent}%</span>
            <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>{resources.memory.used} / {resources.memory.total}</span>
          </div>
          <div style={{ height: 4, borderRadius: 2, background: 'var(--ops-bg-canvas)', overflow: 'hidden', marginTop: 4 }}>
            <div style={{ height: '100%', width: `${resources.memory.percent}%`, background: resources.memory.percent >= 90 ? 'var(--ops-status-danger)' : resources.memory.percent >= 70 ? 'var(--ops-status-warn)' : 'var(--ops-status-ok)', borderRadius: 2 }} />
          </div>
        </Section>
      )}

      {/* Load */}
      {hasLoad && (
        <Section title="系统负载">
          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 18, fontWeight: 600, color: 'var(--ops-fg-primary)' }}>{resources.load.toFixed(2)}</span>
        </Section>
      )}

      {/* Processes */}
      {resources.processes >= 0 && (
        <Section title="进程数">
          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 18, fontWeight: 600, color: 'var(--ops-fg-primary)' }}>{resources.processes}</span>
        </Section>
      )}
    </div>
  )
}

const Section: FC<{ title: string; children: React.ReactNode }> = ({ title, children }) => (
  <div>
    <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 500, color: 'var(--ops-fg-muted)', marginBottom: 6, textTransform: 'uppercase' }}>{title}</div>
    {children}
  </div>
)
