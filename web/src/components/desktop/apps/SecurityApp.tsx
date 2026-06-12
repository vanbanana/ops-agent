import { authFetch } from '../../../lib/auth'
// Data: GET /api/v1/safety/scan
import { type FC, useState, useCallback } from 'react'

interface SecurityAppProps { connected: boolean }

export const SecurityApp: FC<SecurityAppProps> = ({ connected }) => {
  const [command, setCommand] = useState('')
  const [result, setResult] = useState<{ safe: boolean; reason: string } | null>(null)
  const [loading, setLoading] = useState(false)

  const handleScan = useCallback(async () => {
    if (!command.trim() || !connected) return
    setLoading(true)
    setResult(null)
    try {
      const res = await authFetch(`/api/v1/safety/scan?cmd=${encodeURIComponent(command)}`)
      if (res.ok) {
        const json = await res.json()
        setResult({ safe: json.safe ?? json.allowed ?? false, reason: json.reason ?? json.message ?? '未知' })
      } else {
        setResult({ safe: false, reason: '请求失败' })
      }
    } catch {
      setResult({ safe: false, reason: '网络错误' })
    } finally {
      setLoading(false)
    }
  }, [command, connected])

  return (
    <div style={{ height: '100%', padding: 16, display: 'flex', flexDirection: 'column', gap: 12, overflowY: 'auto' }}>
      <div style={{ fontSize: 12, color: 'var(--ops-fg-secondary)' }}>输入命令验证安全护栏拦截能力。</div>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleScan()}
          placeholder="例如: rm -rf /"
          style={{ flex: 1, height: 30, padding: '0 8px', fontSize: 12, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-primary)', background: 'var(--ops-bg-canvas)', border: '1px solid var(--ops-border-default)', borderRadius: 4, outline: 'none' }}
        />
        <button onClick={handleScan} disabled={loading || !command.trim()} style={{ height: 30, padding: '0 12px', fontSize: 12, color: 'var(--ops-fg-primary)', background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-default)', borderRadius: 4, cursor: 'pointer', opacity: loading ? 0.5 : 1 }}>
          {loading ? '...' : '扫描'}
        </button>
      </div>
      {result && (
        <div style={{ padding: 10, borderRadius: 4, borderLeft: `3px solid ${result.safe ? 'var(--ops-status-ok)' : 'var(--ops-status-danger)'}`, background: 'var(--ops-bg-canvas)' }}>
          <div style={{ fontSize: 12, fontWeight: 500, color: result.safe ? 'var(--ops-status-ok)' : 'var(--ops-status-danger)', marginBottom: 4 }}>
            {result.safe ? '✓ 命令安全' : '✗ 已拦截'}
          </div>
          <div style={{ fontSize: 12, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-secondary)', whiteSpace: 'pre-wrap' }}>{result.reason}</div>
        </div>
      )}
      <div style={{ borderTop: '1px solid var(--ops-border-subtle)', paddingTop: 10 }}>
        <div style={{ fontSize: 11, color: 'var(--ops-fg-muted)', marginBottom: 6 }}>快速测试:</div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {['rm -rf /', 'cat /etc/shadow', 'dd if=/dev/zero of=/dev/sda', 'chmod -R 777 /', 'df -h', 'ps aux'].map(cmd => (
            <button key={cmd} onClick={() => setCommand(cmd)} style={{ padding: '2px 6px', fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-secondary)', background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-subtle)', borderRadius: 3, cursor: 'pointer' }}>{cmd}</button>
          ))}
        </div>
      </div>
    </div>
  )
}
