// 工具调用生命周期卡片 — running(旋转) → success(展开结果) → error(红色)
import { type FC, useState } from 'react'
import type { ToolCallBlock } from '../types/api'

interface ToolCallCardProps {
  toolCall: ToolCallBlock
}

export const ToolCallCard: FC<ToolCallCardProps> = ({ toolCall }) => {
  const [expanded, setExpanded] = useState(false)

  const isRunning = toolCall.status === 'running'
  const isError = toolCall.status === 'error'
  const isDone = toolCall.status === 'done'

  const borderColor = isError ? 'var(--ops-status-danger)' : isDone ? 'var(--ops-status-ok)' : 'var(--ops-border-default)'
  const iconColor = isError ? 'var(--ops-status-danger)' : isDone ? 'var(--ops-status-ok)' : 'var(--ops-status-info)'

  return (
    <div
      style={{
        margin: '6px 0',
        border: `1px solid ${borderColor}`,
        borderRadius: 4,
        overflow: 'hidden',
        transition: 'border-color 300ms',
      }}
    >
      {/* Header — always visible */}
      <button
        onClick={() => setExpanded(!expanded)}
        style={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '6px 10px',
          border: 'none',
          background: 'var(--ops-bg-elevated)',
          cursor: 'pointer',
          textAlign: 'left',
        }}
      >
        {/* Status icon */}
        <span
          className="material-symbols-outlined"
          style={{
            fontSize: 14,
            color: iconColor,
            animation: isRunning ? 'spin 1s linear infinite' : 'none',
          }}
        >
          {isRunning ? 'progress_activity' : isError ? 'error' : 'check_circle'}
        </span>

        {/* Tool name */}
        <span style={{ fontSize: 12, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-primary)', flex: 1 }}>
          {toolCall.tool}
        </span>

        {/* Duration */}
        {toolCall.elapsed_ms != null && (
          <span style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)' }}>
            {toolCall.elapsed_ms}ms
          </span>
        )}

        {/* Expand arrow */}
        <span className="material-symbols-outlined" style={{ fontSize: 14, color: 'var(--ops-fg-muted)', transform: expanded ? 'rotate(180deg)' : 'none', transition: 'transform 150ms' }}>
          expand_more
        </span>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div style={{ padding: '8px 10px', borderTop: '1px solid var(--ops-border-subtle)', background: 'var(--ops-bg-canvas)' }}>
          {/* Args */}
          {toolCall.args && Object.keys(toolCall.args).length > 0 && (
            <div style={{ marginBottom: 6 }}>
              <div style={{ fontSize: 10, color: 'var(--ops-fg-muted)', marginBottom: 2 }}>参数</div>
              <pre style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-secondary)', margin: 0, whiteSpace: 'pre-wrap' }}>
                {JSON.stringify(toolCall.args, null, 2)}
              </pre>
            </div>
          )}

          {/* Result */}
          {toolCall.result_preview && (
            <div>
              <div style={{ fontSize: 10, color: 'var(--ops-fg-muted)', marginBottom: 2 }}>结果</div>
              <pre style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: isError ? 'var(--ops-status-danger)' : 'var(--ops-fg-primary)', margin: 0, whiteSpace: 'pre-wrap', maxHeight: 150, overflow: 'auto' }}>
                {toolCall.result_preview}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
