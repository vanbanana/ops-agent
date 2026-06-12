import { type FC, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { ChatMessage as ChatMessageType } from '../types/api'
import { ToolCallCard } from './ToolCallCard'
import { ThinkingIndicator } from './ThinkingIndicator'
import { ThinkingBlock } from './ThinkingBlock'
import { MessageActions } from './MessageActions'
import { CodeBlock } from './CodeBlock'

interface ChatMessageProps {
  message: ChatMessageType
  isStreaming?: boolean
  isLastAgent?: boolean
  onRetry?: () => void
  onStop?: () => void
}

export const ChatMessageComponent: FC<ChatMessageProps> = ({ message, isStreaming = false, isLastAgent = false, onRetry, onStop }) => {
  const isUser = message.role === 'user'

  // Blocked message — red injection card with rule details
  if (message.isBlocked) {
    const rules = (message.error as any)?.rules as Array<{rule_id: string; reason: string; snippet: string}> | undefined
    return (
      <MessageLayout
        avatar={<AvatarBadge bg="var(--ops-status-danger)" icon="shield" iconColor="#0E1116" />}
      >
        <div style={{ flex: 1, borderLeft: '2px solid var(--ops-status-danger)', padding: '8px 12px', borderRadius: '0 6px 6px 0', background: 'var(--ops-bg-elevated)' }}>
          <div style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, fontWeight: 600, color: 'var(--ops-status-danger)', marginBottom: 4 }}>
            安全拦截
          </div>
          <div style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 12, color: 'var(--ops-fg-secondary)' }}>
            {message.content}
          </div>
          {message.error && (
            <div style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)', marginTop: 4 }}>
              {message.error.error_code}: {message.error.message}
            </div>
          )}
          {rules && rules.length > 0 && (
            <div style={{ marginTop: 6, padding: '4px 8px', background: 'rgba(255,59,48,0.06)', borderRadius: 4 }}>
              {rules.map((r, i) => (
                <div key={i} style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)', padding: '2px 0' }}>
                  <span style={{ color: 'var(--ops-status-danger)', fontWeight: 500 }}>{r.rule_id}</span> {r.reason}
                </div>
              ))}
            </div>
          )}
        </div>
      </MessageLayout>
    )
  }

  // Error message card (unchanged)
  if (message.error && !message.isBlocked) {
    return (
      <MessageLayout
        avatar={<AvatarBadge bg="var(--ops-bg-elevated)" icon="error" iconColor="var(--ops-status-danger)" />}
      >
        <div style={{ flex: 1, borderLeft: '2px solid var(--ops-status-danger)', padding: '8px 12px', borderRadius: '0 6px 6px 0', background: 'var(--ops-bg-elevated)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, fontWeight: 600, color: 'var(--ops-status-danger)' }}>
              {message.error.error_code}
            </span>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-secondary)' }}>
              {message.error.message}
            </span>
          </div>
          {message.error.recoverable && onRetry && (
            <button
              onClick={onRetry}
              style={{
                marginTop: 8, padding: '4px 12px', borderRadius: 6,
                border: '1px solid var(--ops-border-default)', background: 'transparent',
                color: 'var(--ops-fg-secondary)', fontFamily: 'var(--ops-font-ui)', fontSize: 11, cursor: 'pointer',
              }}
            >
              重试
            </button>
          )}
        </div>
      </MessageLayout>
    )
  }

  // User message
  if (isUser) {
    return (
      <MessageLayout avatar={<AvatarBadge bg="var(--ops-border-strong)" text="U" />}>
        <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 14, color: 'var(--ops-fg-primary)', lineHeight: 1.6 }}>
          {message.content}
        </div>
      </MessageLayout>
    )
  }

  // Agent message — main rendering path
  const showThinkingIndicator = !message.content && isStreaming && isLastAgent && !(message.toolCalls && message.toolCalls.length > 0)
  const showCursor = isStreaming && isLastAgent && !!message.content

  return (
    <MessageLayout avatar={<AvatarBadge bg="var(--ops-bg-elevated)" icon="memory" iconColor="var(--ops-fg-secondary)" />}>
      <div
        style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
        onMouseEnter={(e) => { const actions = e.currentTarget.querySelector('.msg-actions') as HTMLElement; if (actions) actions.style.opacity = '1' }}
        onMouseLeave={(e) => { const actions = e.currentTarget.querySelector('.msg-actions') as HTMLElement; if (actions && !isStreaming) actions.style.opacity = '0' }}
      >
        {/* Thinking indicator — no content yet, no tools yet */}
        {showThinkingIndicator && <ThinkingIndicator />}

        {/* Reasoning/thinking process — streams in real-time from LLM */}
        {message.reasoning && (
          <ThinkingBlock thinking={{
            status: (message.content || !isStreaming) ? 'done' : 'thinking',
            analyzeSummary: message.reasoning,
          }} />
        )}

        {/* Tool calls */}
        {message.toolCalls && message.toolCalls.length > 0 && (
          <div>
            {message.toolCalls.map((tc, i) => (
              <ToolCallCard key={`${tc.tool}-${i}`} toolCall={tc} />
            ))}
          </div>
        )}

        {/* Markdown content + streaming cursor */}
        {message.content && (
          <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 14, color: 'var(--ops-fg-primary)', lineHeight: 1.6 }}>
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                code: ({ children, className }) => {
                  const langMatch = className?.match(/language-(\w+)/)
                  if (langMatch) {
                    const codeStr = extractText(children)
                    return <CodeBlock language={langMatch[1]} code={codeStr} />
                  }
                  return (
                    <code style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 12, padding: '1px 4px', borderRadius: 4, background: 'var(--ops-bg-input)', color: 'var(--ops-fg-secondary)' }}>
                      {children}
                    </code>
                  )
                },
                pre: ({ children }) => <>{children}</>,
                p: ({ children }) => <p style={{ margin: '4px 0', color: 'var(--ops-fg-primary)' }}>{children}</p>,
                strong: ({ children }) => <strong style={{ fontWeight: 600 }}>{children}</strong>,
                ul: ({ children }) => <ul style={{ paddingLeft: 16, margin: '4px 0' }}>{children}</ul>,
                ol: ({ children }) => <ol style={{ paddingLeft: 16, margin: '4px 0' }}>{children}</ol>,
                li: ({ children }) => <li style={{ margin: '2px 0', lineHeight: '20px' }}>{children}</li>,
                table: ({ children }) => <table style={{ borderCollapse: 'collapse', fontSize: 12, margin: '8px 0', width: '100%' }}>{children}</table>,
                th: ({ children }) => <th style={{ border: '1px solid var(--ops-border-subtle)', padding: '4px 8px', fontWeight: 600, textAlign: 'left', background: 'var(--ops-bg-elevated)' }}>{children}</th>,
                td: ({ children }) => <td style={{ border: '1px solid var(--ops-border-subtle)', padding: '4px 8px' }}>{children}</td>,
              }}
            >
              {message.content}
            </ReactMarkdown>
            {showCursor && <span style={{ animation: 'blink 0.8s step-end infinite', color: 'var(--ops-fg-muted)' }}>▊</span>}
          </div>
        )}

        {/* Action bar */}
        {message.role === 'agent' && (
          <MessageActions
            content={message.content}
            isStreaming={isStreaming && isLastAgent}
            onRetry={onRetry || (() => {})}
            onStop={onStop || (() => {})}
          />
        )}
      </div>
    </MessageLayout>
  )
}

// --- Helper components ---

const MessageLayout: FC<{ avatar: ReactNode; children: ReactNode }> = ({ avatar, children }) => (
  <div style={{ display: 'flex', gap: 10, width: '100%' }}>
    {avatar}
    <div style={{ flex: 1, minWidth: 0 }}>{children}</div>
  </div>
)

const AvatarBadge: FC<{ bg: string; icon?: string; iconColor?: string; text?: string }> = ({ bg, icon, iconColor, text }) => (
  <div style={{ width: 28, height: 28, borderRadius: 6, background: bg, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
    {icon && <span className="material-symbols-outlined" style={{ fontSize: 16, color: iconColor }}>{icon}</span>}
    {text && <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 600, color: 'var(--ops-fg-primary)' }}>{text}</span>}
  </div>
)

/** Extract plain text from ReactMarkdown children (which can be nested React elements) */
function extractText(children: ReactNode): string {
  if (typeof children === 'string') return children
  if (Array.isArray(children)) return children.map(extractText).join('')
  if (children && typeof children === 'object' && 'props' in children) {
    return extractText((children as { props: { children?: ReactNode } }).props.children)
  }
  return String(children ?? '')
}
