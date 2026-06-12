// Data: POST /api/v1/chat/stream
// Supports multiple concurrent SSE connections — each send() creates an independent stream
import { useCallback, useRef } from 'react'
import { authFetch } from '../lib/auth'
import type {
  SSEEventType,
  SSEStartData,
  SSESenseData,
  SSEModeDecisionData,
  SSEAnalyzeData,
  SSEPlanData,
  SSEExecuteData,
  SSEExecuteDoneData,
  SSEOutputData,
  SSEErrorData,
  SSEDoneData,
  SSEAgentRoleData,
  SSEVerifierResultData,
  SSEPermissionRequestData,
  SSECircuitOpenData,
  SSEOutputPersistedData,
  SSEPlanReadyData,
  SSEWarningData,
} from '../types/api'

export type SSEEventHandler = {
  onStart?: (data: SSEStartData) => void
  onSense?: (data: SSESenseData) => void
  onModeDecision?: (data: SSEModeDecisionData) => void
  onAnalyze?: (data: SSEAnalyzeData) => void
  onPlan?: (data: SSEPlanData) => void
  onExecute?: (data: SSEExecuteData) => void
  onExecuteDone?: (data: SSEExecuteDoneData) => void
  onTextDelta?: (data: { delta: string; round: number }) => void
  onReasoningDelta?: (data: { delta: string; round: number }) => void
  onOutput?: (data: SSEOutputData) => void
  onError?: (data: SSEErrorData) => void
  onDone?: (data: SSEDoneData) => void
  onAgentRole?: (data: SSEAgentRoleData) => void
  onVerifierResult?: (data: SSEVerifierResultData) => void
  onPermissionRequest?: (data: SSEPermissionRequestData) => void
  onCircuitOpen?: (data: SSECircuitOpenData) => void
  onOutputPersisted?: (data: SSEOutputPersistedData) => void
  onPlanReady?: (data: SSEPlanReadyData) => void
  onWarning?: (data: SSEWarningData) => void
  onConnectionError?: (error: Error) => void
}

export function useSSE(handlers: SSEEventHandler) {
  // Track all active connections by sessionId
  const connectionsRef = useRef<Map<string, AbortController>>(new Map())
  const handlersRef = useRef(handlers)
  handlersRef.current = handlers

  const send = useCallback(async (message: string, sessionId?: string) => {
    const connId = sessionId || `anon-${Date.now()}`

    // If there's already an active connection for THIS session, abort it (retry scenario)
    const existing = connectionsRef.current.get(connId)
    if (existing) {
      existing.abort()
      connectionsRef.current.delete(connId)
    }

    const controller = new AbortController()
    connectionsRef.current.set(connId, controller)

    try {
      const body: Record<string, string> = { message }
      if (sessionId) body.session_id = sessionId

      const response = await authFetch('/api/v1/chat/stream', {
        method: 'POST',
        body: JSON.stringify(body),
        signal: controller.signal,
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body?.getReader()
      if (!reader) throw new Error('No response body')

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        let currentEvent: SSEEventType | null = null

        for (const line of lines) {
          if (line.startsWith('event:')) {
            currentEvent = line.slice(6).trim() as SSEEventType
          } else if (line.startsWith('data:') && currentEvent) {
            const dataStr = line.slice(5).trim()
            if (!dataStr) continue
            try {
              const data = JSON.parse(dataStr)
              dispatchEvent(currentEvent, data, handlersRef.current)
            } catch {
              // Skip malformed JSON
            }
            currentEvent = null
          } else if (line === '') {
            currentEvent = null
          }
        }
      }
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        handlersRef.current.onConnectionError?.(err as Error)
      }
    } finally {
      connectionsRef.current.delete(connId)
    }
  }, [])

  const abort = useCallback((sessionId?: string) => {
    if (sessionId) {
      // Abort specific session's connection
      const controller = connectionsRef.current.get(sessionId)
      if (controller) {
        controller.abort()
        connectionsRef.current.delete(sessionId)
      }
    } else {
      // Abort ALL connections
      for (const [id, controller] of connectionsRef.current) {
        controller.abort()
        connectionsRef.current.delete(id)
      }
    }
  }, [])

  return { send, abort }
}

function dispatchEvent(
  event: SSEEventType,
  data: unknown,
  handlers: SSEEventHandler
) {
  switch (event) {
    case 'start':
      handlers.onStart?.(data as SSEStartData)
      break
    case 'sense':
      handlers.onSense?.(data as SSESenseData)
      break
    case 'mode_decision':
      handlers.onModeDecision?.(data as SSEModeDecisionData)
      break
    case 'analyze':
      handlers.onAnalyze?.(data as SSEAnalyzeData)
      break
    case 'plan':
      handlers.onPlan?.(data as SSEPlanData)
      break
    case 'execute':
      handlers.onExecute?.(data as SSEExecuteData)
      break
    case 'execute_done':
      handlers.onExecuteDone?.(data as SSEExecuteDoneData)
      break
    case 'text_delta':
      handlers.onTextDelta?.(data as { delta: string; round: number })
      break
    case 'reasoning_delta':
      handlers.onReasoningDelta?.(data as { delta: string; round: number })
      break
    case 'output':
      handlers.onOutput?.(data as SSEOutputData)
      break
    case 'error':
      handlers.onError?.(data as SSEErrorData)
      break
    case 'done':
      handlers.onDone?.(data as SSEDoneData)
      break
    case 'agent_role':
      handlers.onAgentRole?.(data as SSEAgentRoleData)
      break
    case 'verifier_result':
      handlers.onVerifierResult?.(data as SSEVerifierResultData)
      break
    case 'permission_request':
      handlers.onPermissionRequest?.(data as SSEPermissionRequestData)
      break
    case 'circuit_open':
      handlers.onCircuitOpen?.(data as SSECircuitOpenData)
      break
    case 'output_persisted':
      handlers.onOutputPersisted?.(data as SSEOutputPersistedData)
      break
    case 'plan_ready':
      handlers.onPlanReady?.(data as SSEPlanReadyData)
      break
    case 'warning':
      handlers.onWarning?.(data as SSEWarningData)
      break
  }
}
