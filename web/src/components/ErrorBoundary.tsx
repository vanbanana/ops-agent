// 全局错误边界 — 捕获渲染崩溃，显示可调试的错误页面
import { Component, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
  errorInfo: string
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null, errorInfo: '' }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: { componentStack?: string | null }) {
    this.setState({ errorInfo: info.componentStack ?? '' })
    // Log to console for dev tools
    console.error('[ErrorBoundary]', error, info.componentStack)
  }

  render() {
    if (!this.state.hasError) return this.props.children

    const { error, errorInfo } = this.state

    return (
      <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--ops-bg-canvas, #171717)', padding: 32 }}>
        <div style={{ maxWidth: 640, width: '100%' }}>
          {/* Header */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 16 }}>
            <span className="material-symbols-outlined" style={{ fontSize: 24, color: '#E01C22' }}>error</span>
            <span style={{ fontSize: 16, fontWeight: 600, color: '#CCCCCC' }}>渲染错误</span>
          </div>

          {/* Error message */}
          <div style={{ padding: 12, background: '#1a1a1a', border: '1px solid #3D3D3D', borderLeft: '3px solid #E01C22', borderRadius: 4, marginBottom: 12 }}>
            <div style={{ fontSize: 13, fontFamily: 'Menlo, Monaco, Consolas, monospace', color: '#F85149', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
              {error?.message ?? 'Unknown error'}
            </div>
          </div>

          {/* Stack trace */}
          {errorInfo && (
            <details style={{ marginBottom: 12 }}>
              <summary style={{ fontSize: 12, color: '#9E9E9E', cursor: 'pointer', marginBottom: 8 }}>
                组件堆栈 (点击展开)
              </summary>
              <pre style={{ fontSize: 11, fontFamily: 'Menlo, Monaco, Consolas, monospace', color: '#9E9E9E', background: '#1a1a1a', padding: 12, borderRadius: 4, border: '1px solid #3D3D3D', overflow: 'auto', maxHeight: 300, whiteSpace: 'pre-wrap' }}>
                {errorInfo}
              </pre>
            </details>
          )}

          {error?.stack && (
            <details style={{ marginBottom: 16 }}>
              <summary style={{ fontSize: 12, color: '#9E9E9E', cursor: 'pointer', marginBottom: 8 }}>
                JS 堆栈 (点击展开)
              </summary>
              <pre style={{ fontSize: 11, fontFamily: 'Menlo, Monaco, Consolas, monospace', color: '#707070', background: '#1a1a1a', padding: 12, borderRadius: 4, border: '1px solid #3D3D3D', overflow: 'auto', maxHeight: 300, whiteSpace: 'pre-wrap' }}>
                {error.stack}
              </pre>
            </details>
          )}

          {/* Actions */}
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              onClick={() => window.location.reload()}
              style={{ height: 32, padding: '0 16px', fontSize: 12, fontWeight: 500, color: '#fff', background: '#388BFD', border: 'none', borderRadius: 4, cursor: 'pointer' }}
            >
              刷新页面
            </button>
            <button
              onClick={() => this.setState({ hasError: false, error: null, errorInfo: '' })}
              style={{ height: 32, padding: '0 16px', fontSize: 12, color: '#CCCCCC', background: 'transparent', border: '1px solid #3D3D3D', borderRadius: 4, cursor: 'pointer' }}
            >
              尝试恢复
            </button>
            <button
              onClick={() => { navigator.clipboard.writeText(`${error?.message}\n\n${error?.stack}\n\nComponent Stack:\n${errorInfo}`) }}
              style={{ height: 32, padding: '0 16px', fontSize: 12, color: '#9E9E9E', background: 'transparent', border: '1px solid #3D3D3D', borderRadius: 4, cursor: 'pointer' }}
            >
              复制错误信息
            </button>
          </div>

          {/* Footer hint */}
          <div style={{ marginTop: 16, fontSize: 11, color: '#707070' }}>
            提示: 打开浏览器 DevTools (F12) 查看完整错误日志。如果问题持续出现，请检查控制台的 [ErrorBoundary] 输出。
          </div>
        </div>
      </div>
    )
  }
}
