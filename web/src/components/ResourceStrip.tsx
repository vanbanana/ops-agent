// Data: SSE execute_done (probe results)
import { type FC } from 'react'
import type { ResourceData } from '../types/api'

interface ResourceStripProps {
  resources: ResourceData
}

export const ResourceStrip: FC<ResourceStripProps> = ({ resources }) => {
  const { disk, load, memory, processes, ports } = resources
  const rootDisk = disk.find((d) => d.mount === '/') || disk[0]
  const varDisk = disk.find((d) => d.mount === '/var')

  const items: Array<{ text: string; color: string }> = []

  items.push({
    text: rootDisk ? `/  ${rootDisk.percent}%` : '/  --',
    color: rootDisk && rootDisk.percent >= 80 ? 'var(--ops-status-warn)' : 'var(--ops-fg-secondary)',
  })

  if (varDisk) {
    items.push({ text: `/var ${varDisk.percent}%`, color: 'var(--ops-fg-secondary)' })
  }

  items.push({
    text: load >= 0 ? `load ${load.toFixed(2)}` : 'load --',
    color: load >= 2 ? 'var(--ops-status-warn)' : 'var(--ops-fg-secondary)',
  })

  items.push({
    text: memory.percent >= 0 ? `mem ${memory.used}/${memory.total}` : 'mem --',
    color: 'var(--ops-fg-secondary)',
  })

  items.push({
    text: processes >= 0 ? `${processes} proc` : '-- proc',
    color: 'var(--ops-fg-secondary)',
  })

  items.push({
    text: ports >= 0 ? `${ports} port` : '-- port',
    color: 'var(--ops-fg-secondary)',
  })

  return (
    <div
      style={{
        height: 26,
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 6,
        padding: '0 12px',
        background: 'var(--ops-bg-canvas)',
        borderBottom: '1px solid var(--ops-border-subtle)',
        flexShrink: 0,
      }}
    >
      {items.map((item, i) => (
        <span key={i} style={{ display: 'flex', alignItems: 'center' }}>
          {i > 0 && (
            <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-muted)', margin: '0 3px' }}>|</span>
          )}
          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: item.color }}>{item.text}</span>
        </span>
      ))}
    </div>
  )
}
