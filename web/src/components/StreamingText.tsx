// 流式文字渲染组件 — 接收完整文本，逐字动画追加显示
// 模拟打字效果，每个字符间隔可配置
import { type FC, useState, useEffect, useRef } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface StreamingTextProps {
  content: string
  speed?: number // ms per character, default 15
  onComplete?: () => void
  isStreaming?: boolean // true = 正在流式渲染中
}

export const StreamingText: FC<StreamingTextProps> = ({ content, speed = 15, onComplete, isStreaming = true }) => {
  const [displayedLength, setDisplayedLength] = useState(isStreaming ? 0 : content.length)
  const prevContentRef = useRef(content)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    // If content changed (new text arrived), start from where we were
    if (content !== prevContentRef.current) {
      // If content grew (append), continue from current position
      if (content.startsWith(prevContentRef.current)) {
        // Content was appended, keep current displayed length
      } else {
        // Content completely replaced, start fresh
        setDisplayedLength(0)
      }
      prevContentRef.current = content
    }

    if (!isStreaming) {
      // Not streaming, show all immediately
      setDisplayedLength(content.length)
      return
    }

    if (displayedLength >= content.length) {
      onComplete?.()
      return
    }

    // Animate character by character
    timerRef.current = setInterval(() => {
      setDisplayedLength(prev => {
        const next = Math.min(prev + 1, content.length)
        if (next >= content.length) {
          if (timerRef.current) clearInterval(timerRef.current)
          onComplete?.()
        }
        return next
      })
    }, speed)

    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [content, displayedLength, speed, isStreaming, onComplete])

  const displayed = content.slice(0, displayedLength)
  const showCursor = isStreaming && displayedLength < content.length

  return (
    <div style={{ position: 'relative' }}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
        {displayed}
      </ReactMarkdown>
      {showCursor && <span style={{ animation: 'blink 0.8s infinite', color: 'var(--ops-fg-muted)' }}>▊</span>}
    </div>
  )
}

// Minimal markdown component overrides for consistent styling
const markdownComponents = {
  p: ({ children }: any) => <p style={{ margin: '4px 0', lineHeight: '22px' }}>{children}</p>,
  h1: ({ children }: any) => <h1 style={{ fontSize: 16, fontWeight: 600, margin: '12px 0 4px' }}>{children}</h1>,
  h2: ({ children }: any) => <h2 style={{ fontSize: 14, fontWeight: 600, margin: '10px 0 4px' }}>{children}</h2>,
  h3: ({ children }: any) => <h3 style={{ fontSize: 13, fontWeight: 600, margin: '8px 0 4px' }}>{children}</h3>,
  code: ({ inline, children }: any) => inline
    ? <code style={{ fontSize: 12, fontFamily: 'var(--ops-font-mono)', background: 'var(--ops-bg-elevated)', padding: '1px 4px', borderRadius: 3 }}>{children}</code>
    : <pre style={{ fontSize: 12, fontFamily: 'var(--ops-font-mono)', background: 'var(--ops-bg-canvas)', padding: 8, borderRadius: 4, overflow: 'auto', margin: '4px 0' }}><code>{children}</code></pre>,
  ul: ({ children }: any) => <ul style={{ paddingLeft: 16, margin: '4px 0' }}>{children}</ul>,
  ol: ({ children }: any) => <ol style={{ paddingLeft: 16, margin: '4px 0' }}>{children}</ol>,
  li: ({ children }: any) => <li style={{ margin: '2px 0', lineHeight: '20px' }}>{children}</li>,
  table: ({ children }: any) => <table style={{ borderCollapse: 'collapse', fontSize: 12, margin: '8px 0', width: '100%' }}>{children}</table>,
  th: ({ children }: any) => <th style={{ border: '1px solid var(--ops-border-subtle)', padding: '4px 8px', fontWeight: 600, textAlign: 'left', background: 'var(--ops-bg-elevated)' }}>{children}</th>,
  td: ({ children }: any) => <td style={{ border: '1px solid var(--ops-border-subtle)', padding: '4px 8px' }}>{children}</td>,
}
