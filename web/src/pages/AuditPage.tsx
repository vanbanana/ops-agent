import { authFetch } from '../lib/auth'
import { type FC, useState, useEffect } from 'react'

interface AuditEntry {
  id: number
  trace_id: string
  session_id: string
  round: number
  stage: string
  role?: string
  content: string
  triggered_by: string
  status: string
  duration_ms: number
  created_at: string
}

interface TraceGroup {
  traceId: string
  entries: AuditEntry[]
  startTime: string
  hasBlocked: boolean
}

const STAGE_ORDER = ['SENSE', 'ANALYZE', 'PLAN', 'EXECUTE', 'OUTPUT']
const STAGE_ICONS: Record<string, string> = {
  SENSE: 'sensors', ANALYZE: 'psychology', PLAN: 'route',
  EXECUTE: 'build', OUTPUT: 'chat',
}

export const AuditPage: FC = () => {
  const [logs, setLogs] = useState<AuditEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedTrace, setExpandedTrace] = useState<string | null>(null)

  useEffect(() => {
    authFetch('/api/v1/audit/logs')
      .then(r => r.json())
      .then(d => { if (d?.data) setLogs(d.data) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  // Group by trace_id
  const traces: TraceGroup[] = []
  const traceMap = new Map<string, AuditEntry[]>()
  for (const log of logs) {
    if (!traceMap.has(log.trace_id)) traceMap.set(log.trace_id, [])
    traceMap.get(log.trace_id)?.push(log)
  }
  for (const [traceId, entries] of traceMap) {
    // Sort entries by stage order
    entries.sort((a, b) => STAGE_ORDER.indexOf(a.stage) - STAGE_ORDER.indexOf(b.stage))
    traces.push({
      traceId,
      entries,
      startTime: entries[entries.length - 1]?.created_at || '',
      hasBlocked: entries.some(e => e.status === 'blocked'),
    })
  }

  return (
    <div style={{ flex: 1, overflow: 'auto', background: 'var(--ops-bg-canvas)', padding: '16px 24px' }}>
      <h2 style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 14, fontWeight: 600, color: 'var(--ops-fg-primary)', margin: '0 0 12px 0', display: 'flex', alignItems: 'center', gap: 6 }}>
        <span className="material-symbols-outlined" style={{ fontSize: 16 }}>description</span>
        推理链路审计
      </h2>
      <p style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', margin: '0 0 16px 0' }}>
        每次对话的完整链路: 接收 → 感知 → 推理 → 校验 → 执行 → 输出
      </p>

      {loading && <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>加载中...</div>}
      {!loading && traces.length === 0 && <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>暂无审计日志</div>}

      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {traces.map(trace => {
          const isExpanded = expandedTrace === trace.traceId
          const stagesPresent = new Set(trace.entries.map(e => e.stage))
          return (
            <div key={trace.traceId} style={{ border: '1px solid var(--ops-border-subtle)', borderRadius: 6, overflow: 'hidden', borderLeft: trace.hasBlocked ? '3px solid var(--ops-status-danger)' : '3px solid #34c759' }}>
              {/* Header — click to expand */}
              <button
                onClick={() => setExpandedTrace(isExpanded ? null : trace.traceId)}
                style={{ width: '100%', display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', border: 'none', background: 'var(--ops-bg-elevated)', cursor: 'pointer', textAlign: 'left' }}
              >
                <span className="material-symbols-outlined" style={{ fontSize: 12, color: 'var(--ops-fg-muted)' }}>
                  {isExpanded ? 'expand_more' : 'chevron_right'}
                </span>
                <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>
                  {formatTime(trace.startTime)}
                </span>
                {/* Stage pipeline visualization */}
                <div style={{ display: 'flex', gap: 2, flex: 1 }}>
                  {STAGE_ORDER.map(stage => (
                    <span
                      key={stage}
                      style={{
                        width: 18, height: 18, borderRadius: 3,
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        background: stagesPresent.has(stage)
                          ? (trace.entries.find(e => e.stage === stage)?.status === 'blocked' ? 'rgba(255,59,48,0.15)' : 'rgba(52,199,89,0.1)')
                          : 'var(--ops-bg-canvas)',
                      }}
                    >
                      <span className="material-symbols-outlined" style={{
                        fontSize: 11,
                        color: stagesPresent.has(stage)
                          ? (trace.entries.find(e => e.stage === stage)?.status === 'blocked' ? 'var(--ops-status-danger)' : '#34c759')
                          : 'var(--ops-fg-muted)',
                      }}>
                        {STAGE_ICONS[stage] || 'circle'}
                      </span>
                    </span>
                  ))}
                </div>
                <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 9, color: 'var(--ops-fg-muted)' }}>
                  {trace.entries.length} steps
                </span>
                <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 9, color: trace.hasBlocked ? 'var(--ops-status-danger)' : '#34c759' }}>
                  {trace.hasBlocked ? 'BLOCKED' : 'OK'}
                </span>
              </button>

              {/* Expanded detail */}
              {isExpanded && (
                <div style={{ padding: '4px 8px 8px 24px' }}>
                  {trace.entries.map((entry, i) => (
                    <div key={entry.id} style={{ display: 'flex', alignItems: 'flex-start', gap: 8, padding: '4px 0', borderBottom: i < trace.entries.length - 1 ? '1px solid var(--ops-border-subtle)' : 'none' }}>
                      <span className="material-symbols-outlined" style={{ fontSize: 13, color: statusColor(entry.status), marginTop: 1 }}>
                        {STAGE_ICONS[entry.stage] || 'circle'}
                      </span>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, fontWeight: 600, color: 'var(--ops-fg-primary)' }}>{entry.stage}</span>
                          {entry.role && <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 9, color: 'var(--ops-fg-muted)', background: 'var(--ops-bg-canvas)', padding: '1px 4px', borderRadius: 2 }}>{entry.role}</span>}
                          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 9, color: 'var(--ops-fg-muted)' }}>{entry.duration_ms}ms</span>
                          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 9, color: statusColor(entry.status) }}>{entry.status}</span>
                        </div>
                        <div style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 9, color: 'var(--ops-fg-muted)', marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '100%' }}>
                          {entry.content || '-'}
                        </div>
                      </div>
                    </div>
                  ))}
                  <div style={{ marginTop: 4, fontFamily: 'var(--ops-font-mono)', fontSize: 8, color: 'var(--ops-fg-muted)' }}>
                    trace: {trace.traceId.slice(0, 20)}...
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString('en-GB', { hour12: false }) + ' ' + d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
  } catch { return iso }
}

function statusColor(s: string): string {
  switch (s) {
    case 'ok': return '#34c759'
    case 'warning': return 'var(--ops-status-warn)'
    case 'blocked': case 'error': return 'var(--ops-status-danger)'
    default: return 'var(--ops-fg-muted)'
  }
}
