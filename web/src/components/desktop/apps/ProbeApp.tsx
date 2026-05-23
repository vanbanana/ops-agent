// Data: POST /api/v1/desktop/probe/{name}
// 通用探针数据展示组件 — 多个窗口应用复用
import { type FC, useEffect, useState, useCallback } from 'react'

interface ProbeAppProps {
  probeName: string
  connected: boolean
  title?: string
}

export const ProbeApp: FC<ProbeAppProps> = ({ probeName, connected, title }) => {
  const [output, setOutput] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    if (!connected) { setLoading(false); setOutput('未连接后端'); return }
    setLoading(true)
    try {
      const res = await fetch(`/api/v1/desktop/probe/${probeName}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
      })
      if (res.ok) {
        const json = await res.json()
        setOutput(json.data?.result ?? json.data?.summary ?? '无数据')
      } else {
        setOutput(`请求失败 (${res.status})`)
      }
    } catch (e) {
      setOutput(`网络错误: ${(e as Error).message}`)
    } finally {
      setLoading(false)
    }
  }, [probeName, connected])

  useEffect(() => { fetchData() }, [fetchData])

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Optional title + refresh */}
      <div style={{ display: 'flex', alignItems: 'center', padding: '8px 12px', gap: 8, borderBottom: '1px solid var(--ops-border-subtle)', flexShrink: 0 }}>
        {title && <span style={{ fontSize: 12, color: 'var(--ops-fg-secondary)', fontWeight: 500 }}>{title}</span>}
        <div style={{ flex: 1 }} />
        <button
          onClick={fetchData}
          style={{ width: 24, height: 24, display: 'flex', alignItems: 'center', justifyContent: 'center', border: 'none', borderRadius: 4, cursor: 'pointer', background: 'transparent', color: 'var(--ops-fg-muted)' }}
        >
          <span className="material-symbols-outlined" style={{ fontSize: 14 }}>refresh</span>
        </button>
      </div>
      {/* Content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 12 }}>
        {loading ? (
          <span style={{ fontSize: 12, color: 'var(--ops-fg-muted)' }}>加载中...</span>
        ) : (
          <pre style={{ fontSize: 12, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-primary)', whiteSpace: 'pre-wrap', margin: 0, lineHeight: '18px' }}>
            {output}
          </pre>
        )}
      </div>
    </div>
  )
}
