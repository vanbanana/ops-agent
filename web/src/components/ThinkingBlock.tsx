import { type FC, useState, useEffect, useRef } from 'react'
import type { ThinkingData } from '../types/api'

interface ThinkingBlockProps {
  thinking: ThinkingData
}

export const ThinkingBlock: FC<ThinkingBlockProps> = ({ thinking }) => {
  const isThinking = thinking.status === 'thinking'
  const [manualToggle, setManualToggle] = useState<boolean | null>(null) // null = auto mode
  const wasThinkingRef = useRef(isThinking)

  // Auto-collapse: when thinking finishes, always collapse
  useEffect(() => {
    if (wasThinkingRef.current && !isThinking) {
      setManualToggle(false)
    }
    wasThinkingRef.current = isThinking
  }, [isThinking])

  // Determine expanded state: manual override > auto logic
  // While thinking: always expanded. After done: collapsed (unless manually opened)
  const expanded = manualToggle !== null ? manualToggle : isThinking

  return (
    <div
      style={{
        borderLeft: '2px solid var(--ops-status-info)',
        borderRadius: '0 4px 4px 0',
        background: 'var(--ops-bg-elevated)',
        overflow: 'hidden',
      }}
    >
      <button
        onClick={() => setManualToggle(!expanded)}
        style={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '6px 10px',
          border: 'none',
          background: 'transparent',
          cursor: 'pointer',
          textAlign: 'left',
        }}
      >
        <span
          className="material-symbols-outlined"
          style={{
            fontSize: 14,
            color: 'var(--ops-status-info)',
            animation: isThinking ? 'spin 2s linear infinite' : 'none',
          }}
        >
          psychology
        </span>
        <span style={{ fontSize: 12, fontFamily: 'var(--ops-font-ui)', color: 'var(--ops-fg-secondary)', flex: 1 }}>
          {isThinking ? '正在思考...' : '思考已完成'}
        </span>
        <span
          className="material-symbols-outlined"
          style={{
            fontSize: 14,
            color: 'var(--ops-fg-muted)',
            transform: expanded ? 'rotate(180deg)' : 'none',
            transition: 'transform 150ms',
          }}
        >
          expand_more
        </span>
      </button>

      {expanded && (
        <div style={{ padding: '6px 10px 8px', borderTop: '1px solid var(--ops-border-subtle)', maxHeight: 300, overflowY: 'auto' }}>
          {thinking.analyzeSummary && (
            <div style={{ fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)', lineHeight: 1.6, whiteSpace: 'pre-wrap' }}>
              {thinking.analyzeSummary}
            </div>
          )}
          {thinking.planTools && thinking.planTools.length > 0 && (
            <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginTop: 4 }}>
              {thinking.planTools.map((tool) => (
                <span
                  key={tool}
                  style={{
                    fontSize: 10,
                    fontFamily: 'var(--ops-font-mono)',
                    padding: '1px 6px',
                    borderRadius: 3,
                    background: 'var(--ops-bg-canvas)',
                    color: 'var(--ops-fg-muted)',
                  }}
                >
                  {tool}
                </span>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
