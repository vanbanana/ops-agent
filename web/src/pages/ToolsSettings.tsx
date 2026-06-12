import { authFetch } from '../lib/auth'
import { type FC, useState, useEffect } from 'react'

interface ToolItem {
  name: string
  description: string
  type: 'readonly' | 'write' | 'external'
  enabled: boolean
  source: 'builtin' | 'mcp'
}

interface MCPServer {
  id: string
  name: string
  transport: string
  command: string
  args: string[]
  url: string
  is_active: boolean
}

export const ToolsSettings: FC = () => {
  const [tools, setTools] = useState<ToolItem[]>([])
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      authFetch('/api/v1/tools/status').then(r => r.json()),
      authFetch('/api/v1/mcp/servers').then(r => r.json()),
    ])
      .then(([toolsData, mcpData]) => {
        if (toolsData?.data) setTools(toolsData.data)
        if (mcpData?.data) setMcpServers(mcpData.data)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const handleToggle = async (name: string, enabled: boolean) => {
    const res = await authFetch('/api/v1/tools/toggle', {
      method: 'POST',
      body: JSON.stringify({ name, enabled }),
    })
    if (res.ok) {
      setTools(prev => prev.map(t => t.name === name ? { ...t, enabled } : t))
    }
  }

  const readonlyTools = tools.filter(t => t.type === 'readonly')
  const writeTools = tools.filter(t => t.type === 'write')
  const externalTools = tools.filter(t => t.type === 'external')

  if (loading) {
    return <div style={{ padding: 48, fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>加载中...</div>
  }

  return (
    <div style={{ flex: 1, overflow: 'auto', background: 'var(--ops-bg-canvas)', padding: '24px 48px' }}>
      <h2 style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 16, fontWeight: 600, color: 'var(--ops-fg-primary)', margin: '0 0 8px 0', display: 'flex', alignItems: 'center', gap: 8 }}>
        <span className="material-symbols-outlined" style={{ fontSize: 18 }}>extension</span>
        MCP 工具管理
      </h2>
      <p style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)', margin: '0 0 20px 0' }}>
        基于 MCP (Model Context Protocol) 协议的插件化工具。启用的工具会被 Agent 在对话中自动调用。
      </p>

      {/* MCP Servers */}
      <div style={{ marginBottom: 24, padding: 16, background: 'var(--ops-bg-elevated)', borderRadius: 6, border: '1px solid var(--ops-border-subtle)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 12 }}>
          <span className="material-symbols-outlined" style={{ fontSize: 16, color: 'var(--ops-fg-primary)' }}>power</span>
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 13, fontWeight: 500, color: 'var(--ops-fg-primary)' }}>外部 MCP Server</span>
          <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>({mcpServers.length})</span>
        </div>
        <p style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', margin: '0 0 12px 0' }}>
          通过 MCP 协议连接的外部工具服务器。Agent 可调用其提供的工具扩展能力。
        </p>
        {mcpServers.map(srv => (
          <div key={srv.id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '6px 0', borderBottom: '1px solid var(--ops-border-subtle)' }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: srv.is_active ? '#34c759' : 'var(--ops-fg-muted)', flexShrink: 0 }} />
            <div style={{ flex: 1, minWidth: 0 }}>
              <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-primary)' }}>{srv.name}</span>
              <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 9, color: 'var(--ops-fg-muted)', marginLeft: 8 }}>
                {srv.transport === 'stdio' ? `${srv.command} ${(srv.args || []).join(' ')}` : srv.url}
              </span>
            </div>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 9, color: srv.is_active ? '#34c759' : 'var(--ops-fg-muted)' }}>
              {srv.is_active ? '已连接' : '未激活'}
            </span>
          </div>
        ))}
        {mcpServers.length === 0 && (
          <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, color: 'var(--ops-fg-muted)', padding: '8px 0' }}>暂无外部 MCP Server</div>
        )}
      </div>

      {/* Stats */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 20 }}>
        <StatBadge label="总计" value={tools.length} color="var(--ops-fg-secondary)" />
        <StatBadge label="已启用" value={tools.filter(t => t.enabled).length} color="#34c759" />
        <StatBadge label="只读探针" value={readonlyTools.length} color="var(--ops-fg-muted)" />
        <StatBadge label="写操作" value={writeTools.length} color="var(--ops-status-warn)" />
      </div>

      {/* Readonly probes */}
      <ToolGroup title="系统探针 (只读)" icon="sensors" tools={readonlyTools} onToggle={handleToggle} />
      {/* Write tools */}
      <ToolGroup title="受控写操作" icon="edit_note" tools={writeTools} onToggle={handleToggle} />
      {/* External/MCP */}
      {externalTools.length > 0 && <ToolGroup title="外部 MCP 工具" icon="power" tools={externalTools} onToggle={handleToggle} />}
    </div>
  )
}

const StatBadge: FC<{ label: string; value: number; color: string }> = ({ label, value, color }) => (
  <div style={{ padding: '6px 12px', borderRadius: 6, background: 'var(--ops-bg-elevated)', border: '1px solid var(--ops-border-subtle)', display: 'flex', alignItems: 'center', gap: 6 }}>
    <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 14, fontWeight: 600, color }}>{value}</span>
    <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>{label}</span>
  </div>
)

const ToolGroup: FC<{ title: string; icon: string; tools: ToolItem[]; onToggle: (name: string, enabled: boolean) => void }> = ({ title, icon, tools: toolList, onToggle }) => (
  <div style={{ marginBottom: 20 }}>
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
      <span className="material-symbols-outlined" style={{ fontSize: 14, color: 'var(--ops-fg-muted)' }}>{icon}</span>
      <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, fontWeight: 500, color: 'var(--ops-fg-primary)' }}>{title}</span>
      <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 10, color: 'var(--ops-fg-muted)' }}>({toolList.length})</span>
    </div>
    <div style={{ border: '1px solid var(--ops-border-subtle)', borderRadius: 6, overflow: 'hidden' }}>
      {toolList.map((tool, i) => (
        <div key={tool.name} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px', borderBottom: i < toolList.length - 1 ? '1px solid var(--ops-border-subtle)' : 'none' }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 11, color: 'var(--ops-fg-primary)', marginBottom: 2 }}>{tool.name}</div>
            <div style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, color: 'var(--ops-fg-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{tool.description}</div>
          </div>
          <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 9, color: 'var(--ops-fg-muted)', padding: '2px 5px', background: 'var(--ops-bg-canvas)', borderRadius: 3 }}>{tool.source}</span>
          {/* Toggle switch */}
          <button
            onClick={() => onToggle(tool.name, !tool.enabled)}
            style={{
              width: 32, height: 18, borderRadius: 9, border: 'none', cursor: 'pointer', position: 'relative',
              background: tool.enabled ? '#34c759' : 'var(--ops-border-default)',
              transition: 'background 0.2s',
            }}
          >
            <div style={{
              width: 14, height: 14, borderRadius: '50%', background: '#fff', position: 'absolute', top: 2,
              left: tool.enabled ? 16 : 2, transition: 'left 0.2s',
            }} />
          </button>
        </div>
      ))}
    </div>
  </div>
)
