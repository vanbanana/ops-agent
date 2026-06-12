import { authFetch } from '../lib/auth'
// Data: None (local terminal, future WebSocket to backend)
// Layout: bottom drawer with split pane support (horizontal/vertical) + fullscreen mode
import { type FC, useState, useRef, useCallback, useEffect } from 'react'
import { Allotment } from 'allotment'
import 'allotment/dist/style.css'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

// ============================
// Types
// ============================

type SplitDirection = 'horizontal' | 'vertical'

interface TerminalPane {
  id: string
  type: 'terminal'
  title: string
}

interface SplitPane {
  id: string
  type: 'split'
  direction: SplitDirection
  children: PaneNode[]
}

type PaneNode = TerminalPane | SplitPane

// ============================
// Single Terminal Instance
// ============================

const XTermInstance: FC<{ paneId: string; isActive: boolean; onFocus: (id: string) => void }> = ({
  paneId,
  isActive,
  onFocus,
}) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)

  useEffect(() => {
    if (!containerRef.current || termRef.current) return

    const terminal = new Terminal({
      theme: {
        background: '#171717',
        foreground: '#CCCCCC',
        cursor: '#CCCCCC',
        cursorAccent: '#171717',
        selectionBackground: 'rgba(55, 148, 255, 0.25)',
        black: '#2B2B2B',
        red: '#E01C22',
        green: '#009431',
        yellow: '#F5A214',
        blue: '#3794FF',
        magenta: '#B07BD6',
        cyan: '#02ADF1',
        white: '#CCCCCC',
        brightBlack: '#808080',
        brightRed: '#FA4C46',
        brightGreen: '#3EC273',
        brightYellow: '#FFC147',
        brightBlue: '#6CA7EB',
        brightMagenta: '#CE93D8',
        brightCyan: '#56D4DD',
        brightWhite: '#F2F2F2',
      },
      fontFamily: 'Menlo, Monaco, Consolas, "Droid Sans Mono", "Courier New", monospace',
      fontSize: 12,
      lineHeight: 1.3,
      cursorBlink: true,
      cursorStyle: 'bar',
      scrollback: 10000,
      allowTransparency: true,
    })

    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.open(containerRef.current)
    fitAddon.fit()

    termRef.current = terminal
    fitRef.current = fitAddon

    // Local echo replaced with real backend exec
    let line = ''
    terminal.writeln('\x1b[38;5;70m# OPS·AGENT Terminal\x1b[0m')
    terminal.writeln('\x1b[2m# 命令将通过安全校验后在服务器执行\x1b[0m')
    terminal.write('\x1b[38;5;70m>\x1b[0m ')

    terminal.onKey(({ key, domEvent }) => {
      if (domEvent.key === 'Enter') {
        terminal.write('\r\n')
        const cmd = line.trim()
        if (cmd) {
          // Execute via backend API
          authFetch('/api/v1/terminal/exec', {
            method: 'POST',
            body: JSON.stringify({ command: cmd, timeout: 30 }),
          })
            .then(r => r.json())
            .then(data => {
              if (data?.data?.blocked) {
                terminal.writeln(`\x1b[31m[blocked] ${data.data.error}\x1b[0m`)
              } else if (data?.data?.error) {
                terminal.writeln(`\x1b[31m${data.data.error}\x1b[0m`)
              } else if (data?.data?.output) {
                const lines = data.data.output.split('\n')
                for (const l of lines) {
                  terminal.writeln(l)
                }
              }
              if (data?.data?.exit_code && data.data.exit_code !== 0) {
                terminal.writeln(`\x1b[2m[exit ${data.data.exit_code}]\x1b[0m`)
              }
              terminal.write('\x1b[38;5;70m>\x1b[0m ')
            })
            .catch(err => {
              terminal.writeln(`\x1b[31m[error] ${err}\x1b[0m`)
              terminal.write('\x1b[38;5;70m>\x1b[0m ')
            })
        } else {
          terminal.write('\x1b[38;5;70m>\x1b[0m ')
        }
        line = ''
      } else if (domEvent.key === 'Backspace') {
        if (line.length > 0) {
          line = line.slice(0, -1)
          terminal.write('\b \b')
        }
      } else if (domEvent.key === 'Tab') {
        domEvent.preventDefault()
      } else if (key.length === 1 && !domEvent.ctrlKey && !domEvent.altKey && !domEvent.metaKey) {
        line += key
        terminal.write(key)
      }
    })

    // Listen for external command execution (from quick commands panel)
    const handleExternalExec = (e: Event) => {
      const cmd = (e as CustomEvent).detail?.command
      if (!cmd || !terminal) return
      terminal.writeln('\x1b[2m' + '\u2500'.repeat(40) + '\x1b[0m')
      terminal.writeln(`\x1b[38;5;70m>\x1b[0m \x1b[33m${cmd}\x1b[0m`)
      authFetch('/api/v1/terminal/exec', {
        method: 'POST',
        body: JSON.stringify({ command: cmd, timeout: 30 }),
      })
        .then(r => r.json())
        .then(data => {
          if (data?.data?.blocked) {
            terminal.writeln(`\x1b[31m[blocked] ${data.data.error}\x1b[0m`)
          } else if (data?.data?.error) {
            terminal.writeln(`\x1b[31m${data.data.error}\x1b[0m`)
          } else if (data?.data?.output) {
            const lines = data.data.output.split('\n')
            for (const l of lines) {
              terminal.writeln(l)
            }
          }
          if (data?.data?.exit_code && data.data.exit_code !== 0) {
            terminal.writeln(`\x1b[2m[exit ${data.data.exit_code}]\x1b[0m`)
          }
          terminal.write('\x1b[38;5;70m>\x1b[0m ')
        })
        .catch(err => {
          terminal.writeln(`\x1b[31m[error] ${err}\x1b[0m`)
          terminal.write('\x1b[38;5;70m>\x1b[0m ')
        })
    }
    window.addEventListener('terminal:exec', handleExternalExec)

    return () => {
      window.removeEventListener('terminal:exec', handleExternalExec)
      terminal.dispose()
      termRef.current = null
      fitRef.current = null
    }
  }, [])

  // Refit on resize via ResizeObserver
  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const observer = new ResizeObserver(() => {
      if (fitRef.current) {
        try {
          fitRef.current.fit()
        } catch {
          // ignore fit errors during transitions
        }
      }
    })
    observer.observe(container)
    return () => observer.disconnect()
  }, [])

  return (
    <div
      ref={containerRef}
      onClick={() => onFocus(paneId)}
      style={{
        width: '100%',
        height: '100%',
        overflow: 'hidden',
        background: '#171717',
        outline: isActive ? '1px solid rgba(91, 155, 213, 0.4)' : 'none',
        outlineOffset: -1,
      }}
    />
  )
}

// ============================
// Recursive Pane Renderer
// ============================

const PaneRenderer: FC<{
  node: PaneNode
  activeId: string
  onFocus: (id: string) => void
}> = ({ node, activeId, onFocus }) => {
  if (node.type === 'terminal') {
    return (
      <div style={{ width: '100%', height: '100%', overflow: 'hidden' }}>
        <XTermInstance paneId={node.id} isActive={activeId === node.id} onFocus={onFocus} />
      </div>
    )
  }

  // Split pane — wrapper must have explicit dimensions for Allotment to work
  return (
    <div style={{ width: '100%', height: '100%', overflow: 'hidden' }}>
      <Allotment vertical={node.direction === 'vertical'}>
        {node.children.map((child) => (
          <Allotment.Pane key={child.id} minSize={60}>
            <PaneRenderer node={child} activeId={activeId} onFocus={onFocus} />
          </Allotment.Pane>
        ))}
      </Allotment>
    </div>
  )
}

// ============================
// Main TerminalDrawer
// ============================

interface TerminalDrawerProps {
  isFullscreen: boolean
  onToggleFullscreen: () => void
}

export const TerminalDrawer: FC<TerminalDrawerProps> = ({ isFullscreen, onToggleFullscreen }) => {
  const [expanded, setExpanded] = useState(false)
  const [height, setHeight] = useState(280)
  const [rootPane, setRootPane] = useState<PaneNode>({
    id: 'root-term',
    type: 'terminal',
    title: 'Terminal 1',
  })
  const [activeId, setActiveId] = useState('root-term')
  const [termCount, setTermCount] = useState(1)

  const dragging = useRef(false)
  const startY = useRef(0)
  const startH = useRef(0)

  // Split active pane
  const splitPane = useCallback(
    (direction: SplitDirection) => {
      const newId = `term-${Date.now()}`
      setTermCount((c) => c + 1)

      // Auto-expand height when vertical splitting to fit both panes
      if (direction === 'vertical') {
        const maxH = window.innerHeight - 130
        setHeight((h) => Math.min(maxH, Math.max(h, 400)))
      }

      if (!expanded) setExpanded(true)

      setRootPane((prev) => {
        const newTerminal: TerminalPane = {
          id: newId,
          type: 'terminal',
          title: `Terminal ${termCount + 1}`,
        }

        const splitNode = (node: PaneNode): PaneNode => {
          if (node.type === 'terminal' && node.id === activeId) {
            return {
              id: `split-${Date.now()}`,
              type: 'split',
              direction,
              children: [node, newTerminal],
            }
          }
          if (node.type === 'split') {
            return {
              ...node,
              children: node.children.map(splitNode),
            }
          }
          return node
        }

        return splitNode(prev)
      })

      setActiveId(newId)
    },
    [activeId, expanded, termCount]
  )

  // Close active pane
  const closeActivePane = useCallback(() => {
    setRootPane((prev) => {
      const removeNode = (node: PaneNode): PaneNode | null => {
        if (node.type === 'terminal') {
          return node.id === activeId ? null : node
        }
        const remaining = node.children
          .map(removeNode)
          .filter(Boolean) as PaneNode[]
        if (remaining.length === 0) return null
        if (remaining.length === 1) return remaining[0]
        return { ...node, children: remaining }
      }

      const result = removeNode(prev)
      if (!result) {
        // Last pane closed, reset
        setExpanded(false)
        const newTerm: TerminalPane = { id: 'root-term', type: 'terminal', title: 'Terminal 1' }
        setActiveId('root-term')
        return newTerm
      }
      // Set active to first terminal found
      const findFirst = (n: PaneNode): string => {
        if (n.type === 'terminal') return n.id
        return findFirst(n.children[0])
      }
      setActiveId(findFirst(result))
      return result
    })
  }, [activeId])

  // Drag resize
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      dragging.current = true
      startY.current = e.clientY
      startH.current = height

      const maxH = window.innerHeight - 130 // leave room for header + resource strip
      const onMove = (ev: MouseEvent) => {
        if (!dragging.current) return
        const diff = startY.current - ev.clientY
        setHeight(Math.min(maxH, Math.max(140, startH.current + diff)))
      }
      const onUp = () => {
        dragging.current = false
        document.removeEventListener('mousemove', onMove)
        document.removeEventListener('mouseup', onUp)
      }
      document.addEventListener('mousemove', onMove)
      document.addEventListener('mouseup', onUp)
    },
    [height]
  )

  const toggleExpand = () => setExpanded(!expanded)

  // Fullscreen mode — render just the terminal with toolbar
  if (isFullscreen) {
    return (
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          background: '#171717',
          overflow: 'hidden',
        }}
      >
        <TerminalToolbar
          onSplitH={() => splitPane('horizontal')}
          onSplitV={() => splitPane('vertical')}
          onClose={closeActivePane}
          onToggleFullscreen={onToggleFullscreen}
          isFullscreen={true}
        />
        <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
          <div style={{ position: 'absolute', inset: 0 }}>
            <PaneRenderer node={rootPane} activeId={activeId} onFocus={setActiveId} />
          </div>
        </div>
      </div>
    )
  }

  // Collapsed state
  if (!expanded) {
    return (
      <div
        onClick={toggleExpand}
        style={{
          height: 32,
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '0 16px',
          background: 'var(--ops-bg-surface)',
          borderTop: '1px solid var(--ops-border-subtle)',
          flexShrink: 0,
          cursor: 'pointer',
          userSelect: 'none',
        }}
      >
        <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>▴</span>
        <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-secondary)' }}>Terminal</span>
        <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>devops-runner@host</span>
      </div>
    )
  }

  // Expanded drawer mode
  return (
    <div
      style={{
        height,
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
        background: '#171717',
        borderTop: '1px solid var(--ops-border-subtle)',
        flexShrink: 0,
      }}
    >
      {/* Drag handle */}
      <div
        onMouseDown={handleMouseDown}
        style={{
          height: 4,
          cursor: 'ns-resize',
          background: 'transparent',
          flexShrink: 0,
        }}
      >
        <div style={{ height: 1, background: 'var(--ops-border-subtle)', margin: '1px 0' }} />
      </div>

      {/* Toolbar */}
      <TerminalToolbar
        onSplitH={() => splitPane('horizontal')}
        onSplitV={() => splitPane('vertical')}
        onClose={closeActivePane}
        onToggleFullscreen={onToggleFullscreen}
        isFullscreen={false}
        onCollapse={toggleExpand}
      />

      {/* Terminal panes */}
      <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
        <div style={{ position: 'absolute', inset: 0 }}>
          <PaneRenderer node={rootPane} activeId={activeId} onFocus={setActiveId} />
        </div>
      </div>
    </div>
  )
}

// ============================
// Toolbar
// ============================

const TerminalToolbar: FC<{
  onSplitH: () => void
  onSplitV: () => void
  onClose: () => void
  onToggleFullscreen: () => void
  isFullscreen: boolean
  onCollapse?: () => void
}> = ({ onSplitH, onSplitV, onClose, onToggleFullscreen, isFullscreen, onCollapse }) => (
  <div
    style={{
      height: 34,
      display: 'flex',
      alignItems: 'center',
      gap: 2,
      padding: '0 8px',
      background: 'var(--ops-bg-sidebar)',
      borderBottom: '1px solid var(--ops-border-subtle)',
      flexShrink: 0,
      userSelect: 'none',
    }}
  >
    {/* Left: title */}
    <span
      style={{
        fontFamily: 'var(--ops-font-mono)',
        fontSize: 11,
        color: 'var(--ops-fg-muted)',
        padding: '0 6px',
      }}
    >
      TERMINAL
    </span>

    <div style={{ flex: 1 }} />

    {/* Right: action buttons */}
    <ToolbarBtn icon="vertical_split" title="水平分割" onClick={onSplitH} />
    <ToolbarBtn icon="horizontal_split" title="垂直分割" onClick={onSplitV} />
    <ToolbarBtn
      icon={isFullscreen ? 'close_fullscreen' : 'open_in_full'}
      title={isFullscreen ? '退出全屏' : '全屏终端'}
      onClick={onToggleFullscreen}
    />
    <ToolbarBtn icon="close" title="关闭当前" onClick={onClose} />
    {!isFullscreen && onCollapse && (
      <ToolbarBtn icon="keyboard_arrow_down" title="收起" onClick={onCollapse} />
    )}
  </div>
)

const ToolbarBtn: FC<{ icon: string; title: string; onClick: () => void }> = ({ icon, title, onClick }) => (
  <button
    onClick={onClick}
    title={title}
    style={{
      width: 26,
      height: 26,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      borderRadius: 3,
      border: 'none',
      background: 'transparent',
      cursor: 'pointer',
      color: 'var(--ops-fg-muted)',
    }}
    onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
    onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
  >
    <span className="material-symbols-outlined" style={{ fontSize: 16 }}>{icon}</span>
  </button>
)
