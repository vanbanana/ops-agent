// Data: None (通用窗口壳组件 — 拖拽/缩放/最小最大化/关闭)
import { type FC, type ReactNode, useRef, useCallback, useEffect } from 'react'
import type { WindowState } from '../../hooks/useWindowManager'

interface WindowProps {
  state: WindowState
  focused: boolean
  onFocus: () => void
  onClose: () => void
  onMinimize: () => void
  onMaximize: () => void
  onMove: (x: number, y: number) => void
  onResize: (width: number, height: number) => void
  children: ReactNode
}

export const Window: FC<WindowProps> = ({
  state, focused, onFocus, onClose, onMinimize, onMaximize, onMove, onResize, children,
}) => {
  const titleBarRef = useRef<HTMLDivElement>(null)
  const windowRef = useRef<HTMLDivElement>(null)
  const dragRef = useRef<{ startX: number; startY: number; winX: number; winY: number } | null>(null)
  const resizeRef = useRef<{ startX: number; startY: number; winW: number; winH: number; edge: string } | null>(null)

  // --- Drag logic ---
  const handleTitleMouseDown = useCallback((e: React.MouseEvent) => {
    if (state.maximized) return
    e.preventDefault()
    onFocus()
    dragRef.current = { startX: e.clientX, startY: e.clientY, winX: state.x, winY: state.y }
  }, [state.x, state.y, state.maximized, onFocus])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (dragRef.current) {
        const dx = e.clientX - dragRef.current.startX
        const dy = e.clientY - dragRef.current.startY
        onMove(dragRef.current.winX + dx, Math.max(0, dragRef.current.winY + dy))
      }
      if (resizeRef.current) {
        const dx = e.clientX - resizeRef.current.startX
        const dy = e.clientY - resizeRef.current.startY
        let newW = resizeRef.current.winW
        let newH = resizeRef.current.winH
        const edge = resizeRef.current.edge
        if (edge.includes('e')) newW = Math.max(state.minWidth, resizeRef.current.winW + dx)
        if (edge.includes('s')) newH = Math.max(state.minHeight, resizeRef.current.winH + dy)
        if (edge.includes('w')) newW = Math.max(state.minWidth, resizeRef.current.winW - dx)
        if (edge.includes('n')) newH = Math.max(state.minHeight, resizeRef.current.winH - dy)
        onResize(newW, newH)
        // If resizing from west/north, also move
        if (edge.includes('w')) {
          const actualDx = resizeRef.current.winW - newW
          onMove(state.x + actualDx, state.y)
        }
        if (edge.includes('n')) {
          const actualDy = resizeRef.current.winH - newH
          onMove(state.x, state.y + actualDy)
        }
      }
    }
    const handleMouseUp = () => {
      dragRef.current = null
      resizeRef.current = null
    }
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [state.x, state.y, state.minWidth, state.minHeight, onMove, onResize])

  // --- Resize edge detection ---
  const handleResizeMouseDown = useCallback((edge: string) => (e: React.MouseEvent) => {
    if (state.maximized) return
    e.preventDefault()
    e.stopPropagation()
    onFocus()
    resizeRef.current = { startX: e.clientX, startY: e.clientY, winW: state.width, winH: state.height, edge }
  }, [state.width, state.height, state.maximized, onFocus])

  // Double-click title = maximize toggle
  const handleTitleDoubleClick = useCallback(() => {
    onMaximize()
  }, [onMaximize])

  if (state.minimized) return null

  const style: React.CSSProperties = state.maximized
    ? { position: 'absolute', inset: 0, zIndex: state.zIndex }
    : { position: 'absolute', left: state.x, top: state.y, width: state.width, height: state.height, zIndex: state.zIndex }

  return (
    <div
      ref={windowRef}
      onMouseDown={onFocus}
      style={{
        ...style,
        display: 'flex',
        flexDirection: 'column',
        border: `1px solid ${focused ? 'var(--ops-border-default)' : 'var(--ops-border-subtle)'}`,
        borderRadius: state.maximized ? 0 : 6,
        overflow: 'hidden',
        background: 'var(--ops-bg-surface)',
        boxShadow: focused ? '0 8px 32px rgba(0,0,0,0.5)' : '0 4px 16px rgba(0,0,0,0.3)',
      }}
    >
      {/* Title bar */}
      <div
        ref={titleBarRef}
        onMouseDown={handleTitleMouseDown}
        onDoubleClick={handleTitleDoubleClick}
        style={{
          height: 32,
          display: 'flex',
          alignItems: 'center',
          padding: '0 8px',
          background: focused ? 'var(--ops-bg-elevated)' : 'var(--ops-bg-surface)',
          borderBottom: '1px solid var(--ops-border-subtle)',
          cursor: state.maximized ? 'default' : 'grab',
          flexShrink: 0,
          userSelect: 'none',
          gap: 6,
        }}
      >
        <span className="material-symbols-outlined" style={{ fontSize: 14, color: 'var(--ops-fg-secondary)' }}>
          {state.icon}
        </span>
        <span style={{ fontSize: 12, color: 'var(--ops-fg-primary)', fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {state.title}
        </span>
        {/* Window controls */}
        <WinBtn icon="remove" onClick={onMinimize} title="最小化" />
        <WinBtn icon={state.maximized ? 'filter_none' : 'crop_square'} onClick={onMaximize} title={state.maximized ? '还原' : '最大化'} />
        <WinBtn icon="close" onClick={onClose} title="关闭" danger />
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
        {children}
      </div>

      {/* Resize handles (only when not maximized) */}
      {!state.maximized && (
        <>
          <ResizeHandle edge="e" onMouseDown={handleResizeMouseDown('e')} />
          <ResizeHandle edge="s" onMouseDown={handleResizeMouseDown('s')} />
          <ResizeHandle edge="w" onMouseDown={handleResizeMouseDown('w')} />
          <ResizeHandle edge="n" onMouseDown={handleResizeMouseDown('n')} />
          <ResizeHandle edge="se" onMouseDown={handleResizeMouseDown('se')} />
          <ResizeHandle edge="sw" onMouseDown={handleResizeMouseDown('sw')} />
          <ResizeHandle edge="ne" onMouseDown={handleResizeMouseDown('ne')} />
          <ResizeHandle edge="nw" onMouseDown={handleResizeMouseDown('nw')} />
        </>
      )}
    </div>
  )
}

const WinBtn: FC<{ icon: string; onClick: () => void; title: string; danger?: boolean }> = ({ icon, onClick, title, danger }) => (
  <button
    onClick={(e) => { e.stopPropagation(); onClick() }}
    title={title}
    style={{
      width: 24, height: 24, display: 'flex', alignItems: 'center', justifyContent: 'center',
      border: 'none', borderRadius: 4, cursor: 'pointer', background: 'transparent',
      color: danger ? 'var(--ops-status-danger)' : 'var(--ops-fg-muted)',
    }}
    onMouseEnter={(e) => { e.currentTarget.style.background = danger ? 'rgba(228,48,48,0.15)' : 'var(--ops-bg-canvas)' }}
    onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
  >
    <span className="material-symbols-outlined" style={{ fontSize: 14 }}>{icon}</span>
  </button>
)

const EDGE_SIZE = 5

const edgeStyles: Record<string, React.CSSProperties> = {
  n:  { top: 0, left: EDGE_SIZE, right: EDGE_SIZE, height: EDGE_SIZE, cursor: 'n-resize' },
  s:  { bottom: 0, left: EDGE_SIZE, right: EDGE_SIZE, height: EDGE_SIZE, cursor: 's-resize' },
  e:  { top: EDGE_SIZE, right: 0, bottom: EDGE_SIZE, width: EDGE_SIZE, cursor: 'e-resize' },
  w:  { top: EDGE_SIZE, left: 0, bottom: EDGE_SIZE, width: EDGE_SIZE, cursor: 'w-resize' },
  se: { bottom: 0, right: 0, width: EDGE_SIZE * 2, height: EDGE_SIZE * 2, cursor: 'se-resize' },
  sw: { bottom: 0, left: 0, width: EDGE_SIZE * 2, height: EDGE_SIZE * 2, cursor: 'sw-resize' },
  ne: { top: 0, right: 0, width: EDGE_SIZE * 2, height: EDGE_SIZE * 2, cursor: 'ne-resize' },
  nw: { top: 0, left: 0, width: EDGE_SIZE * 2, height: EDGE_SIZE * 2, cursor: 'nw-resize' },
}

const ResizeHandle: FC<{ edge: string; onMouseDown: (e: React.MouseEvent) => void }> = ({ edge, onMouseDown }) => (
  <div onMouseDown={onMouseDown} style={{ position: 'absolute', ...edgeStyles[edge] }} />
)
