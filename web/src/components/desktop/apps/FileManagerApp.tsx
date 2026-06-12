import { authFetch } from '../../../lib/auth'
// Data: GET /api/v1/fs/list?path=... (文件列表)
import { type FC, useState, useEffect, useCallback, useRef } from 'react'

interface FileEntry {
  name: string
  type: 'file' | 'directory' | 'symlink'
  size: number
  mtime: string
  permissions: string
}

interface Clipboard {
  action: 'copy' | 'cut'
  sourcePath: string
  items: string[]
}

interface FileManagerAppProps {
  connected: boolean
}

type SortKey = 'name' | 'size' | 'mtime'

export const FileManagerApp: FC<FileManagerAppProps> = ({ connected }) => {
  const [currentPath, setCurrentPath] = useState('/')
  const [entries, setEntries] = useState<FileEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set())
  const [clipboard, setClipboard] = useState<Clipboard | null>(null)
  const [sortBy, setSortBy] = useState<SortKey>('name')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; target: string | null } | null>(null)
  const [renaming, setRenaming] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)

  // Fetch directory listing
  const fetchDir = useCallback(async (path: string) => {
    if (!connected) { setError('未连接后端'); return }
    setLoading(true)
    setError(null)
    setSelectedNames(new Set())
    try {
      const res = await authFetch(`/api/v1/fs/list?path=${encodeURIComponent(path)}`)
      if (!res.ok) {
        const text = await res.text()
        setError(`请求失败: ${res.status} ${text.slice(0, 100)}`)
        setEntries([])
      } else {
        const json = await res.json()
        setEntries(json.entries ?? [])
        setCurrentPath(path)
      }
    } catch (e) {
      setError(`网络错误: ${(e as Error).message}`)
    } finally {
      setLoading(false)
    }
  }, [connected])

  useEffect(() => { fetchDir(currentPath) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Navigation
  const navigateTo = useCallback((path: string) => {
    fetchDir(path)
  }, [fetchDir])

  const navigateUp = useCallback(() => {
    if (currentPath === '/') return
    const parent = currentPath.replace(/\/[^/]+\/?$/, '') || '/'
    navigateTo(parent)
  }, [currentPath, navigateTo])

  const handleDoubleClick = useCallback((entry: FileEntry) => {
    if (entry.type === 'directory') {
      const newPath = currentPath === '/' ? `/${entry.name}` : `${currentPath}/${entry.name}`
      navigateTo(newPath)
    }
    // TODO: file preview for text files
  }, [currentPath, navigateTo])

  // Selection
  const handleSelect = useCallback((name: string, e: React.MouseEvent) => {
    setContextMenu(null)
    if (e.ctrlKey || e.metaKey) {
      setSelectedNames(prev => { const s = new Set(prev); s.has(name) ? s.delete(name) : s.add(name); return s })
    } else {
      setSelectedNames(new Set([name]))
    }
  }, [])

  // Sorting
  const handleSort = useCallback((key: SortKey) => {
    if (sortBy === key) setSortOrder(prev => prev === 'asc' ? 'desc' : 'asc')
    else { setSortBy(key); setSortOrder('asc') }
  }, [sortBy])

  const sortedEntries = [...entries].sort((a, b) => {
    // Directories first
    if (a.type === 'directory' && b.type !== 'directory') return -1
    if (a.type !== 'directory' && b.type === 'directory') return 1
    let cmp = 0
    switch (sortBy) {
      case 'name': cmp = a.name.localeCompare(b.name); break
      case 'size': cmp = a.size - b.size; break
      case 'mtime': cmp = a.mtime.localeCompare(b.mtime); break
    }
    return sortOrder === 'asc' ? cmp : -cmp
  })

  // Context menu
  const handleContextMenu = useCallback((e: React.MouseEvent, target: string | null) => {
    e.preventDefault()
    setContextMenu({ x: e.clientX, y: e.clientY, target })
    if (target && !selectedNames.has(target)) {
      setSelectedNames(new Set([target]))
    }
  }, [selectedNames])

  // File operations
  const handleCopy = useCallback(() => {
    if (selectedNames.size === 0) return
    setClipboard({ action: 'copy', sourcePath: currentPath, items: [...selectedNames] })
    setContextMenu(null)
  }, [selectedNames, currentPath])

  const handleCut = useCallback(() => {
    if (selectedNames.size === 0) return
    setClipboard({ action: 'cut', sourcePath: currentPath, items: [...selectedNames] })
    setContextMenu(null)
  }, [selectedNames, currentPath])

  const handlePaste = useCallback(async () => {
    if (!clipboard) return
    setContextMenu(null)
    const endpoint = clipboard.action === 'copy' ? '/api/v1/fs/copy' : '/api/v1/fs/move'
    try {
      for (const item of clipboard.items) {
        await fetch(endpoint, {
          method: 'POST',
          body: JSON.stringify({ source: `${clipboard.sourcePath}/${item}`, destination: `${currentPath}/${item}` }),
        })
      }
      if (clipboard.action === 'cut') setClipboard(null)
      fetchDir(currentPath)
    } catch { /* ignore */ }
  }, [clipboard, currentPath, fetchDir])

  const handleDelete = useCallback(async () => {
    if (selectedNames.size === 0) return
    setContextMenu(null)
    const confirmed = window.confirm(`确认删除 ${selectedNames.size} 个项目？此操作将走风险预演。`)
    if (!confirmed) return
    try {
      for (const name of selectedNames) {
        await authFetch('/api/v1/fs/delete', {
          method: 'POST',
          body: JSON.stringify({ path: `${currentPath}/${name}` }),
        })
      }
      fetchDir(currentPath)
    } catch { /* ignore */ }
  }, [selectedNames, currentPath, fetchDir])

  const handleRenameStart = useCallback(() => {
    const name = [...selectedNames][0]
    if (!name) return
    setContextMenu(null)
    setRenaming(name)
    setRenameValue(name)
  }, [selectedNames])

  const handleRenameSubmit = useCallback(async () => {
    if (!renaming || !renameValue.trim() || renameValue === renaming) {
      setRenaming(null)
      return
    }
    try {
      await authFetch('/api/v1/fs/rename', {
        method: 'POST',
        body: JSON.stringify({ source: `${currentPath}/${renaming}`, destination: `${currentPath}/${renameValue}` }),
      })
      fetchDir(currentPath)
    } catch { /* ignore */ }
    setRenaming(null)
  }, [renaming, renameValue, currentPath, fetchDir])

  const handleMkdir = useCallback(async () => {
    setContextMenu(null)
    const name = window.prompt('新建文件夹名称:')
    if (!name) return
    try {
      await authFetch('/api/v1/fs/mkdir', {
        method: 'POST',
        body: JSON.stringify({ path: `${currentPath}/${name}` }),
      })
      fetchDir(currentPath)
    } catch { /* ignore */ }
  }, [currentPath, fetchDir])

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (!containerRef.current?.contains(document.activeElement) && document.activeElement !== containerRef.current) return
      if (e.key === 'Delete') { handleDelete(); return }
      if (e.key === 'F2') { handleRenameStart(); return }
      if ((e.ctrlKey || e.metaKey) && e.key === 'c') { handleCopy(); e.preventDefault() }
      if ((e.ctrlKey || e.metaKey) && e.key === 'x') { handleCut(); e.preventDefault() }
      if ((e.ctrlKey || e.metaKey) && e.key === 'v') { handlePaste(); e.preventDefault() }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [handleDelete, handleRenameStart, handleCopy, handleCut, handlePaste])

  // Breadcrumb parts
  const pathParts = currentPath === '/' ? ['/'] : ['/', ...currentPath.split('/').filter(Boolean)]

  return (
    <div ref={containerRef} tabIndex={-1} style={{ height: '100%', display: 'flex', flexDirection: 'column', outline: 'none' }} onClick={() => setContextMenu(null)}>
      {/* Toolbar */}
      <div style={{ height: 32, display: 'flex', alignItems: 'center', padding: '0 8px', gap: 4, borderBottom: '1px solid var(--ops-border-subtle)', flexShrink: 0 }}>
        <ToolBtn icon="arrow_back" onClick={navigateUp} title="上级目录" disabled={currentPath === '/'} />
        <ToolBtn icon="refresh" onClick={() => fetchDir(currentPath)} title="刷新" />
        <div style={{ width: 1, height: 16, background: 'var(--ops-border-subtle)', margin: '0 4px' }} />
        {/* Breadcrumb */}
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 2, overflow: 'hidden' }}>
          {pathParts.map((part, i) => (
            <span key={i} style={{ display: 'flex', alignItems: 'center' }}>
              {i > 0 && <span className="material-symbols-outlined" style={{ fontSize: 12, color: 'var(--ops-fg-muted)' }}>chevron_right</span>}
              <button
                onClick={() => {
                  const path = i === 0 ? '/' : '/' + pathParts.slice(1, i + 1).join('/')
                  navigateTo(path)
                }}
                style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 12, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-secondary)', padding: '2px 4px', borderRadius: 3 }}
                onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
                onMouseLeave={(e) => { e.currentTarget.style.background = 'none' }}
              >
                {part}
              </button>
            </span>
          ))}
        </div>
      </div>

      {/* Main content */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* Left sidebar - quick nav */}
        <div style={{ width: 160, borderRight: '1px solid var(--ops-border-subtle)', padding: '8px 0', overflowY: 'auto', flexShrink: 0 }}>
          {['/','/ home','/var','/var/log','/tmp','/etc','/opt','/usr'].map(p => {
            const display = p === '/' ? '/' : p.split('/').pop()!
            const fullPath = p.replace('/ ', '/')
            return (
              <button
                key={p}
                onClick={() => navigateTo(fullPath)}
                style={{
                  width: '100%', height: 26, display: 'flex', alignItems: 'center', gap: 6, padding: '0 12px',
                  border: 'none', cursor: 'pointer', fontSize: 12, fontFamily: 'var(--ops-font-ui)',
                  color: currentPath === fullPath ? 'var(--ops-fg-primary)' : 'var(--ops-fg-secondary)',
                  background: currentPath === fullPath ? 'var(--ops-bg-elevated)' : 'transparent',
                }}
                onMouseEnter={(e) => { if (currentPath !== fullPath) e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
                onMouseLeave={(e) => { if (currentPath !== fullPath) e.currentTarget.style.background = 'transparent' }}
              >
                <span className="material-symbols-outlined" style={{ fontSize: 14 }}>folder</span>
                {display}
              </button>
            )
          })}
        </div>

        {/* File list */}
        <div
          style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          onContextMenu={(e) => handleContextMenu(e, null)}
        >
          {/* Table header */}
          <div style={{ height: 28, display: 'flex', alignItems: 'center', padding: '0 8px', borderBottom: '1px solid var(--ops-border-subtle)', flexShrink: 0, gap: 0 }}>
            <ColHeader label="名称" width="flex" sortKey="name" current={sortBy} order={sortOrder} onSort={handleSort} />
            <ColHeader label="修改时间" width={140} sortKey="mtime" current={sortBy} order={sortOrder} onSort={handleSort} />
            <ColHeader label="大小" width={80} sortKey="size" current={sortBy} order={sortOrder} onSort={handleSort} />
            <div style={{ width: 80, fontSize: 11, color: 'var(--ops-fg-muted)', padding: '0 4px' }}>权限</div>
          </div>

          {/* Rows */}
          <div style={{ flex: 1, overflowY: 'auto' }}>
            {loading && <div style={{ padding: 16, fontSize: 12, color: 'var(--ops-fg-muted)' }}>加载中...</div>}
            {error && <div style={{ padding: 16, fontSize: 12, color: 'var(--ops-status-danger)' }}>{error}</div>}
            {!loading && !error && sortedEntries.length === 0 && (
              <div style={{ padding: 16, fontSize: 12, color: 'var(--ops-fg-muted)' }}>空目录</div>
            )}
            {sortedEntries.map(entry => (
              <FileRow
                key={entry.name}
                entry={entry}
                selected={selectedNames.has(entry.name)}
                renaming={renaming === entry.name}
                renameValue={renameValue}
                onRenameChange={setRenameValue}
                onRenameSubmit={handleRenameSubmit}
                onClick={(e) => handleSelect(entry.name, e)}
                onDoubleClick={() => handleDoubleClick(entry)}
                onContextMenu={(e) => handleContextMenu(e, entry.name)}
              />
            ))}
          </div>

          {/* Status bar */}
          <div style={{ height: 24, display: 'flex', alignItems: 'center', padding: '0 8px', borderTop: '1px solid var(--ops-border-subtle)', flexShrink: 0 }}>
            <span style={{ fontSize: 11, color: 'var(--ops-fg-muted)', fontFamily: 'var(--ops-font-mono)' }}>
              {entries.length} 项{selectedNames.size > 0 && ` · 已选 ${selectedNames.size}`}
              {clipboard && ` · 剪贴板: ${clipboard.items.length} 项 (${clipboard.action === 'copy' ? '复制' : '剪切'})`}
            </span>
          </div>
        </div>
      </div>

      {/* Context menu */}
      {contextMenu && (
        <ContextMenuOverlay
          x={contextMenu.x}
          y={contextMenu.y}
          hasTarget={contextMenu.target !== null}
          hasClipboard={clipboard !== null}
          onCopy={handleCopy}
          onCut={handleCut}
          onPaste={handlePaste}
          onDelete={handleDelete}
          onRename={handleRenameStart}
          onMkdir={handleMkdir}
          onRefresh={() => { setContextMenu(null); fetchDir(currentPath) }}
          onClose={() => setContextMenu(null)}
        />
      )}
    </div>
  )
}

// --- Sub-components ---

const ColHeader: FC<{ label: string; width: number | 'flex'; sortKey: SortKey; current: SortKey; order: 'asc' | 'desc'; onSort: (k: SortKey) => void }> = ({ label, width, sortKey, current, order, onSort }) => (
  <button
    onClick={() => onSort(sortKey)}
    style={{
      ...(width === 'flex' ? { flex: 1 } : { width }),
      height: '100%', display: 'flex', alignItems: 'center', gap: 2, padding: '0 4px',
      border: 'none', background: 'none', cursor: 'pointer',
      fontSize: 11, color: current === sortKey ? 'var(--ops-fg-primary)' : 'var(--ops-fg-muted)',
    }}
  >
    {label}
    {current === sortKey && <span style={{ fontSize: 10 }}>{order === 'asc' ? '▲' : '▼'}</span>}
  </button>
)

const FileRow: FC<{
  entry: FileEntry
  selected: boolean
  renaming: boolean
  renameValue: string
  onRenameChange: (v: string) => void
  onRenameSubmit: () => void
  onClick: (e: React.MouseEvent) => void
  onDoubleClick: () => void
  onContextMenu: (e: React.MouseEvent) => void
}> = ({ entry, selected, renaming, renameValue, onRenameChange, onRenameSubmit, onClick, onDoubleClick, onContextMenu }) => {
  const icon = entry.type === 'directory' ? 'folder' : entry.type === 'symlink' ? 'link' : 'description'
  const iconColor = entry.type === 'directory' ? '#5B9BD5' : 'var(--ops-fg-muted)'

  return (
    <div
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      onContextMenu={onContextMenu}
      style={{
        height: 28,
        display: 'flex',
        alignItems: 'center',
        padding: '0 8px',
        cursor: 'pointer',
        background: selected ? 'rgba(55,148,253,0.12)' : 'transparent',
        borderBottom: '1px solid var(--ops-border-subtle)',
      }}
      onMouseEnter={(e) => { if (!selected) e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
      onMouseLeave={(e) => { if (!selected) e.currentTarget.style.background = 'transparent' }}
    >
      {/* Name column */}
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 6, overflow: 'hidden' }}>
        <span className="material-symbols-outlined" style={{ fontSize: 16, color: iconColor, flexShrink: 0 }}>{icon}</span>
        {renaming ? (
          <input
            autoFocus
            value={renameValue}
            onChange={(e) => onRenameChange(e.target.value)}
            onBlur={onRenameSubmit}
            onKeyDown={(e) => { if (e.key === 'Enter') onRenameSubmit(); if (e.key === 'Escape') onRenameChange(entry.name) }}
            onClick={(e) => e.stopPropagation()}
            style={{ flex: 1, height: 20, fontSize: 12, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-primary)', background: 'var(--ops-bg-canvas)', border: '1px solid var(--ops-border-default)', borderRadius: 2, padding: '0 4px', outline: 'none' }}
          />
        ) : (
          <span style={{ fontSize: 12, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {entry.name}
          </span>
        )}
      </div>
      {/* Mtime */}
      <span style={{ width: 140, fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)', padding: '0 4px' }}>
        {entry.mtime ? formatMtime(entry.mtime) : '--'}
      </span>
      {/* Size */}
      <span style={{ width: 80, fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)', padding: '0 4px', textAlign: 'right' }}>
        {entry.type === 'directory' ? '--' : formatSize(entry.size)}
      </span>
      {/* Permissions */}
      <span style={{ width: 80, fontSize: 11, fontFamily: 'var(--ops-font-mono)', color: 'var(--ops-fg-muted)', padding: '0 4px' }}>
        {entry.permissions || '--'}
      </span>
    </div>
  )
}

const ContextMenuOverlay: FC<{
  x: number; y: number; hasTarget: boolean; hasClipboard: boolean
  onCopy: () => void; onCut: () => void; onPaste: () => void
  onDelete: () => void; onRename: () => void; onMkdir: () => void
  onRefresh: () => void; onClose: () => void
}> = ({ x, y, hasTarget, hasClipboard, onCopy, onCut, onPaste, onDelete, onRename, onMkdir, onRefresh, onClose }) => (
  <>
    <div onClick={onClose} style={{ position: 'fixed', inset: 0, zIndex: 9998 }} />
    <div style={{ position: 'fixed', left: x, top: y, zIndex: 9999, minWidth: 160, background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-default)', borderRadius: 4, padding: '4px 0', boxShadow: '0 4px 16px rgba(0,0,0,0.4)' }}>
      {hasTarget && (
        <>
          <MenuItem label="复制" shortcut="Ctrl+C" onClick={onCopy} />
          <MenuItem label="剪切" shortcut="Ctrl+X" onClick={onCut} />
          <MenuDivider />
          <MenuItem label="重命名" shortcut="F2" onClick={onRename} />
          <MenuItem label="删除" shortcut="Del" onClick={onDelete} danger />
          <MenuDivider />
        </>
      )}
      <MenuItem label="粘贴" shortcut="Ctrl+V" onClick={onPaste} disabled={!hasClipboard} />
      <MenuItem label="新建文件夹" onClick={onMkdir} />
      <MenuDivider />
      <MenuItem label="刷新" onClick={onRefresh} />
    </div>
  </>
)

const MenuItem: FC<{ label: string; shortcut?: string; onClick: () => void; disabled?: boolean; danger?: boolean }> = ({ label, shortcut, onClick, disabled, danger }) => (
  <button
    onClick={disabled ? undefined : onClick}
    style={{
      width: '100%', height: 28, display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 12px',
      border: 'none', cursor: disabled ? 'default' : 'pointer', fontSize: 12, fontFamily: 'var(--ops-font-ui)',
      color: disabled ? 'var(--ops-fg-muted)' : danger ? 'var(--ops-status-danger)' : 'var(--ops-fg-primary)',
      background: 'transparent', opacity: disabled ? 0.5 : 1,
    }}
    onMouseEnter={(e) => { if (!disabled) e.currentTarget.style.background = 'var(--ops-bg-surface)' }}
    onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
  >
    <span>{label}</span>
    {shortcut && <span style={{ fontSize: 11, color: 'var(--ops-fg-muted)' }}>{shortcut}</span>}
  </button>
)

const MenuDivider: FC = () => <div style={{ height: 1, background: 'var(--ops-border-subtle)', margin: '4px 8px' }} />

// --- Utils ---
function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function formatMtime(iso: string): string {
  try {
    const d = new Date(iso)
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
  } catch { return iso }
}

const ToolBtn: FC<{ icon: string; onClick: () => void; title: string; disabled?: boolean }> = ({ icon, onClick, title, disabled }) => (
  <button
    onClick={disabled ? undefined : onClick}
    title={title}
    style={{
      width: 26, height: 26, display: 'flex', alignItems: 'center', justifyContent: 'center',
      border: 'none', borderRadius: 4, cursor: disabled ? 'default' : 'pointer',
      background: 'transparent', color: disabled ? 'var(--ops-fg-muted)' : 'var(--ops-fg-secondary)',
      opacity: disabled ? 0.4 : 1,
    }}
    onMouseEnter={(e) => { if (!disabled) e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
    onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
  >
    <span className="material-symbols-outlined" style={{ fontSize: 16 }}>{icon}</span>
  </button>
)
