// Data: POST /api/v1/desktop/probe/* (30s轮询) + SSE execute_done
// 对标: 阿里云 Workbench 右侧 "系统监控" 面板
// 图表: CPU使用率 | 内存使用量 | 磁盘 | 网络流量
import { type FC, useEffect, useRef, useState } from 'react'
import type { ResourceData } from '../types/api'

interface MonitorPanelProps {
  resources: ResourceData
}

// Store history for charts (last 20 data points)
interface DataPoint {
  time: string
  value: number
}

export const MonitorPanel: FC<MonitorPanelProps> = ({ resources }) => {
  const [cpuHistory, setCpuHistory] = useState<DataPoint[]>([])
  const [memHistory, setMemHistory] = useState<DataPoint[]>([])

  // Accumulate history on resource updates
  useEffect(() => {
    if (resources.load >= 0) {
      const now = new Date().toLocaleTimeString('en-GB', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
      setCpuHistory(prev => [...prev.slice(-19), { time: now, value: resources.load }])
    }
    if (resources.memory.percent >= 0) {
      const now = new Date().toLocaleTimeString('en-GB', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
      setMemHistory(prev => [...prev.slice(-19), { time: now, value: resources.memory.percent }])
    }
  }, [resources.load, resources.memory.percent])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
      {/* CPU / Load */}
      <MetricSection title="CPU / 负载">
        <InfoBar left={`负载 ${resources.load >= 0 ? resources.load.toFixed(2) : '--'}`} right={resources.load >= 4 ? '偏高' : resources.load >= 0 ? '正常' : ''} />
        <MiniChart data={cpuHistory} color="#7B68EE" maxY={10} unit="" />
      </MetricSection>

      {/* Memory */}
      <MetricSection title="内存使用量（MB）">
        <InfoBar
          left={`总计 ${resources.memory.total || '--'}`}
          right={`使用率 ${resources.memory.percent >= 0 ? resources.memory.percent + '%' : '--'}`}
        />
        <InfoBar
          left={`已用 ${resources.memory.used || '--'}`}
          right={resources.memory.percent >= 90 ? '警告: 内存不足' : resources.memory.percent >= 75 ? '偏高' : '正常'}
        />
        <MiniChart data={memHistory} color="#00CED1" maxY={100} unit="%" />
      </MetricSection>

      {/* Disk */}
      <MetricSection title="磁盘使用率">
        {resources.disk.length === 0 ? (
          <EmptyState />
        ) : (
          <div style={{ padding: '8px 0' }}>
            {resources.disk.map((d) => (
              <DiskRow key={d.mount} mount={d.mount} percent={d.percent} used={d.used} total={d.total} />
            ))}
          </div>
        )}
      </MetricSection>

      {/* Processes */}
      <MetricSection title="进程">
        <InfoBar
          left={`活跃 ${resources.processes >= 0 ? resources.processes : '--'}`}
          right={`端口 ${resources.ports >= 0 ? resources.ports : '--'}`}
        />
      </MetricSection>
    </div>
  )
}

// Section wrapper — mimics Alibaba's card style
const MetricSection: FC<{ title: string; children: React.ReactNode }> = ({ title, children }) => (
  <div style={{ borderBottom: '1px solid var(--ops-border-subtle)', padding: '12px 0' }}>
    <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 13, fontWeight: 600, color: 'var(--ops-fg-primary)', marginBottom: 6 }}>
      {title}
    </div>
    {children}
  </div>
)

// Info bar — gray background row with two values
const InfoBar: FC<{ left: string; right: string }> = ({ left, right }) => (
  <div
    style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      padding: '4px 10px',
      background: 'var(--ops-bg-elevated)',
      borderRadius: 2,
      marginBottom: 6,
    }}
  >
    <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-secondary)' }}>{left}</span>
    <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-secondary)' }}>{right}</span>
  </div>
)

// Mini sparkline chart using canvas — matches Alibaba's purple/cyan line style
const MiniChart: FC<{ data: DataPoint[]; color: string; maxY: number; unit: string }> = ({ data, color, maxY }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || data.length < 1) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const dpr = window.devicePixelRatio || 1
    const w = canvas.clientWidth
    const h = canvas.clientHeight
    canvas.width = w * dpr
    canvas.height = h * dpr
    ctx.scale(dpr, dpr)

    // Clear
    ctx.clearRect(0, 0, w, h)

    // Draw grid lines
    ctx.strokeStyle = '#2B2B2B'
    ctx.lineWidth = 0.5
    for (let i = 0; i <= 4; i++) {
      const y = (h / 4) * i
      ctx.beginPath()
      ctx.moveTo(0, y)
      ctx.lineTo(w, y)
      ctx.stroke()
    }

    // Draw line
    const actualMax = Math.max(maxY, ...data.map(d => d.value)) * 1.1
    ctx.strokeStyle = color
    ctx.lineWidth = 1.5
    ctx.lineJoin = 'round'
    ctx.beginPath()

    data.forEach((point, i) => {
      const x = data.length === 1 ? w / 2 : (i / (data.length - 1)) * w
      const y = h - (point.value / actualMax) * h
      if (i === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    })

    // For single point, draw a small horizontal line
    if (data.length === 1) {
      const y = h - (data[0].value / actualMax) * h
      ctx.moveTo(0, y)
      ctx.lineTo(w, y)
    }

    ctx.stroke()

    // Draw fill gradient
    if (data.length > 1) {
      ctx.lineTo(w, h)
      ctx.lineTo(0, h)
      ctx.closePath()
      const gradient = ctx.createLinearGradient(0, 0, 0, h)
      gradient.addColorStop(0, color + '20')
      gradient.addColorStop(1, 'transparent')
      ctx.fillStyle = gradient
      ctx.fill()
    }

    // X-axis labels
    ctx.fillStyle = '#707070'
    ctx.font = '9px monospace'
    if (data.length > 0) {
      ctx.fillText(data[0].time, 2, h - 2)
      ctx.textAlign = 'end'
      ctx.fillText(data[data.length - 1].time, w - 2, h - 2)
    }
  }, [data, color, maxY])

  if (data.length < 1) {
    return <EmptyState />
  }

  return (
    <canvas
      ref={canvasRef}
      style={{ width: '100%', height: 120, display: 'block' }}
    />
  )
}

// Disk row with progress bar
const DiskRow: FC<{ mount: string; percent: number; used: string; total: string }> = ({ mount, percent, used, total }) => {
  const color = percent >= 90 ? 'var(--ops-status-danger)' : percent >= 75 ? 'var(--ops-status-warn)' : '#7B68EE'
  return (
    <div style={{ marginBottom: 6 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
        <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-secondary)' }}>{mount}</span>
        <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-secondary)' }}>{used}/{total} ({percent}%)</span>
      </div>
      <div style={{ height: 3, background: 'var(--ops-bg-canvas)', borderRadius: 1, overflow: 'hidden' }}>
        <div style={{ height: '100%', width: `${percent}%`, background: color, borderRadius: 1 }} />
      </div>
    </div>
  )
}

// Empty state placeholder
const EmptyState: FC = () => (
  <div style={{ padding: '16px 0', textAlign: 'center' }}>
    <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
      暂无数据
    </span>
  </div>
)
