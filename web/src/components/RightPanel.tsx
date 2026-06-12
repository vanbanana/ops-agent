// Data: SSE events (推理链路) + POST /desktop/probe/* (系统监控)
// 右侧面板: 系统监控(默认) | 推理链路(对话模式) | 快捷命令(终端模式)
import { type FC, useState, useEffect } from 'react'
import { ReasoningPanel } from './ReasoningPanel'
import { MonitorPanel } from './MonitorPanel'
import type { ReasoningStep, ResourceData, HealthResponse } from '../types/api'

interface RightPanelProps {
  reasoning: ReasoningStep[]
  resources: ResourceData
  health: HealthResponse | null
  healthLoading: boolean
  isStreaming: boolean
  onClose: () => void
  pageMode?: string
}

type Tab = 'monitor' | 'reasoning' | 'commands'

// 快捷命令分类
interface CommandGroup {
  category: string
  icon: string
  commands: { label: string; cmd: string; desc: string }[]
}

const QUICK_COMMANDS: CommandGroup[] = [
  {
    category: '系统状态',
    icon: 'monitor_heart',
    commands: [
      { label: '系统信息', cmd: 'uname -a', desc: '内核版本、主机名' },
      { label: '运行时间', cmd: 'uptime', desc: '系统运行时间和负载' },
      { label: '内存使用', cmd: 'free -h', desc: '内存和 swap 使用情况' },
      { label: 'CPU 信息', cmd: 'lscpu | head -20', desc: 'CPU 核心数和架构' },
      { label: '负载详情', cmd: 'cat /proc/loadavg', desc: '1/5/15分钟平均负载' },
    ],
  },
  {
    category: '磁盘存储',
    icon: 'storage',
    commands: [
      { label: '磁盘使用', cmd: 'df -h', desc: '各分区使用情况' },
      { label: '大文件', cmd: 'du -sh /* 2>/dev/null | sort -rh | head -10', desc: '根目录下最大的10个目录' },
      { label: 'inode', cmd: 'df -i', desc: 'inode 使用率' },
      { label: '挂载信息', cmd: 'mount | grep -v tmpfs', desc: '当前挂载点' },
    ],
  },
  {
    category: '网络',
    icon: 'lan',
    commands: [
      { label: 'IP 地址', cmd: 'ip addr show | grep inet', desc: '所有网络接口 IP' },
      { label: '监听端口', cmd: 'ss -tlnp', desc: 'TCP 监听端口和进程' },
      { label: '连接统计', cmd: 'ss -s', desc: 'TCP/UDP 连接数统计' },
      { label: '路由表', cmd: 'ip route', desc: '路由表信息' },
      { label: 'DNS', cmd: 'cat /etc/resolv.conf', desc: 'DNS 配置' },
    ],
  },
  {
    category: '进程',
    icon: 'apps',
    commands: [
      { label: 'TOP 进程', cmd: 'ps aux --sort=-%mem | head -15', desc: '按内存排序前15个进程' },
      { label: 'CPU TOP', cmd: 'ps aux --sort=-%cpu | head -15', desc: '按 CPU 排序前15个进程' },
      { label: '进程树', cmd: 'ps -ejH | head -40', desc: '进程层级关系' },
      { label: '僵尸进程', cmd: 'ps aux | grep Z | grep -v grep', desc: '查找僵尸进程' },
    ],
  },
  {
    category: '服务管理',
    icon: 'settings_applications',
    commands: [
      { label: '运行服务', cmd: 'systemctl list-units --type=service --state=running', desc: '当前运行的服务' },
      { label: '失败服务', cmd: 'systemctl --failed', desc: '启动失败的服务' },
      { label: 'nginx 状态', cmd: 'systemctl status nginx', desc: 'nginx 服务状态' },
      { label: 'mysql 状态', cmd: 'systemctl status mysql', desc: 'mysql 服务状态' },
      { label: 'docker 状态', cmd: 'systemctl status docker', desc: 'docker 服务状态' },
    ],
  },
  {
    category: '日志',
    icon: 'article',
    commands: [
      { label: '系统日志', cmd: 'journalctl -n 50 --no-pager', desc: '最近50条系统日志' },
      { label: '错误日志', cmd: 'journalctl -p err -n 30 --no-pager', desc: '最近30条错误' },
      { label: '今日日志', cmd: 'journalctl --since today -n 50 --no-pager', desc: '今天的日志' },
      { label: 'dmesg', cmd: 'dmesg | tail -30', desc: '内核环形缓冲区' },
      { label: 'auth 日志', cmd: 'tail -30 /var/log/auth.log', desc: '认证日志' },
    ],
  },
]

export const RightPanel: FC<RightPanelProps> = ({
  reasoning,
  resources,
  isStreaming,
  onClose,
  pageMode,
}) => {
  const isTerminalMode = pageMode === 'terminal'
  const [activeTab, setActiveTab] = useState<Tab>(isTerminalMode ? 'commands' : 'monitor')

  // Auto-switch tab when page mode changes
  useEffect(() => {
    if (pageMode === 'terminal') {
      setActiveTab('commands')
    } else {
      setActiveTab(prev => prev === 'commands' ? 'monitor' : prev)
    }
  }, [pageMode])

  const tabs: Array<{ key: Tab; label: string }> = isTerminalMode
    ? [
        { key: 'commands', label: '快捷命令' },
        { key: 'monitor', label: '系统监控' },
      ]
    : [
        { key: 'monitor', label: '系统监控' },
        { key: 'reasoning', label: '推理链路' },
      ]

  return (
    <aside
      style={{
        width: 256,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--ops-bg-surface)',
        borderLeft: '1px solid var(--ops-border-subtle)',
        flexShrink: 0,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          height: 36,
          padding: '0 10px',
          borderBottom: '1px solid var(--ops-border-subtle)',
          flexShrink: 0,
          gap: 2,
        }}
      >
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            style={{
              height: 24,
              padding: '0 10px',
              borderRadius: 4,
              border: 'none',
              cursor: 'pointer',
              fontFamily: 'var(--ops-font-ui)',
              fontSize: 11,
              fontWeight: activeTab === tab.key ? 500 : 400,
              color: activeTab === tab.key ? 'var(--ops-fg-primary)' : 'var(--ops-fg-muted)',
              background: activeTab === tab.key ? 'var(--ops-bg-elevated)' : 'transparent',
            }}
          >
            {tab.label}
          </button>
        ))}
        <div style={{ flex: 1 }} />
        <button onClick={onClose} style={{ width: 20, height: 20, display: 'flex', alignItems: 'center', justifyContent: 'center', border: 'none', background: 'transparent', cursor: 'pointer', borderRadius: 3, color: 'var(--ops-fg-muted)' }}>
          <span className="material-symbols-outlined" style={{ fontSize: 14 }}>close</span>
        </button>
      </div>

      {/* Content */}
      {activeTab === 'monitor' && (
        <div style={{ flex: 1, overflow: 'auto', padding: '0 12px' }}>
          <MonitorPanel resources={resources} />
        </div>
      )}

      {activeTab === 'reasoning' && (
        <div style={{ flex: 1, overflow: 'auto', padding: '10px 12px' }}>
          <ReasoningPanel steps={reasoning} isStreaming={isStreaming} />
        </div>
      )}

      {activeTab === 'commands' && (
        <div style={{ flex: 1, overflow: 'auto', padding: '8px 0' }}>
          <QuickCommandsPanel />
        </div>
      )}
    </aside>
  )
}

// Quick Commands Panel
const QuickCommandsPanel: FC = () => {
  const [expandedGroup, setExpandedGroup] = useState<string | null>(QUICK_COMMANDS[0].category)

  const handleCopyToTerminal = (cmd: string) => {
    // Dispatch custom event for terminal to pick up and execute
    window.dispatchEvent(new CustomEvent('terminal:exec', { detail: { command: cmd } }))
  }

  return (
    <div>
      <div style={{ padding: '6px 8px 4px', fontFamily: 'var(--ops-font-ui)', fontSize: 9, color: 'var(--ops-fg-muted)', display: 'flex', alignItems: 'center', gap: 4 }}>
        <span className="material-symbols-outlined" style={{ fontSize: 10 }}>info</span>
        点击命令直接在终端执行
      </div>
      {QUICK_COMMANDS.map(group => (
        <div key={group.category}>
          <button
            onClick={() => setExpandedGroup(expandedGroup === group.category ? null : group.category)}
            style={{
              width: '100%',
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '5px 8px',
              border: 'none',
              background: expandedGroup === group.category ? 'var(--ops-bg-elevated)' : 'transparent',
              cursor: 'pointer',
              textAlign: 'left',
            }}
          >
            <span className="material-symbols-outlined" style={{ fontSize: 13, color: 'var(--ops-fg-muted)' }}>{group.icon}</span>
            <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 11, fontWeight: 500, color: 'var(--ops-fg-primary)', flex: 1 }}>{group.category}</span>
            <span className="material-symbols-outlined" style={{ fontSize: 11, color: 'var(--ops-fg-muted)' }}>
              {expandedGroup === group.category ? 'expand_less' : 'expand_more'}
            </span>
          </button>
          {expandedGroup === group.category && (
            <div style={{ padding: '2px 0' }}>
              {group.commands.map(cmd => (
                <button
                  key={cmd.cmd}
                  onClick={() => handleCopyToTerminal(cmd.cmd)}
                  title={cmd.desc}
                  style={{
                    width: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '4px 8px 4px 20px',
                    border: 'none',
                    background: 'transparent',
                    cursor: 'pointer',
                    textAlign: 'left',
                    borderRadius: 3,
                    transition: 'background 0.1s',
                  }}
                  onMouseEnter={e => { e.currentTarget.style.background = 'var(--ops-bg-elevated)' }}
                  onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
                >
                  <span className="material-symbols-outlined" style={{ fontSize: 11, color: '#34c759', flexShrink: 0 }}>play_arrow</span>
                  <span style={{ flex: 1, minWidth: 0, overflow: 'hidden' }}>
                    <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 10, color: 'var(--ops-fg-secondary)', display: 'block', lineHeight: 1.3 }}>{cmd.label}</span>
                    <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 9, color: 'var(--ops-fg-muted)', display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', lineHeight: 1.3 }}>{cmd.cmd}</span>
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
