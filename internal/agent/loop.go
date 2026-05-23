package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"ops-agent/internal/agent/prompt"
	"ops-agent/internal/llm"
	"ops-agent/internal/permission"
	"ops-agent/internal/safety"
	"ops-agent/internal/tools"
)

const MaxToolRounds = 10

// Event is an SSE event sent to the client.
type Event struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// AgentConfig holds agent runtime configuration.
type AgentConfig struct {
	Model string
}

// Agent orchestrates the tool-use loop.
type Agent struct {
	llm        LLMClient
	registry   tools.ToolRegistry
	multi      *MultiAgent
	judge      *ComplexityJudge
	cfg        AgentConfig
	audit      AuditWriter
	permission *permission.Service
}

// AuditWriter interface for dependency injection (nil-safe).
type AuditWriter interface {
	Write(entry AuditEntry) error
}

// AuditEntry mirrors audit.Entry for decoupling.
type AuditEntry struct {
	TraceID     string
	SessionID   string
	RoundNumber int
	Stage       string
	Role        string
	Content     any
	TriggeredBy string
	Status      string
	DurationMs  int
}

// NewAgent creates a new Agent instance.
func NewAgent(llm LLMClient, reg tools.ToolRegistry, cfg AgentConfig, permSvc *permission.Service) *Agent {
	return &Agent{
		llm:        llm,
		registry:   reg,
		multi:      NewMultiAgent(llm, reg, cfg),
		judge:      NewComplexityJudge(nil),
		cfg:        cfg,
		permission: permSvc,
	}
}

// SetAuditWriter wires an audit writer into the agent (optional, nil-safe).
func (a *Agent) SetAuditWriter(w AuditWriter) {
	a.audit = w
}

// MultiAgent returns the multi-agent executor for tool registration.
func (a *Agent) MultiAgent() *MultiAgent {
	return a.multi
}

// writeAudit is a nil-safe helper to write audit entries.
func (a *Agent) writeAudit(entry AuditEntry) {
	if a.audit != nil {
		a.audit.Write(entry)
	}
}

// RunStream executes the agent loop and returns an event channel.
func (a *Agent) RunStream(ctx context.Context, sessionID, userMessage string, history []Message) <-chan Event {
	return a.RunStreamWithMode(ctx, sessionID, userMessage, ModeAuto, history)
}

// RunStreamWithMode executes the agent loop with a specific mode override.
func (a *Agent) RunStreamWithMode(ctx context.Context, sessionID, userMessage string, forceMode Mode, history []Message) <-chan Event {
	out := make(chan Event, 64)
	go func() {
		defer close(out)
		traceID := fmt.Sprintf("trc_%d", time.Now().UnixNano())

		out <- Event{Type: "start", Data: map[string]any{
			"trace_id": traceID, "session_id": sessionID, "mode": "single",
		}}

		// Sense: injection scan (fast, ~1ms)
		scan := safety.ScanInjection(userMessage)
		if scan.IsBlocked {
			out <- Event{Type: "sense", Data: map[string]any{"status": "blocked", "reason": scan.ErrorCode}}
			out <- Event{Type: "error", Data: map[string]any{
				"error_code": scan.ErrorCode, "message": "检测到提示词注入风险", "recoverable": false,
			}}
			a.writeAudit(AuditEntry{TraceID: traceID, SessionID: sessionID, Stage: "SENSE", Status: "blocked", Content: map[string]any{"reason": scan.ErrorCode}})
			out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "error"}}
			return
		}
		out <- Event{Type: "sense", Data: map[string]any{"status": "ok"}}
		a.writeAudit(AuditEntry{TraceID: traceID, SessionID: sessionID, Stage: "SENSE", Status: "ok", Content: map[string]any{"input_length": len(userMessage)}})

		// Mode decision: instant, no LLM call (OpenCode pattern: model-as-router via tools)
		// forceMode=multi is the only case we still honor the old multi path directly
		if forceMode == ModeMulti {
			out <- Event{Type: "mode_decision", Data: map[string]any{"mode": "multi", "reason": "强制多Agent模式"}}
			a.multi.Run(ctx, sessionID, traceID, userMessage, out)
			return
		}

		out <- Event{Type: "mode_decision", Data: map[string]any{"mode": "single", "reason": "default"}}

		// Single streaming agent loop — same architecture as OpenCode's processGeneration()
		a.runSingle(ctx, traceID, sessionID, userMessage, history, out)
	}()
	return out
}

func (a *Agent) runSingle(ctx context.Context, traceID, sessionID, userMessage string, history []Message, out chan<- Event) {
	// Get model-aware token budget for compaction
	modelInfo := llm.GetModelInfo(a.cfg.Model)
	tokenBudget := modelInfo.ContextBudget()

	// Build messages with history context
	messages := []Message{
		{Role: "system", Content: buildSystemPrompt(a.registry)},
	}
	if len(history) > 0 {
		messages = append(messages, history...)
	}
	messages = append(messages, Message{Role: "user", Content: safety.WrapUserInput(userMessage)})

	// Tool-use loop with STREAMING
	for round := 1; round <= MaxToolRounds; round++ {
		// Check for cancellation before each round
		select {
		case <-ctx.Done():
			out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "cancelled"}}
			return
		default:
		}

		messages = compactMessages(messages, tokenBudget)

		// Try streaming first, fallback to non-streaming
		stream, streamErr := a.llm.ChatStream(ctx, ChatRequest{
			Messages:       messages,
			Tools:          a.buildToolDefs(),
			EnableThinking: true,
		})

		var replyContent string
		var toolCalls []ToolCall
		var finishReason string
		var usage *Usage

		if streamErr != nil {
			// Streaming not supported or failed — fallback to Chat()
			resp, err := a.llm.Chat(ctx, ChatRequest{Messages: messages, Tools: a.buildToolDefs(), EnableThinking: true})
			if err != nil {
				errCode := "LLM_SERVICE_001"
				if asLLM, ok := err.(*LLMError); ok {
					errCode = string(asLLM.Code)
				}
				out <- Event{Type: "error", Data: map[string]any{"error_code": errCode, "message": err.Error(), "recoverable": false}}
				out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "error"}}
				return
			}
			if len(resp.Choices) > 0 {
				replyContent = resp.Choices[0].Message.Content
				toolCalls = resp.Choices[0].Message.ToolCalls
				finishReason = resp.Choices[0].FinishReason
			}
			usage = &resp.Usage
		} else {
			// True streaming — emit text_delta events for each token
			for chunk := range stream {
				if chunk.Error != nil {
					out <- Event{Type: "error", Data: map[string]any{"error_code": "LLM_STREAM_001", "message": chunk.Error.Error(), "recoverable": false}}
					out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "error"}}
					return
				}
				if chunk.ReasoningDelta != "" {
					out <- Event{Type: "reasoning_delta", Data: map[string]any{"delta": chunk.ReasoningDelta, "round": round}}
				}
				if chunk.Delta != "" {
					replyContent += chunk.Delta
					// Push each token delta to frontend in real-time
					out <- Event{Type: "text_delta", Data: map[string]any{"delta": chunk.Delta, "round": round}}
				}
				if chunk.ToolCalls != nil {
					toolCalls = chunk.ToolCalls
				}
				if chunk.FinishReason != "" {
					finishReason = chunk.FinishReason
				}
				if chunk.Usage != nil {
					usage = chunk.Usage
				}
			}
		}

		hasToolCalls := len(toolCalls) > 0

		out <- Event{Type: "analyze", Data: map[string]any{
			"round": round, "has_tool_calls": hasToolCalls,
			"finish_reason": finishReason, "reply_preview": truncateStr(replyContent, 100),
		}}

		if !hasToolCalls {
			// Final reply (text_delta already streamed, now send complete output for backward compat)
			tokensUsed := map[string]any{"prompt": 0, "completion": 0}
			if usage != nil {
				tokensUsed = map[string]any{"prompt": usage.PromptTokens, "completion": usage.CompletionTokens}
			}
			out <- Event{Type: "output", Data: map[string]any{
				"reply": replyContent, "tokens_used": tokensUsed, "elapsed_ms": 0, "mode": "single",
			}}
			a.writeAudit(AuditEntry{TraceID: traceID, SessionID: sessionID, Stage: "OUTPUT", Status: "ok", Content: map[string]any{"reply_length": len(replyContent)}})
			out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "ok"}}
			return
		}

		// Execute tool calls — parallel for readonly, sequential for write ops (OpenCode pattern)
		out <- Event{Type: "plan", Data: map[string]any{"round": round, "tools": summarizeToolCalls(toolCalls)}}
		messages = append(messages, Message{Role: "assistant", ToolCalls: toolCalls})

		// Separate readonly and write tools
		type toolExecResult struct {
			index      int
			tc         ToolCall
			result     tools.Result
			content    string
			secResult  string
		}

		// Classify and execute
		readonlyTools := []int{}
		writeTools := []int{}
		for i, tc := range toolCalls {
			if t, ok := a.registry.Get(tc.Function.Name); ok && t.Type() == tools.ToolWrite {
				writeTools = append(writeTools, i)
			} else {
				readonlyTools = append(readonlyTools, i)
			}
		}

		// Execute readonly tools in parallel
		readResults := make([]toolExecResult, len(toolCalls))
		if len(readonlyTools) > 0 {
			var wg sync.WaitGroup
			for _, idx := range readonlyTools {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					tc := toolCalls[i]
					args := parseToolArgs(tc.Function.Arguments)
					start := time.Now()

					out <- Event{Type: "execute", Data: map[string]any{"tool": tc.Function.Name, "args": args, "security_check": "PASSED"}}

					result := a.registry.Dispatch(ctx, tc.Function.Name, args)
					elapsed := time.Since(start).Milliseconds()

					out <- Event{Type: "execute_done", Data: map[string]any{
						"tool": tc.Function.Name, "status": statusFromResult(result),
						"result_preview": truncateStr(result.Summary, 200), "elapsed_ms": elapsed,
					}}

					content := result.Summary
					if result.Error != "" {
						content = "Error: " + result.Error
					}
					if len(content) > 4096 {
						content = content[:4096] + "\n[Truncated]"
					}
					readResults[i] = toolExecResult{index: i, tc: tc, result: result, content: content, secResult: "PASSED"}
				}(idx)
			}
			wg.Wait()
		}

		// Execute write tools sequentially (need security check + ordering + permission)
		for _, idx := range writeTools {
			tc := toolCalls[idx]
			args := parseToolArgs(tc.Function.Arguments)
			start := time.Now()

			// Security check: only for bash tool (write tools have their own internal validation)
			securityResult := "PASSED"
			if tc.Function.Name == "bash" {
				cmd := tc.Function.Name + " " + tc.Function.Arguments
				vr := safety.ValidateCommand(cmd)
				if vr.Status == safety.StatusBlocked {
					securityResult = string(vr.Reason)
				}
			}

			out <- Event{Type: "execute", Data: map[string]any{"tool": tc.Function.Name, "args": args, "security_check": securityResult}}

			var result tools.Result
			if securityResult != "PASSED" {
				result = tools.Result{Error: "BLOCKED: " + securityResult}
			} else {
				// Permission check: block until user confirms
				if a.permission != nil {
					permReq := permission.Request{
						SessionID:   sessionID,
						ToolName:    tc.Function.Name,
						Command:     tc.Function.Arguments,
						RiskLevel:   classifyRisk(tc.Function.Name, args),
						Description: tc.Function.Name + ": " + truncateStr(tc.Function.Arguments, 80),
					}
					permitted, _ := a.permission.RequestPermission(ctx, permReq, func(req permission.Request) {
						out <- Event{Type: "permission_request", Data: map[string]any{
							"request_id":  req.ID,
							"tool":        req.ToolName,
							"command":     req.Command,
							"risk_level":  req.RiskLevel,
							"description": req.Description,
							"expires_at":  req.ExpiresAt,
						}}
					})
					if !permitted {
						result = tools.Result{Error: "用户拒绝执行此操作"}
						elapsed := time.Since(start).Milliseconds()
						out <- Event{Type: "execute_done", Data: map[string]any{
							"tool": tc.Function.Name, "status": "denied",
							"result_preview": "用户拒绝执行此操作", "elapsed_ms": elapsed,
						}}
						content := "Permission denied by user"
						readResults[idx] = toolExecResult{index: idx, tc: tc, result: result, content: content, secResult: securityResult}
						continue
					}
				}
				result = a.registry.Dispatch(ctx, tc.Function.Name, args)
			}
			elapsed := time.Since(start).Milliseconds()

			out <- Event{Type: "execute_done", Data: map[string]any{
				"tool": tc.Function.Name, "status": statusFromResult(result),
				"result_preview": truncateStr(result.Summary, 200), "elapsed_ms": elapsed,
			}}

			content := result.Summary
			if result.Error != "" {
				content = "Error: " + result.Error
			}
			if len(content) > 4096 {
				content = content[:4096] + "\n[Truncated]"
			}
			readResults[idx] = toolExecResult{index: idx, tc: tc, result: result, content: content, secResult: securityResult}
		}

		// Append all tool results in order
		for _, tc := range toolCalls {
			for _, res := range readResults {
				if res.tc.ID == tc.ID {
					messages = append(messages, Message{Role: "tool", Content: res.content, ToolCallID: tc.ID})
					break
				}
			}
		}
	}

	// Max rounds reached
		out <- Event{Type: "output", Data: map[string]any{
			"reply": "操作步骤过多，已达到最大轮次限制（10轮），已中止。", "mode": "single",
		}}
		out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "ok"}}
}

func (a *Agent) buildToolDefs() []ToolDef {
	defs := a.registry.Definitions()
	result := make([]ToolDef, len(defs))
	for i, d := range defs {
		result[i] = ToolDef{
			Type: d.Type,
			Function: FunctionDef{
				Name:        d.Function.Name,
				Description: d.Function.Description,
				Parameters:  d.Function.Parameters,
			},
		}
	}
	return result
}

func buildSystemPrompt(reg tools.ToolRegistry) string {
	workDir, _ := os.Getwd()
	return prompt.GetPrompt(prompt.RoleCoder, workDir)
}

func parseToolArgs(argsJSON string) map[string]any {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return map[string]any{}
	}
	return args
}

func summarizeToolCalls(calls []ToolCall) []map[string]any {
	result := make([]map[string]any, len(calls))
	for i, tc := range calls {
		result[i] = map[string]any{"name": tc.Function.Name, "args": tc.Function.Arguments}
	}
	return result
}

func statusFromResult(r tools.Result) string {
	if r.Error != "" {
		return "error"
	}
	return "ok"
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// classifyRisk assigns a risk level based on tool name and arguments.
func classifyRisk(toolName string, args map[string]any) string {
	switch toolName {
	case "kill_process":
		return "high"
	case "service_control":
		if action, _ := args["action"].(string); action == "stop" {
			return "high"
		}
		return "medium"
	case "delete_file":
		return "high"
	case "truncate_log_file", "vacuum_journal", "logrotate_now":
		return "medium"
	case "bash":
		return "medium"
	default:
		return "low"
	}
}
