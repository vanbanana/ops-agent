// Data: None (纯前端窗口状态管理)
import { useState, useCallback, useRef } from 'react'

export type DesktopApp =
  | 'files'
  | 'trash'
  | 'monitor'
  | 'process'
  | 'network'
  | 'logs'
  | 'services'
  | 'terminal'
  | 'security'

export interface WindowState {
  id: string
  app: DesktopApp
  title: string
  icon: string
  x: number
  y: number
  width: number
  height: number
  minWidth: number
  minHeight: number
  zIndex: number
  minimized: boolean
  maximized: boolean
}

// Default sizes/positions for each app type
const APP_DEFAULTS: Record<DesktopApp, { title: string; icon: string; width: number; height: number; minWidth: number; minHeight: number }> = {
  files:    { title: '文件管理器', icon: 'folder',           width: 800, height: 500, minWidth: 600, minHeight: 400 },
  trash:    { title: '回收站',     icon: 'delete',           width: 700, height: 450, minWidth: 500, minHeight: 350 },
  monitor:  { title: '系统监控',   icon: 'monitoring',       width: 700, height: 500, minWidth: 500, minHeight: 400 },
  process:  { title: '进程管理',   icon: 'memory',           width: 750, height: 500, minWidth: 550, minHeight: 350 },
  network:  { title: '网络',       icon: 'lan',              width: 700, height: 450, minWidth: 500, minHeight: 350 },
  logs:     { title: '日志查看器', icon: 'article',          width: 800, height: 500, minWidth: 600, minHeight: 400 },
  services: { title: '服务管理',   icon: 'settings_suggest', width: 700, height: 450, minWidth: 500, minHeight: 350 },
  terminal: { title: '终端',       icon: 'terminal',         width: 800, height: 500, minWidth: 500, minHeight: 300 },
  security: { title: '安全中心',   icon: 'shield',           width: 650, height: 450, minWidth: 500, minHeight: 350 },
}

let windowIdCounter = 0
let globalZIndex = 100

export function useWindowManager() {
  const [windows, setWindows] = useState<WindowState[]>([])
  const [focusedId, setFocusedId] = useState<string | null>(null)
  const offsetRef = useRef(0) // cascade offset for new windows

  const openWindow = useCallback((app: DesktopApp) => {
    const defaults = APP_DEFAULTS[app]
    const id = `win-${++windowIdCounter}`
    globalZIndex++
    const offset = offsetRef.current * 30
    offsetRef.current = (offsetRef.current + 1) % 8

    const newWin: WindowState = {
      id,
      app,
      title: defaults.title,
      icon: defaults.icon,
      x: 80 + offset,
      y: 40 + offset,
      width: defaults.width,
      height: defaults.height,
      minWidth: defaults.minWidth,
      minHeight: defaults.minHeight,
      zIndex: globalZIndex,
      minimized: false,
      maximized: false,
    }

    setWindows(prev => [...prev, newWin])
    setFocusedId(id)
  }, [])

  const closeWindow = useCallback((id: string) => {
    setWindows(prev => prev.filter(w => w.id !== id))
    setFocusedId(prev => prev === id ? null : prev)
  }, [])

  const focusWindow = useCallback((id: string) => {
    globalZIndex++
    setWindows(prev => prev.map(w =>
      w.id === id ? { ...w, zIndex: globalZIndex, minimized: false } : w
    ))
    setFocusedId(id)
  }, [])

  const minimizeWindow = useCallback((id: string) => {
    setWindows(prev => prev.map(w =>
      w.id === id ? { ...w, minimized: true } : w
    ))
    setFocusedId(prev => prev === id ? null : prev)
  }, [])

  const maximizeWindow = useCallback((id: string) => {
    setWindows(prev => prev.map(w =>
      w.id === id ? { ...w, maximized: !w.maximized } : w
    ))
  }, [])

  const updateWindow = useCallback((id: string, patch: Partial<WindowState>) => {
    setWindows(prev => prev.map(w =>
      w.id === id ? { ...w, ...patch } : w
    ))
  }, [])

  // Toggle: if minimized, restore; if focused, minimize; else focus
  const toggleWindow = useCallback((id: string) => {
    const win = windows.find(w => w.id === id)
    if (!win) return
    if (win.minimized) {
      focusWindow(id)
    } else if (focusedId === id) {
      minimizeWindow(id)
    } else {
      focusWindow(id)
    }
  }, [windows, focusedId, focusWindow, minimizeWindow])

  return {
    windows,
    focusedId,
    openWindow,
    closeWindow,
    focusWindow,
    minimizeWindow,
    maximizeWindow,
    updateWindow,
    toggleWindow,
  }
}
