// Chat state management — messages indexed by session ID (no copy-on-switch)
// Architecture: messagesBySession is the single source of truth.
// state.activeSessionId selects which session to display.
// SSE writes directly to the target session regardless of what's displayed.
import { useReducer, useCallback, useEffect } from 'react'
import type {
  ChatMessage,
  ThinkingData,
  ToolCallBlock,
  Session,
  ResourceData,
  ReasoningStep,
  SSEExecuteDoneData,
  AgentChatMessage,
  PermissionRequestData,
} from '../types/api'

const STORAGE_KEY = 'ops_agent_chat_v2'

interface PersistedState {
  sessions: Session[]
  activeSessionId: string | null
  messagesBySession: Record<string, ChatMessage[]>
}

function loadPersisted(): PersistedState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    return JSON.parse(raw)
  } catch { return null }
}

function persist(data: PersistedState) {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(data)) } catch { /* ignore */ }
}

export interface ChatState {
  sessions: Session[]
  activeSessionId: string | null
  messagesBySession: Record<string, ChatMessage[]>
  isStreaming: boolean
  reasoning: ReasoningStep[]
  resources: ResourceData
  multiAgentMessages: AgentChatMessage[]
  multiAgentMode: boolean
  multiAgentRound: number
  multiAgentActiveRole: string | null
  multiAgentStatus: string
  pendingPermission: PermissionRequestData | null
  permissionMode: 'default' | 'auto_approve'
  contextUsage: number  // 0-100 percent of context window used
}

// Helper: get messages for a session (never undefined)
export function getSessionMessages(state: ChatState, sessionId?: string | null): ChatMessage[] {
  const id = sessionId ?? state.activeSessionId
  if (!id) return []
  return state.messagesBySession[id] ?? []
}

type Action =
  | { type: 'SET_SESSIONS'; sessions: Session[] }
  | { type: 'SET_ACTIVE_SESSION'; id: string | null }
  | { type: 'ADD_MESSAGE'; sessionId: string; message: ChatMessage }
  | { type: 'UPDATE_LAST_AGENT'; sessionId: string; updater: (msg: ChatMessage) => ChatMessage }
  | { type: 'APPEND_DELTA'; sessionId: string; delta: string }
  | { type: 'APPEND_REASONING'; sessionId: string; delta: string }
  | { type: 'ADD_TOOL_CALL'; sessionId: string; toolCall: ToolCallBlock }
  | { type: 'UPDATE_TOOL_CALL'; sessionId: string; tool: string; update: Partial<ToolCallBlock> }
  | { type: 'SET_STREAMING'; streaming: boolean }
  | { type: 'SET_THINKING'; sessionId: string; data: Partial<ThinkingData> }
  | { type: 'ADD_REASONING_STEP'; step: ReasoningStep }
  | { type: 'CLEAR_REASONING' }
  | { type: 'UPDATE_RESOURCES'; data: Partial<ResourceData> }
  | { type: 'SET_BLOCKED'; sessionId: string; messageId: string }
  | { type: 'CREATE_SESSION'; session: Session }
  | { type: 'SET_MULTI_AGENT_MODE'; enabled: boolean }
  | { type: 'ADD_AGENT_MESSAGE'; message: AgentChatMessage }
  | { type: 'SET_MULTI_AGENT_STATUS'; role: string | null; round: number; status: string }
  | { type: 'CLEAR_MULTI_AGENT' }
  | { type: 'SET_PERMISSION_REQUEST'; sessionId: string; data: PermissionRequestData }
  | { type: 'UPDATE_PERMISSION_STATUS'; status: 'allowed' | 'denied' | 'expired' }
  | { type: 'SET_PERMISSION_MODE'; mode: 'default' | 'auto_approve' }
  | { type: 'SET_CONTEXT_USAGE'; percent: number }

const initialResources: ResourceData = {
  disk: [],
  load: -1,
  memory: { used: '--', total: '--', percent: -1 },
  processes: -1,
  ports: -1,
}

const saved = loadPersisted()

const initialState: ChatState = {
  sessions: saved?.sessions ?? [],
  activeSessionId: saved?.activeSessionId ?? null,
  messagesBySession: saved?.messagesBySession ?? {},
  isStreaming: false,
  reasoning: [],
  resources: initialResources,
  multiAgentMessages: [],
  multiAgentMode: false,
  multiAgentRound: 0,
  multiAgentActiveRole: null,
  multiAgentStatus: '',
  pendingPermission: null,
  permissionMode: 'default',
  contextUsage: 0,
}

// Helper to update messages for a specific session
function updateSessionMessages(state: ChatState, sessionId: string, updater: (msgs: ChatMessage[]) => ChatMessage[]): ChatState {
  const current = state.messagesBySession[sessionId] ?? []
  return { ...state, messagesBySession: { ...state.messagesBySession, [sessionId]: updater(current) } }
}

function chatReducer(state: ChatState, action: Action): ChatState {
  switch (action.type) {
    case 'SET_SESSIONS':
      return { ...state, sessions: action.sessions }

    case 'SET_ACTIVE_SESSION':
      return { ...state, activeSessionId: action.id, reasoning: [] }

    case 'ADD_MESSAGE':
      return updateSessionMessages(state, action.sessionId, msgs => [...msgs, action.message])

    case 'APPEND_DELTA':
      return updateSessionMessages(state, action.sessionId, msgs => {
        const updated = [...msgs]
        for (let i = updated.length - 1; i >= 0; i--) {
          if (updated[i].role === 'agent') {
            updated[i] = { ...updated[i], content: updated[i].content + action.delta }
            break
          }
        }
        return updated
      })

    case 'APPEND_REASONING':
      return updateSessionMessages(state, action.sessionId, msgs => {
        const updated = [...msgs]
        for (let i = updated.length - 1; i >= 0; i--) {
          if (updated[i].role === 'agent') {
            updated[i] = { ...updated[i], reasoning: (updated[i].reasoning || '') + action.delta }
            break
          }
        }
        return updated
      })

    case 'UPDATE_LAST_AGENT':
      return updateSessionMessages(state, action.sessionId, msgs => {
        const updated = [...msgs]
        for (let i = updated.length - 1; i >= 0; i--) {
          if (updated[i].role === 'agent') {
            updated[i] = action.updater(updated[i])
            break
          }
        }
        return updated
      })

    case 'ADD_TOOL_CALL':
      return updateSessionMessages(state, action.sessionId, msgs => {
        const updated = [...msgs]
        for (let i = updated.length - 1; i >= 0; i--) {
          if (updated[i].role === 'agent') {
            updated[i] = { ...updated[i], toolCalls: [...(updated[i].toolCalls || []), action.toolCall] }
            break
          }
        }
        return updated
      })

    case 'UPDATE_TOOL_CALL':
      return updateSessionMessages(state, action.sessionId, msgs => {
        const updated = [...msgs]
        for (let i = updated.length - 1; i >= 0; i--) {
          if (updated[i].role === 'agent' && updated[i].toolCalls) {
            updated[i] = { ...updated[i], toolCalls: updated[i].toolCalls!.map(tc => tc.tool === action.tool ? { ...tc, ...action.update } : tc) }
            break
          }
        }
        return updated
      })

    case 'SET_BLOCKED':
      return updateSessionMessages(state, action.sessionId, msgs =>
        msgs.map(m => m.id === action.messageId ? { ...m, isBlocked: true } : m)
      )

    case 'SET_STREAMING':
      return { ...state, isStreaming: action.streaming }

    case 'SET_THINKING':
      return updateSessionMessages(state, action.sessionId, msgs => {
        const updated = [...msgs]
        for (let i = updated.length - 1; i >= 0; i--) {
          if (updated[i].role === 'agent') {
            updated[i] = { ...updated[i], thinking: { ...(updated[i].thinking || { status: 'thinking' }), ...action.data } }
            break
          }
        }
        return updated
      })
    case 'ADD_REASONING_STEP':
      return { ...state, reasoning: [...state.reasoning, action.step] }
    case 'CLEAR_REASONING':
      return { ...state, reasoning: [] }
    case 'UPDATE_RESOURCES':
      return { ...state, resources: { ...state.resources, ...action.data } }

    case 'CREATE_SESSION':
      return { ...state, sessions: [action.session, ...state.sessions], activeSessionId: action.session.id }

    case 'SET_MULTI_AGENT_MODE':
      return { ...state, multiAgentMode: action.enabled }
    case 'ADD_AGENT_MESSAGE':
      return { ...state, multiAgentMessages: [...state.multiAgentMessages, action.message] }
    case 'SET_MULTI_AGENT_STATUS':
      return { ...state, multiAgentActiveRole: action.role, multiAgentRound: action.round, multiAgentStatus: action.status }
    case 'CLEAR_MULTI_AGENT':
      return { ...state, multiAgentMessages: [], multiAgentMode: false, multiAgentRound: 0, multiAgentActiveRole: null, multiAgentStatus: '' }

    case 'SET_PERMISSION_REQUEST':
      return { ...state, pendingPermission: action.data }

    case 'UPDATE_PERMISSION_STATUS':
      if (!state.pendingPermission) return state
      // Once resolved, clear it after brief display
      return { ...state, pendingPermission: null }

    case 'SET_PERMISSION_MODE':
      return { ...state, permissionMode: action.mode }

    case 'SET_CONTEXT_USAGE':
      return { ...state, contextUsage: action.percent }

    default:
      return state
  }
}

export function useChatStore() {
  const [state, dispatch] = useReducer(chatReducer, initialState)

  // Auto-persist on meaningful changes
  useEffect(() => {
    persist({ sessions: state.sessions, activeSessionId: state.activeSessionId, messagesBySession: state.messagesBySession })
  }, [state.sessions, state.activeSessionId, state.messagesBySession])

  const extractResources = useCallback(
    (data: SSEExecuteDoneData) => {
      const preview = data.result_preview
      if (!preview) return
      if (data.tool === 'probe_disk') {
        try { const p = JSON.parse(preview); if (Array.isArray(p)) dispatch({ type: 'UPDATE_RESOURCES', data: { disk: p } }) } catch {}
      }
      if (data.tool === 'probe_top') {
        try { const p = JSON.parse(preview); if (p.load_avg) dispatch({ type: 'UPDATE_RESOURCES', data: { load: p.load_avg[0] ?? p.load_avg } }) } catch {
          const m = preview.match(/load[:\s]+(\d+\.?\d*)/i); if (m) dispatch({ type: 'UPDATE_RESOURCES', data: { load: parseFloat(m[1]) } })
        }
      }
      if (data.tool === 'probe_memory') { try { dispatch({ type: 'UPDATE_RESOURCES', data: { memory: JSON.parse(preview) } }) } catch {} }
      if (data.tool === 'probe_process') { try { const p = JSON.parse(preview); if (p.total !== undefined) dispatch({ type: 'UPDATE_RESOURCES', data: { processes: p.total } }) } catch {} }
      if (data.tool === 'probe_network_connections') { try { const p = JSON.parse(preview); if (p.listening !== undefined) dispatch({ type: 'UPDATE_RESOURCES', data: { ports: p.listening } }) } catch {} }
    },
    []
  )

  return { state, dispatch, extractResources }
}
