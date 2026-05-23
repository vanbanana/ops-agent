// Data: POST /api/v1/safety/preview, POST /api/v1/safety/confirm
import { type FC, useState, useEffect, useCallback } from 'react'

interface RiskPreviewModalProps {
  command: string
  description: string
  onConfirm: () => void
  onCancel: () => void
}

interface PreviewData {
  preview_id: string
  command: string
  description: string
  risk: string
  expires_at: string
}

export const RiskPreviewModal: FC<RiskPreviewModalProps> = ({ command, description, onConfirm, onCancel }) => {
  const [preview, setPreview] = useState<PreviewData | null>(null)
  const [loading, setLoading] = useState(true)
  const [confirming, setConfirming] = useState(false)
  const [countdown, setCountdown] = useState(3)
  const [error, setError] = useState<string | null>(null)

  // Step 1: Create preview
  useEffect(() => {
    const createPreview = async () => {
      try {
        const res = await fetch('/api/v1/safety/preview', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ command, description }),
        })
        if (res.ok) {
          const json = await res.json()
          setPreview(json.data)
        } else {
          setError('创建预演失败')
        }
      } catch {
        setError('网络错误')
      } finally {
        setLoading(false)
      }
    }
    createPreview()
  }, [command, description])

  // Countdown before confirm is enabled
  useEffect(() => {
    if (!preview || countdown <= 0) return
    const timer = setTimeout(() => setCountdown(c => c - 1), 1000)
    return () => clearTimeout(timer)
  }, [preview, countdown])

  // Step 2: Confirm or cancel
  const handleConfirm = useCallback(async () => {
    if (!preview || countdown > 0) return
    setConfirming(true)
    try {
      const res = await fetch('/api/v1/safety/confirm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ preview_id: preview.preview_id, confirmed: true }),
      })
      if (res.ok) {
        onConfirm()
      } else {
        const json = await res.json()
        setError(json.error_code === 'PREVIEW_EXPIRED_001' ? '预演已过期，请重新操作' : json.error ?? '确认失败')
      }
    } catch {
      setError('网络错误')
    } finally {
      setConfirming(false)
    }
  }, [preview, countdown, onConfirm])

  const handleCancel = useCallback(async () => {
    if (preview) {
      // Notify backend of cancellation (best effort)
      fetch('/api/v1/safety/confirm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ preview_id: preview.preview_id, confirmed: false }),
      }).catch(() => {})
    }
    onCancel()
  }, [preview, onCancel])

  const riskColor = preview?.risk === 'blocked' ? 'var(--ops-status-danger)' : preview?.risk === 'high' ? 'var(--ops-status-danger)' : 'var(--ops-status-warn)'

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 10000, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(0,0,0,0.6)' }}>
      <div style={{ width: 480, maxHeight: '80vh', display: 'flex', flexDirection: 'column', background: 'var(--ops-bg-surface)', border: '1px solid var(--ops-border-default)', borderRadius: 6, overflow: 'hidden', boxShadow: '0 16px 48px rgba(0,0,0,0.5)' }}>
        {/* Header */}
        <div style={{ height: 40, display: 'flex', alignItems: 'center', padding: '0 16px', borderBottom: '1px solid var(--ops-border-subtle)', gap: 8 }}>
          <span className="material-symbols-outlined" style={{ fontSize: 18, color: 'var(--ops-status-warn)' }}>warning</span>
          <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--ops-fg-primary)' }}>风险预演</span>
        </div>

        {/* Body */}
        <div style={{ flex: 1, padding: 16, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 12 }}>
          {loading && <span style={{ fontSize: 12, color: 'var(--ops-fg-muted)' }}>正在创建预演...</span>}
          {error && <div style={{ fontSize: 12, color: 'var(--ops-status-danger)', padding: 8, background: 'rgba(228,48,48,0.08)', borderRadius: 4 }}>{error}</div>}

          {preview && (
            <>
              {/* Command display */}
              <div>
                <div style={{ fontSize: 11, color: 'var(--ops-fg-muted)', marginBottom: 4 }}>将要执行的命令:</div>
                <div style={{ padding: 8, background: 'var(--ops-bg-canvas)', borderRadius: 4, border: '1px solid var(--ops-border-subtle)', borderLeft: `3px solid ${riskColor}` }}>
                  <code style={{ fontSize: 13, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-primary)' }}>
                    {preview.command}
                  </code>
                </div>
              </div>

              {/* Risk level */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ fontSize: 11, color: 'var(--ops-fg-muted)' }}>风险等级:</span>
                <span style={{ fontSize: 12, fontWeight: 500, color: riskColor, textTransform: 'uppercase' }}>{preview.risk}</span>
              </div>

              {/* Description */}
              {preview.description && (
                <div>
                  <div style={{ fontSize: 11, color: 'var(--ops-fg-muted)', marginBottom: 4 }}>操作说明:</div>
                  <div style={{ fontSize: 12, color: 'var(--ops-fg-secondary)' }}>{preview.description}</div>
                </div>
              )}

              {/* Expiry */}
              <div style={{ fontSize: 11, color: 'var(--ops-fg-muted)' }}>
                预演有效期: 5 分钟（过期需重新操作）
              </div>

              {/* Blocked warning */}
              {preview.risk === 'blocked' && (
                <div style={{ padding: 8, background: 'rgba(228,48,48,0.08)', borderRadius: 4, border: '1px solid rgba(228,48,48,0.2)' }}>
                  <span style={{ fontSize: 12, color: 'var(--ops-status-danger)', fontWeight: 500 }}>
                    ⚠ 此命令已被安全护栏拦截，无法执行。
                  </span>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div style={{ height: 52, display: 'flex', alignItems: 'center', justifyContent: 'flex-end', padding: '0 16px', gap: 8, borderTop: '1px solid var(--ops-border-subtle)' }}>
          <button
            onClick={handleCancel}
            style={{ height: 32, padding: '0 16px', fontSize: 12, color: 'var(--ops-fg-secondary)', background: 'transparent', border: '1px solid var(--ops-border-default)', borderRadius: 4, cursor: 'pointer' }}
          >
            取消
          </button>
          <button
            onClick={handleConfirm}
            disabled={countdown > 0 || confirming || preview?.risk === 'blocked' || !!error}
            style={{
              height: 32, padding: '0 16px', fontSize: 12, fontWeight: 500,
              color: countdown > 0 || preview?.risk === 'blocked' ? 'var(--ops-fg-muted)' : '#fff',
              background: countdown > 0 || preview?.risk === 'blocked' ? 'var(--ops-bg-elevated)' : 'var(--ops-status-danger)',
              border: 'none', borderRadius: 4,
              cursor: countdown > 0 || preview?.risk === 'blocked' ? 'not-allowed' : 'pointer',
              opacity: confirming ? 0.6 : 1,
            }}
          >
            {confirming ? '执行中...' : countdown > 0 ? `确认执行 (${countdown}s)` : '确认执行'}
          </button>
        </div>
      </div>
    </div>
  )
}
