// Data: GET /health, POST /api/v1/chat/stream
import { useCallback, useRef, useEffect, useState } from 'react'
import { AppHeader } from './components/AppHeader'
import { ResourceStrip } from './components/ResourceStrip'
import { SideNav } from './components/SideNav'
import { SessionList } from './components/SessionList'
import { ChatMessageComponent } from './components/ChatMessage'
import { ChatInput } from './components/ChatInput'
import { RightPanel } from './components/RightPanel'
import { TerminalDrawer } from './components/TerminalDrawer'
import { StatusBar } from './components/StatusBar'
import { PlaceholderPage } from './components/PlaceholderPage'
import { DesktopMode } from './pages/DesktopMode'
import { SettingsPage } from './pages/SettingsPage'
import { MultiAgentChat } from './components/MultiAgentChat'
import { SuggestedPrompts } from './components/SuggestedPrompts'
import { useHealth } from './hooks/useHealth'
import { useSSE } from './hooks/useSSE'
import { useChatStore, getSessionMessages } from './stores/chatStore'
import { useResourcePolling } from './hooks/useResourcePolling'
import { demoSessions, demoMessages, demoReasoning, demoResources } from './lib/demo'

type PageMode = 'agent' | 'terminal' | 'files' | 'audit' | 'settings' | 'desktop'

function App() {
  const { health, loading: healthLoading, connected } = useHealth()
  const { state, dispatch, extractResources } = useChatStore()
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const sessionIdRef = useRef<string | null>(null)
  const streamingForSessionRef = useRef<string | null>(null) // tracks which session the active SSE belongs to
  const [pageMode, setPageMode] = useState<PageMode>('agent')
  const [rightPanelVisible, setRightPanelVisible] = useState(true)

  // Poll resource data directly from /desktop/probe/* every 30s (Task 14)
  useResourcePolling(connected, dispatch)

  // Load permission mode from backend on connect
  useEffect(() => {
    if (connected) {
      fetch('/api/v1/permission/mode')
        .then(res => res.json())
        .then(data => {
          if (data?.data?.mode) dispatch({ type: 'SET_PERMISSION_MODE', mode: data.data.mode })
        })
        .catch(() => {})
    }
  }, [connected, dispatch])

  // Load demo data ONLY if backend confirmed unreachable (not during initial loading)
  useEffect(() => {
    if (!healthLoading && !connected && state.sessions.length === 0) {
      // Wait a bit to avoid race condition with first health check
      const timer = setTimeout(() => {
        if (!connected) {
          dispatch({ type: 'SET_SESSIONS', sessions: demoSessions })
          const demoSessionId = 'demo-1'
          dispatch({ type: 'SET_ACTIVE_SESSION', id: demoSessionId })
          sessionIdRef.current = demoSessionId
          for (const msg of demoMessages) dispatch({ type: 'ADD_MESSAGE', sessionId: demoSessionId, message: msg })
          for (const step of demoReasoning) dispatch({ type: 'ADD_REASONING_STEP', step })
          dispatch({ type: 'UPDATE_RESOURCES', data: demoResources })
        }
      }, 500)
      return () => clearTimeout(timer)
    }
  }, [healthLoading, connected, state.sessions.length, dispatch])

  const messages = getSessionMessages(state)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const { send, abort } = useSSE({
    onStart: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      if (!sessionIdRef.current) {
        sessionIdRef.current = data.session_id
        dispatch({ type: 'SET_ACTIVE_SESSION', id: data.session_id })
      }
      dispatch({ type: 'SET_STREAMING', streaming: true })
      dispatch({ type: 'CLEAR_REASONING' })
      dispatch({ type: 'ADD_MESSAGE', sessionId, message: { id: `agent-${Date.now()}`, role: 'agent', content: '', timestamp: new Date().toISOString(), toolCalls: [] } })
    },
    onSense: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      dispatch({ type: 'ADD_REASONING_STEP', step: { phase: 'sense', timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), data, status: 'done' } })
      if (data.status === 'blocked') {
        const currentMessages = getSessionMessages(state, sessionId)
        const lastAgent = currentMessages[currentMessages.length - 1]
        if (lastAgent) {
          dispatch({ type: 'SET_BLOCKED', sessionId, messageId: lastAgent.id })
          dispatch({ type: 'UPDATE_LAST_AGENT', sessionId, updater: (m) => ({ ...m, content: data.reason || '输入被安全护栏拦截' }) })
        }
      }
    },
    onModeDecision: (data) => {
      if (data.mode === 'multi') {
        dispatch({ type: 'SET_MULTI_AGENT_MODE', enabled: true })
        dispatch({ type: 'SET_MULTI_AGENT_STATUS', role: null, round: 1, status: '多Agent协作启动...' })
      }
    },
    onAnalyze: (data) => {
      const sessionId = streamingForSessionRef.current
      dispatch({ type: 'ADD_REASONING_STEP', step: { phase: 'analyze', timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), data, status: 'done' } })
      if (sessionId) {
        dispatch({ type: 'SET_THINKING', sessionId, data: { status: 'thinking', analyzeSummary: data.reply_preview } })
      }
    },
    onPlan: (data) => {
      const sessionId = streamingForSessionRef.current
      dispatch({ type: 'ADD_REASONING_STEP', step: { phase: 'plan', timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), data, status: 'done' } })
      if (sessionId) {
        dispatch({ type: 'SET_THINKING', sessionId, data: { status: 'thinking', planTools: data.tools.map((t: { name: string }) => t.name) } })
      }
    },
    onExecute: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      dispatch({ type: 'ADD_TOOL_CALL', sessionId, toolCall: { tool: data.tool, args: data.args, status: 'running' } })
      dispatch({ type: 'ADD_REASONING_STEP', step: { phase: 'execute', timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), data: { tool: data.tool }, status: 'active' } })
    },
    onExecuteDone: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      dispatch({ type: 'UPDATE_TOOL_CALL', sessionId, tool: data.tool, update: { status: (data.status === 'ok' || data.status === 'success') ? 'done' : 'error', result_preview: data.result_preview, elapsed_ms: data.elapsed_ms } })
      dispatch({ type: 'ADD_REASONING_STEP', step: { phase: 'execute', timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), data: { tool: data.tool, elapsed_ms: data.elapsed_ms }, status: 'done' } })
      extractResources(data)
    },
    onTextDelta: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      dispatch({ type: 'APPEND_DELTA', sessionId, delta: data.delta })
    },
    onReasoningDelta: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      dispatch({ type: 'APPEND_REASONING', sessionId, delta: data.delta })
    },
    onOutput: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      dispatch({ type: 'UPDATE_LAST_AGENT', sessionId, updater: (m) => ({ ...m, content: data.reply }) })
      dispatch({ type: 'SET_THINKING', sessionId, data: { status: 'done' } })
      dispatch({ type: 'ADD_REASONING_STEP', step: { phase: 'output', timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), data: { tokens_used: data.tokens_used, elapsed_ms: data.elapsed_ms }, status: 'done' } })
    },
    onError: (data) => {
      const sessionId = streamingForSessionRef.current || sessionIdRef.current
      if (!sessionId) return
      dispatch({ type: 'ADD_MESSAGE', sessionId, message: { id: `error-${Date.now()}`, role: 'agent', content: '', timestamp: new Date().toISOString(), error: data } })
    },
    onDone: () => { dispatch({ type: 'SET_STREAMING', streaming: false }) },
    onAgentRole: (data) => {
      const role = data.role as 'planner' | 'executor' | 'verifier' | 'coordinator'
      const round = data.iteration ?? (state.multiAgentRound || 1)
      let content = data.message || data.sub_task || ''
      if (data.action === 'dispatch') content = `分配 ${data.total ?? '?'} 个子任务给 Executor 并行执行`
      if (data.action === 'waiting') content = data.message || '等待中...'
      if (data.action === 'collected') content = data.message || '全部收齐'

      dispatch({ type: 'SET_MULTI_AGENT_STATUS', role, round, status: `协作中 · 第${round}轮 · ${role}` })
      if (content) {
        dispatch({ type: 'ADD_AGENT_MESSAGE', message: { id: `ma-${Date.now()}-${Math.random()}`, role, content, timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), round } })
      }
    },
    onVerifierResult: (data) => {
      const content = data.verified
        ? `✓ 验证通过 (confidence: ${data.confidence}) — ${data.reason}`
        : `✗ 验证未通过 — ${data.reason}${data.missing_info?.length ? '\n缺失: ' + data.missing_info.join(', ') : ''}`
      dispatch({ type: 'ADD_AGENT_MESSAGE', message: { id: `ma-vr-${Date.now()}`, role: 'verifier', content, timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false }), round: data.iteration } })
    },
    onPermissionRequest: (data) => {
      const sessionId = streamingForSessionRef.current
      if (!sessionId) return
      dispatch({
        type: 'SET_PERMISSION_REQUEST',
        sessionId,
        data: { ...data, status: 'pending' },
      })
    },
    onConnectionError: (error) => {
      const sessionId = streamingForSessionRef.current || sessionIdRef.current
      dispatch({ type: 'SET_STREAMING', streaming: false })
      if (sessionId) {
        dispatch({ type: 'ADD_MESSAGE', sessionId, message: { id: `conn-err-${Date.now()}`, role: 'agent', content: '', timestamp: new Date().toISOString(), error: { error_code: 'NET_001', message: `连接断开: ${error.message}`, recoverable: true } } })
      }
    },
  })

  const handleSend = useCallback((message: string) => {
    // Task 3: Handle clear command — create a new session
    if (message === '清空对话' || message.startsWith('/clear')) {
      const newId = `local-${Date.now()}`
      dispatch({ type: 'CREATE_SESSION', session: {
        id: newId,
        title: '新对话',
        last_message: '',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      } })
      sessionIdRef.current = newId
      return
    }

    const currentSessionId = sessionIdRef.current || state.activeSessionId
    // Add user message
    if (currentSessionId) {
      dispatch({ type: 'ADD_MESSAGE', sessionId: currentSessionId, message: { id: `user-${Date.now()}`, role: 'user', content: message, timestamp: new Date().toISOString() } })
    }

    if (!currentSessionId) {
      // No active session — create one
      const newId = `local-${Date.now()}`
      dispatch({ type: 'CREATE_SESSION', session: { id: newId, title: message.slice(0, 20), last_message: message.slice(0, 30), created_at: new Date().toISOString(), updated_at: new Date().toISOString() } })
      sessionIdRef.current = newId
      // Add user message to the newly created session
      dispatch({ type: 'ADD_MESSAGE', sessionId: newId, message: { id: `user-${Date.now()}`, role: 'user', content: message, timestamp: new Date().toISOString() } })
    } else {
      // Update existing session title (use first message as title if still "新对话")
      dispatch({ type: 'SET_SESSIONS', sessions: state.sessions.map(s =>
        s.id === currentSessionId
          ? { ...s, title: s.title === '新对话' ? message.slice(0, 20) : s.title, last_message: message.slice(0, 30), updated_at: new Date().toISOString() }
          : s
      ) })
    }

    const targetSessionId = sessionIdRef.current || state.activeSessionId
    // Only send to backend if connected
    if (connected && targetSessionId) {
      streamingForSessionRef.current = targetSessionId
      send(message, targetSessionId)
    } else if (targetSessionId) {
      // Offline: show a simulated error
      dispatch({ type: 'ADD_MESSAGE', sessionId: targetSessionId, message: { id: `offline-${Date.now()}`, role: 'agent', content: '', timestamp: new Date().toISOString(), error: { error_code: 'NET_001', message: '后端未连接，无法发送。请启动 ops-agent server (localhost:8080)', recoverable: true } } })
    }
  }, [send, dispatch, state.sessions, state.activeSessionId, connected])

  const handleNewSession = useCallback(() => {
    // DON'T abort — let the old session's SSE continue in background
    // The streaming response will still write to messagesBySession via onTextDelta/onOutput
    const newId = `local-${Date.now()}`
    dispatch({ type: 'CREATE_SESSION', session: {
      id: newId,
      title: '新对话',
      last_message: '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    } })
    dispatch({ type: 'SET_STREAMING', streaming: false })
    sessionIdRef.current = newId
  }, [dispatch])

  const handleSelectSession = useCallback((id: string) => {
    if (id === sessionIdRef.current) return
    // Don't abort — let background SSE connections continue writing to their sessions
    dispatch({ type: 'SET_STREAMING', streaming: false })
    dispatch({ type: 'SET_ACTIVE_SESSION', id })
    dispatch({ type: 'CLEAR_MULTI_AGENT' })
    sessionIdRef.current = id
  }, [dispatch])

  const handleRetry = useCallback(() => {
    const lastUserMsg = [...messages].reverse().find(m => m.role === 'user')
    if (lastUserMsg) {
      handleSend(lastUserMsg.content)
    }
  }, [messages, handleSend])

  const handleNavigate = useCallback((page: string) => {
    setPageMode(page as PageMode)
  }, [])

  const handleModeChange = useCallback((mode: 'chat' | 'desktop') => {
    setPageMode(mode === 'desktop' ? 'desktop' : 'agent')
  }, [])

  const handlePermissionRespond = useCallback((_requestId: string, action: 'allow' | 'allow_session' | 'deny') => {
    const status = action === 'deny' ? 'denied' as const : 'allowed' as const
    dispatch({ type: 'UPDATE_PERMISSION_STATUS', status })
  }, [dispatch])

  const handlePermissionModeChange = useCallback((mode: 'default' | 'auto_approve') => {
    fetch('/api/v1/permission/mode', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode }),
    }).then(res => {
      if (res.ok) dispatch({ type: 'SET_PERMISSION_MODE', mode })
    })
  }, [dispatch])

  // Render main content area based on pageMode
  const renderMainContent = () => {
    switch (pageMode) {
      case 'agent':
        return (
          <>
            <SessionList sessions={state.sessions} activeId={state.activeSessionId} onSelect={handleSelectSession} onNew={handleNewSession} />
            <main style={{ flex: 1, display: 'flex', flexDirection: 'column', background: 'var(--ops-bg-canvas)', overflow: 'hidden', minWidth: 0 }}>
              <ResourceStrip resources={state.resources} />
              <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column' }}>
                <div style={{ padding: '16px 48px', display: 'flex', flexDirection: 'column', gap: 16, maxWidth: 720, margin: '0 auto', width: '100%', flex: messages.length === 0 ? 1 : undefined }}>
                  {messages.length === 0 && (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', flex: 1, minHeight: 300, gap: 16 }}>
                      <span style={{ fontFamily: 'var(--ops-font-mono)', fontSize: 14, fontWeight: 500, color: 'var(--ops-fg-muted)' }}>OPS·AGENT</span>
                      <span style={{ fontFamily: 'var(--ops-font-ui)', fontSize: 12, color: 'var(--ops-fg-muted)' }}>输入运维诉求开始对话</span>
                      <SuggestedPrompts onSelect={handleSend} />
                    </div>
                  )}
                  {messages.map((msg, i) => (
                    <ChatMessageComponent
                      key={msg.id}
                      message={msg}
                      isStreaming={state.isStreaming}
                      isLastAgent={msg.role === 'agent' && i === messages.length - 1}
                      onRetry={handleRetry}
                      onStop={abort}
                    />
                  ))}
                  {/* Multi-Agent group chat view */}
                  {state.multiAgentMode && state.multiAgentMessages.length > 0 && (
                    <div style={{ border: '1px solid var(--ops-border-subtle)', borderRadius: 4, overflow: 'hidden', margin: '8px 0' }}>
                      <MultiAgentChat
                        messages={state.multiAgentMessages}
                        currentRound={state.multiAgentRound}
                        activeRole={state.multiAgentActiveRole as any}
                        status={state.multiAgentStatus}
                      />
                    </div>
                  )}
                  <div ref={messagesEndRef} />
                </div>
              </div>
              <ChatInput onSend={handleSend} disabled={state.isStreaming} pendingPermission={state.pendingPermission} onPermissionRespond={handlePermissionRespond} permissionMode={state.permissionMode} onPermissionModeChange={handlePermissionModeChange} />
            </main>
          </>
        )
      case 'terminal':
        return <TerminalDrawer isFullscreen={true} onToggleFullscreen={() => setPageMode('agent')} />
      case 'files':
        return <PlaceholderPage icon="folder" title="文件管理" description="文件浏览器功能开发中 (Task 14)" />
      case 'audit':
        return <PlaceholderPage icon="description" title="审计日志" description="审计日志功能开发中 (Task 10)" />
      case 'settings':
        return <SettingsPage />
      case 'desktop':
        return (
          <DesktopMode
            resources={state.resources}
            health={health}
            connected={connected}
            onSwitchToChat={() => setPageMode('agent')}
          />
        )
      default:
        return null
    }
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <AppHeader health={health} mode={pageMode === 'desktop' ? 'desktop' : 'chat'} onModeChange={handleModeChange} connected={connected} />

      {/* Main body */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* SideNav — 44px */}
        <SideNav active={pageMode} onNavigate={handleNavigate} />

        {/* Main content */}
        {renderMainContent()}

        {/* Right Panel — visible in agent mode, shared */}
        {rightPanelVisible && pageMode !== 'desktop' && (
          <RightPanel
            reasoning={state.reasoning}
            resources={state.resources}
            health={health}
            healthLoading={healthLoading}
            isStreaming={state.isStreaming}
            showCopilot={pageMode === 'terminal'}
            onSendMessage={handleSend}
            messages={messages}
            onClose={() => setRightPanelVisible(false)}
          />
        )}
        {!rightPanelVisible && pageMode !== 'desktop' && (
          <button
            onClick={() => setRightPanelVisible(true)}
            title="展开侧栏"
            style={{
              position: 'absolute',
              right: 0,
              top: '50%',
              transform: 'translateY(-50%)',
              width: 24,
              height: 80,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'var(--ops-bg-surface)',
              border: '1px solid var(--ops-border-subtle)',
              borderRight: 'none',
              borderRadius: '4px 0 0 4px',
              cursor: 'pointer',
              zIndex: 10,
            }}
          >
            <span className="material-symbols-outlined" style={{ fontSize: 16, color: 'var(--ops-fg-muted)' }}>
              chevron_left
            </span>
          </button>
        )}
      </div>

      {/* Status Bar — 22px */}
      <StatusBar health={health} connected={connected} />
    </div>
  )
}

export default App
