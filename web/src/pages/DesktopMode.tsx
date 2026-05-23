// Data: POST /api/v1/desktop/probe/{name}, GET /health
import { type FC, useCallback } from 'react'
import { useWindowManager, type DesktopApp } from '../hooks/useWindowManager'
import { Window } from '../components/desktop/Window'
import { DesktopIcons } from '../components/desktop/DesktopIcons'
import { Taskbar } from '../components/desktop/Taskbar'
import { FileManagerApp } from '../components/desktop/apps/FileManagerApp'
import { MonitorApp } from '../components/desktop/apps/MonitorApp'
import { TerminalApp } from '../components/desktop/apps/TerminalApp'
import { ProcessApp } from '../components/desktop/apps/ProcessApp'
import { NetworkApp } from '../components/desktop/apps/NetworkApp'
import { LogApp } from '../components/desktop/apps/LogApp'
import { ServiceApp } from '../components/desktop/apps/ServiceApp'
import { SecurityApp } from '../components/desktop/apps/SecurityApp'
import { TrashApp } from '../components/desktop/apps/TrashApp'
import type { ResourceData, HealthResponse } from '../types/api'

interface DesktopModeProps {
  resources: ResourceData
  health: HealthResponse | null
  connected: boolean
  onSwitchToChat: () => void
}

export const DesktopMode: FC<DesktopModeProps> = ({ resources, health, connected, onSwitchToChat }) => {
  const wm = useWindowManager()

  const handleOpenApp = useCallback((app: DesktopApp) => {
    wm.openWindow(app)
  }, [wm])

  const renderAppContent = (app: DesktopApp) => {
    switch (app) {
      case 'files':    return <FileManagerApp connected={connected} />
      case 'trash':    return <TrashApp connected={connected} />
      case 'monitor':  return <MonitorApp resources={resources} connected={connected} />
      case 'terminal': return <TerminalApp />
      case 'process':  return <ProcessApp connected={connected} />
      case 'network':  return <NetworkApp connected={connected} />
      case 'logs':     return <LogApp connected={connected} />
      case 'services': return <ServiceApp connected={connected} />
      case 'security': return <SecurityApp connected={connected} />
    }
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', position: 'relative' }}>
      {/* Desktop background + icons */}
      <div style={{ flex: 1, position: 'relative', background: 'var(--ops-bg-canvas)', overflow: 'hidden' }}>
        {/* Icons on desktop */}
        <DesktopIcons resources={resources} onOpenApp={handleOpenApp} onSwitchToChat={onSwitchToChat} />

        {/* Floating windows */}
        {wm.windows.map(win => (
          <Window
            key={win.id}
            state={win}
            focused={wm.focusedId === win.id}
            onFocus={() => wm.focusWindow(win.id)}
            onClose={() => wm.closeWindow(win.id)}
            onMinimize={() => wm.minimizeWindow(win.id)}
            onMaximize={() => wm.maximizeWindow(win.id)}
            onMove={(x, y) => wm.updateWindow(win.id, { x, y })}
            onResize={(width, height) => wm.updateWindow(win.id, { width, height })}
          >
            {renderAppContent(win.app)}
          </Window>
        ))}
      </div>

      {/* Taskbar */}
      <Taskbar
        windows={wm.windows}
        focusedId={wm.focusedId}
        health={health}
        connected={connected}
        onToggleWindow={wm.toggleWindow}
      />
    </div>
  )
}
