// Data: POST /api/v1/chat/stream — SSE Event Types

export type SSEEventType =
  | 'start'
  | 'sense'
  | 'mode_decision'
  | 'analyze'
  | 'plan'
  | 'execute'
  | 'execute_done'
  | 'text_delta'
  | 'reasoning_delta'
  | 'output'
  | 'error'
  | 'done'
  | 'agent_role'
  | 'verifier_result'
  | 'permission_request'

export interface SSEStartData {
  trace_id: string
  session_id: string
  mode: 'single' | 'multi'
}

export interface SSESenseData {
  status: 'ok' | 'blocked'
  reason?: string
}

export interface SSEModeDecisionData {
  mode: 'single' | 'multi'
  reason: string
}

export interface SSEAnalyzeData {
  round: number
  has_tool_calls: boolean
  finish_reason: string
  reply_preview: string
}

export interface SSEPlanData {
  round: number
  tools: Array<{ name: string; args: Record<string, unknown> }>
}

export interface SSEExecuteData {
  tool: string
  args: Record<string, unknown>
  security_check: string
}

export interface SSEExecuteDoneData {
  tool: string
  status: string
  result_preview: string
  elapsed_ms: number
}

export interface SSEOutputData {
  reply: string
  tokens_used: number
  elapsed_ms: number
  mode?: string
}

export interface SSEErrorData {
  error_code: string
  message: string
  recoverable: boolean
}

export interface SSEDoneData {
  trace_id: string
  session_id: string
  status: string
}

// Permission request SSE event data
export interface SSEPermissionRequestData {
  request_id: string
  tool: string
  command: string
  risk_level: 'low' | 'medium' | 'high'
  description: string
  expires_at: string
}

// Permission request stored in state
export interface PermissionRequestData {
  request_id: string
  tool: string
  command: string
  risk_level: 'low' | 'medium' | 'high'
  description: string
  expires_at: string
  status: 'pending' | 'allowed' | 'denied' | 'expired'
}

// Health endpoint types
// Data: GET /health
export interface HealthResponse {
  status: 'healthy' | 'degraded' | 'unhealthy'
  components: Record<string, ComponentHealth>
}

export interface ComponentHealth {
  status: 'up' | 'down' | 'degraded'
  latency_ms?: number
  message?: string
}

// Thinking state for analyze/plan phase rendering
export interface ThinkingData {
  status: 'thinking' | 'done'
  analyzeSummary?: string
  planTools?: string[]
}

// Chat message types
export interface ChatMessage {
  id: string
  role: 'user' | 'agent' | 'system'
  content: string
  timestamp: string
  toolCalls?: ToolCallBlock[]
  thinking?: ThinkingData
  reasoning?: string          // accumulated reasoning/thinking content from LLM
  isBlocked?: boolean
  error?: SSEErrorData
  permissionRequest?: PermissionRequestData
}

export interface ToolCallBlock {
  tool: string
  args: Record<string, unknown>
  status: 'running' | 'done' | 'error'
  result_preview?: string
  elapsed_ms?: number
}

// Session types
export interface Session {
  id: string
  title: string
  last_message?: string
  created_at: string
  updated_at: string
}

// Resource strip data
export interface ResourceData {
  disk: { mount: string; percent: number; used: string; total: string }[]
  load: number
  memory: { used: string; total: string; percent: number }
  processes: number
  ports: number
}

// Reasoning timeline
export type ReasoningPhase = 'sense' | 'analyze' | 'plan' | 'execute' | 'output'

export interface ReasoningStep {
  phase: ReasoningPhase
  timestamp: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: any
  status: 'active' | 'done'
}


// Multi-Agent events
export interface SSEAgentRoleData {
  role: 'planner' | 'coordinator' | 'executor' | 'verifier'
  iteration?: number
  action?: string
  message?: string
  sub_task?: string
  executor_id?: number
  total?: number
  parallel?: boolean
}

export interface SSEVerifierResultData {
  verified: boolean
  reason: string
  confidence: number
  missing_info?: string[]
  iteration: number
}

// Multi-Agent chat message (for UI rendering)
export interface AgentChatMessage {
  id: string
  role: 'planner' | 'executor' | 'verifier' | 'coordinator'
  content: string
  timestamp: string
  round: number
  tokensUsed?: number
  toolName?: string
}
