// Data: SSE events (sense, analyze, plan, execute, execute_done, output)
// Layout: vertical timeline with dot + connector line (10px column) + content column
import { type FC } from 'react'
import type { ReasoningStep } from '../types/api'

interface ReasoningPanelProps {
  steps: ReasoningStep[]
  isStreaming: boolean
}

const phaseConfig: Record<string, { label: string; color: string }> = {
  sense: { label: 'SENSE', color: 'var(--ops-fg-secondary)' },
  analyze: { label: 'ANALYZE', color: 'var(--ops-status-info)' },
  plan: { label: 'PLAN', color: '#A371F7' },
  execute: { label: 'EXECUTE', color: 'var(--ops-status-warn)' },
  output: { label: 'OUTPUT', color: 'var(--ops-status-ok)' },
}

export const ReasoningPanel: FC<ReasoningPanelProps> = ({ steps, isStreaming }) => {
  if (steps.length === 0 && !isStreaming) {
    return (
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>
          发起对话后显示推理链路
        </span>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {steps.map((step, i) => {
        const config = phaseConfig[step.phase] || phaseConfig.sense
        const isLast = i === steps.length - 1

        return (
          <div key={i} style={{ display: 'flex', gap: 10, width: '100%' }}>
            {/* Timeline column — 10px wide: dot + connector line */}
            <div
              style={{
                width: 10,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
              }}
            >
              {/* Dot */}
              <div
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  background: config.color,
                  flexShrink: 0,
                }}
              />
              {/* Connector line */}
              {!isLast && (
                <div
                  style={{
                    width: 2,
                    height: 24,
                    background: 'var(--ops-border-default)',
                  }}
                />
              )}
            </div>

            {/* Content column */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
              {/* Header row: phase name + timing + timestamp */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, width: '100%' }}>
                <span
                  style={{
                    fontFamily: 'var(--ops-font-mono)',
                    fontSize: 11,
                    fontWeight: 600,
                    color: 'var(--ops-fg-primary)',
                  }}
                >
                  {config.label}
                </span>
                {step.data?.elapsed_ms !== undefined && (
                  <span
                    style={{
                      fontFamily: 'var(--ops-font-mono)',
                      fontSize: 10,
                      color: 'var(--ops-status-ok)',
                    }}
                  >
                    {step.data.elapsed_ms}ms
                  </span>
                )}
                <div style={{ flex: 1 }} />
                <span
                  style={{
                    fontFamily: 'var(--ops-font-mono)',
                    fontSize: 9,
                    color: 'var(--ops-fg-muted)',
                  }}
                >
                  {step.timestamp}
                </span>
              </div>

              {/* Detail text */}
              <span
                style={{
                  fontFamily: 'var(--ops-font-mono)',
                  fontSize: 10,
                  color: 'var(--ops-fg-muted)',
                }}
              >
                {getStepDetail(step)}
              </span>
            </div>
          </div>
        )
      })}

      {/* Summary footer */}
      {steps.length > 0 && !isStreaming && (
        <>
          <div style={{ flex: 1 }} />
          <span
            style={{
              fontFamily: 'var(--ops-font-mono)',
              fontSize: 10,
              color: 'var(--ops-fg-muted)',
            }}
          >
            总计 {getTotalTime(steps)} · {steps.length}/{steps.length} 完成
          </span>
        </>
      )}
    </div>
  )
}

function getStepDetail(step: ReasoningStep): string {
  const d = step.data
  if (!d) return ''

  switch (step.phase) {
    case 'sense':
      return d.status === 'ok' ? `输入安全 ✓ · tokens: ${d.tokens || '--'}` : `拦截: ${d.reason || ''}`
    case 'analyze':
      return d.reply_preview ? `意图: ${d.reply_preview}` : `第 ${d.round || 1} 轮分析`
    case 'plan':
      if (d.tools && Array.isArray(d.tools)) {
        return d.tools.map((t: any) => t.name).join(', ') + ' · 低风险'
      }
      return ''
    case 'execute':
      if (d.status === 'done' || d.elapsed_ms) return `${d.tool || ''} 完成`
      return `${d.tool || ''} 执行中...`
    case 'output':
      return `tokens: ${d.tokens_used || '--'} out · 置信 ${d.confidence || '--'}`
    default:
      return ''
  }
}

function getTotalTime(steps: ReasoningStep[]): string {
  let total = 0
  for (const s of steps) {
    if (s.data?.elapsed_ms) total += Number(s.data.elapsed_ms)
  }
  if (total >= 1000) return `${(total / 1000).toFixed(3)}s`
  return `${total}ms`
}
