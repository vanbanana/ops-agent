// Demo data for development — 按 frontend-api-contract.md 规则:
// 这些数据仅在后端不可用时用于展示 UI 效果
// 生产环境所有数字来自真实 API
// 注释标记: <!-- DEMO --> 用于区分

import type { ChatMessage, Session, ReasoningStep, ResourceData } from '../types/api'

export const demoSessions: Session[] = [
  {
    id: 'demo-1',
    title: '磁盘空间检查',
    last_message: '/ 87% 接近告警阈值',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'demo-2',
    title: 'Nginx 重启',
    last_message: 'systemctl restart nginx',
    created_at: new Date(Date.now() - 3600000).toISOString(),
    updated_at: new Date(Date.now() - 3600000).toISOString(),
  },
]

export const demoMessages: ChatMessage[] = [
  {
    id: 'demo-msg-1',
    role: 'user',
    content: '看下磁盘还剩多少，load 高不高',
    timestamp: new Date().toISOString(),
  },
  {
    id: 'demo-msg-2',
    role: 'agent',
    content: '根分区 (/) 使用率 **87%**，已超过 80% 告警阈值。5 分钟负载 2.14 略高于阈值。\n\n建议清理 `/var/log` 或扩容根分区。',
    timestamp: new Date().toISOString(),
    toolCalls: [
      { tool: 'probe_disk', args: { timeout: 10 }, status: 'done', result_preview: '/ 87% 3.4G/4.0G', elapsed_ms: 23 },
      { tool: 'probe_top', args: {}, status: 'done', result_preview: 'load avg: 2.14', elapsed_ms: 18 },
    ],
  },
]

export const demoReasoning: ReasoningStep[] = [
  { phase: 'sense', timestamp: '14:32:01', data: { status: 'ok', tokens: 412, elapsed_ms: 12 }, status: 'done' },
  { phase: 'analyze', timestamp: '14:32:01', data: { round: 1, has_tool_calls: true, reply_preview: '磁盘+负载查询', elapsed_ms: 220 }, status: 'done' },
  { phase: 'plan', timestamp: '14:32:01', data: { round: 1, tools: [{ name: 'probe_disk', args: {} }, { name: 'probe_top', args: {} }], elapsed_ms: 434 }, status: 'done' },
  { phase: 'execute', timestamp: '14:32:02', data: { tool: 'probe_disk', elapsed_ms: 23 }, status: 'done' },
  { phase: 'execute', timestamp: '14:32:02', data: { tool: 'probe_top', elapsed_ms: 18 }, status: 'done' },
  { phase: 'output', timestamp: '14:32:02', data: { tokens_used: 286, elapsed_ms: 220 }, status: 'done' },
]

export const demoResources: ResourceData = {
  disk: [
    { mount: '/', percent: 87, used: '3.4G', total: '4.0G' },
    { mount: '/var', percent: 42, used: '8.4G', total: '20G' },
  ],
  load: 2.14,
  memory: { used: '4.2G', total: '8.0G', percent: 52 },
  processes: 37,
  ports: 12,
}
