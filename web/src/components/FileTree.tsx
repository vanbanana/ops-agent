import { authFetch } from '../lib/auth'
import { type FC, useState, useCallback } from 'react'

interface FileEntry {
  name: string
  type: 'file' | 'directory' | 'symlink'
  size: number
  mtime: string
  permissions: string
}

interface TreeNode {
  entry: FileEntry
  path: string
  children: TreeNode[] | null // null = not loaded yet
  expanded: boolean
  loading: boolean
}

export const FileTree: FC = () => {
  const [rootPath] = useState('/')
  const [nodes, setNodes] = useState<TreeNode[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [initialized, setInitialized] = useState(false)

  const loadDir = useCallback(async (path: string): Promise<TreeNode[]> => {
    const res = await authFetch(`/api/v1/fs/list?path=${encodeURIComponent(path)}`)
    const data = await res.json()
    if (data.code !== 0) throw new Error(data.error || 'load failed')
    const entries: FileEntry[] = data.entries || []
    // Sort: directories first, then files, alphabetical
    entries.sort((a, b) => {
      if (a.type === 'directory' && b.type !== 'directory') return -1
      if (a.type !== 'directory' && b.type === 'directory') return 1
      return a.name.localeCompare(b.name)
    })
    return entries.map(e => ({
      entry: e,
      path: path === '/' ? '/' + e.name : path + '/' + e.name,
      children: e.type === 'directory' ? null : [],
      expanded: false,
      loading: false,
    }))
  }, [])

  // Load root on first render
  if (!initialized) {
    setInitialized(true)
    setLoading(true)
    loadDir(rootPath)
      .then(n => { setNodes(n); setLoading(false) })
      .catch(e => { setError(String(e)); setLoading(false) })
  }

  const toggleDir = async (targetPath: string) => {
    const toggle = async (list: TreeNode[]): Promise<TreeNode[]> => {
      const result: TreeNode[] = []
      for (const node of list) {
        if (node.path === targetPath) {
          if (node.expanded) {
            result.push({ ...node, expanded: false })
          } else {
            if (node.children === null) {
              // Need to load
              result.push({ ...node, loading: true })
              try {
                const children = await loadDir(node.path)
                result[result.length - 1] = { ...node, expanded: true, loading: false, children }
              } catch {
                result[result.length - 1] = { ...node, loading: false, children: [] }
              }
            } else {
              result.push({ ...node, expanded: true })
            }
          }
        } else if (node.children && node.children.length > 0 && node.expanded) {
          const newChildren = await toggle(node.children)
          result.push({ ...node, children: newChildren })
        } else {
          result.push(node)
        }
      }
      return result
    }
    const updated = await toggle(nodes)
    setNodes(updated)
  }

  return (
    <div style={{ width: 200, height: '100%', display: 'flex', flexDirection: 'column', background: 'var(--ops-bg-surface)', borderRight: '1px solid var(--ops-border-subtle)', overflow: 'hidden', flexShrink: 0 }}>
      {/* Header */}
      <div style={{ height: 32, display: 'flex', alignItems: 'center', padding: '0 10px', borderBottom: '1px solid var(--ops-border-subtle)', flexShrink: 0 }}>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 500, color: 'var(--ops-fg-muted)', flex: 1 }}>文件浏览</span>
        <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 9, color: 'var(--ops-fg-muted)' }}>/</span>
      </div>
      {/* Tree */}
      <div style={{ flex: 1, overflow: 'auto', padding: '4px 0' }}>
        {loading && <div style={{ padding: '8px 10px', fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)' }}>加载中...</div>}
        {error && <div style={{ padding: '8px 10px', fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-status-danger)' }}>{error}</div>}
        {nodes.map(node => (
          <TreeNodeItem key={node.path} node={node} depth={0} onToggle={toggleDir} />
        ))}
      </div>
    </div>
  )
}

const TreeNodeItem: FC<{ node: TreeNode; depth: number; onToggle: (path: string) => void }> = ({ node, depth, onToggle }) => {
  const isDir = node.entry.type === 'directory'
  const indent = 8 + depth * 12

  const icon = isDir
    ? (node.expanded ? 'folder_open' : 'folder')
    : getFileIcon(node.entry.name)

  const iconColor = isDir ? '#e8ab53' : 'var(--ops-fg-muted)'

  return (
    <>
      <button
        onClick={() => isDir && onToggle(node.path)}
        style={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          padding: `2px 6px 2px ${indent}px`,
          border: 'none',
          background: 'transparent',
          cursor: isDir ? 'pointer' : 'default',
          textAlign: 'left',
          borderRadius: 2,
        }}
        onMouseEnter={e => { e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
        onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
      >
        {isDir && (
          <span className="material-symbols-outlined" style={{ fontSize: 10, color: 'var(--ops-fg-muted)', width: 10 }}>
            {node.expanded ? 'expand_more' : 'chevron_right'}
          </span>
        )}
        {!isDir && <span style={{ width: 10 }} />}
        <span className="material-symbols-outlined" style={{ fontSize: 13, color: iconColor }}>{icon}</span>
        <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
          {node.entry.name}
        </span>
        {node.loading && <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 9, color: 'var(--ops-fg-muted)' }}>...</span>}
      </button>
      {node.expanded && node.children && node.children.map(child => (
        <TreeNodeItem key={child.path} node={child} depth={depth + 1} onToggle={onToggle} />
      ))}
    </>
  )
}

function getFileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase()
  switch (ext) {
    case 'go': return 'code'
    case 'ts': case 'tsx': case 'js': case 'jsx': return 'javascript'
    case 'json': return 'data_object'
    case 'md': return 'description'
    case 'yaml': case 'yml': case 'toml': return 'settings'
    case 'sql': return 'storage'
    case 'sh': case 'bash': return 'terminal'
    case 'log': return 'article'
    case 'env': return 'lock'
    default: return 'draft'
  }
}
