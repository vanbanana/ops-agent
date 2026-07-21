package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ops-agent/internal/agent/prompt"
	"ops-agent/internal/llm"
	"ops-agent/internal/permission"
	"ops-agent/internal/safety"
	"ops-agent/internal/tools"
)

const MaxToolRounds = 1000

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
	cb         *CircuitBreaker // Circuit breaker for LLM API calls
	planState  *PlanState      // Plan mode state machine
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
		cb:         NewCircuitBreaker(3, 30*time.Second),
		planState:  NewPlanState(),
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

// PlanState returns the plan mode state machine for API handlers.
func (a *Agent) PlanState() *PlanState {
	return a.planState
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
			// Build details from matched rules
			ruleDetails := make([]map[string]any, 0, len(scan.Matches))
			for _, m := range scan.Matches {
				ruleDetails = append(ruleDetails, map[string]any{
					"rule_id": m.Rule.ID,
					"reason":  m.Rule.Reason,
					"snippet": m.Snippet,
				})
			}
			out <- Event{Type: "sense", Data: map[string]any{"status": "blocked", "reason": scan.ErrorCode, "rules": ruleDetails}}
			out <- Event{Type: "error", Data: map[string]any{
				"error_code": scan.ErrorCode, "message": "检测到提示词注入风险", "recoverable": false,
				"rules": ruleDetails,
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
	isPlanMode := a.planState.IsPlanning()
	messages := []Message{
		{Role: "system", Content: buildSystemPrompt(a.registry, isPlanMode)},
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

		// Circuit breaker: fail fast if LLM API is consistently failing
		if !a.cb.Allow() {
			out <- Event{Type: "circuit_open", Data: map[string]any{
				"retry_after_sec": 30,
				"message":         "LLM API 连续失败，熔断器已开启，30秒后自动恢复",
			}}
			out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "circuit_open"}}
			return
		}

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
				a.cb.RecordFailure() // Circuit breaker: record failure
				errCode := "LLM_SERVICE_001"
				if asLLM, ok := err.(*LLMError); ok {
					errCode = string(asLLM.Code)
				}
				out <- Event{Type: "error", Data: map[string]any{"error_code": errCode, "message": err.Error(), "recoverable": false}}
				out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "error"}}
				return
			}
			a.cb.RecordSuccess() // Circuit breaker: record success
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
					a.cb.RecordFailure() // Circuit breaker: record stream failure
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

		// Circuit breaker: if we got here, stream completed successfully
		a.cb.RecordSuccess()

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

		// Handle virtual exit_plan_mode tool (plan mode completion signal)
		if a.planState.IsPlanning() {
			for _, tc := range toolCalls {
				if tc.Function.Name == "exit_plan_mode" {
					// LLM has finished planning — extract plan text and submit
					args := parseToolArgs(tc.Function.Arguments)
					planText, _ := args["plan_text"].(string)
					if planText == "" {
						planText = replyContent // Fallback: use the text reply as plan
					}
					a.planState.SubmitPlan(planText)

					// Send plan_ready event to frontend
					out <- Event{Type: "plan_ready", Data: map[string]any{
						"plan_id":   a.planState.PlanID(),
						"plan_text": planText,
						"steps":     extractPlanSteps(planText),
					}}

					// Respond to the tool call
					messages = append(messages, Message{Role: "assistant", ToolCalls: toolCalls})
					messages = append(messages, Message{Role: "tool", Content: "计划已提交，等待用户审批。", ToolCallID: tc.ID})

					out <- Event{Type: "output", Data: map[string]any{
						"reply": planText, "mode": "plan",
					}}
					out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "plan_submitted"}}
					return
				}
			}
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

		// Plan mode: reject all write tools silently, only allow readonly
		if a.planState.IsPlanning() && len(writeTools) > 0 {
			// In plan mode, write operations are queued into the plan, not executed
			readResults := make([]toolExecResult, len(toolCalls))
			for _, idx := range writeTools {
				tc := toolCalls[idx]
				readResults[idx] = toolExecResult{
					index:   idx,
					tc:      tc,
					result:  tools.Result{Summary: "[计划模式] 此写操作已被记录但未执行。请继续收集信息，最终以文本形式输出完整操作计划。"},
					content: "[计划模式] 此写操作已被记录但未执行。请继续收集信息，最终以文本形式输出完整操作计划。",
				}
			}
			// Still execute readonly tools normally
			if len(readonlyTools) > 0 {
				var wg sync.WaitGroup
				for _, idx := range readonlyTools {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						tc := toolCalls[i]
						args := parseToolArgs(tc.Function.Arguments)
						result := a.registry.Dispatch(ctx, tc.Function.Name, args)
						content := result.Summary
						if result.Error != "" {
							content = "Error: " + result.Error
						}
						readResults[i] = toolExecResult{index: i, tc: tc, result: result, content: content}
					}(idx)
				}
				wg.Wait()
			}
			// Append all results
			for _, tc := range toolCalls {
				for _, res := range readResults {
					if res.tc.ID == tc.ID {
						messages = append(messages, Message{Role: "tool", Content: res.content, ToolCallID: tc.ID})
						break
					}
				}
			}
			continue // Next round
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
					// Large output persistence: save to disk, keep preview in context
					if len(content) > OutputPersistThreshold {
						path, preview, persistErr := PersistOutput(sessionID, traceID, tc.Function.Name, content)
						if persistErr == nil {
							out <- Event{Type: "output_persisted", Data: map[string]any{
								"path": path, "original_size": len(content), "tool": tc.Function.Name,
							}}
							content = preview
						} else {
							// Fallback: simple truncation
							content = content[:4096] + "\n[Truncated]"
						}
					} else if len(content) > 4096 {
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

			// Security check: validate the actual command via AST-based pipeline
			// In auto_approve mode, skip security validation (user takes full responsibility)
			securityResult := "PASSED"
			securityDetail := ""
			destructiveWarning := ""
			isAutoApprove := a.permission != nil && a.permission.GetMode() == permission.ModeAutoApprove
			if tc.Function.Name == "bash" && !isAutoApprove {
				cmd, _ := args["command"].(string)
				if cmd != "" {
					vr := safety.ValidateCommand(cmd)
					if vr.Status == safety.StatusBlocked {
						securityResult = string(vr.Reason)
						securityDetail = vr.Detail
					}
					// Destructive warning (non-blocking info for confirmation UI)
					destructiveWarning = safety.GetDestructiveWarning(cmd)
					if vr.Warning != "" && destructiveWarning == "" {
						destructiveWarning = vr.Warning
					}
				}
			}

			out <- Event{Type: "execute", Data: map[string]any{"tool": tc.Function.Name, "args": args, "security_check": securityResult, "warning": destructiveWarning}}

			var result tools.Result
			if securityResult != "PASSED" {
				result = tools.Result{Error: "BLOCKED: " + securityResult + " — " + securityDetail}
				a.writeAudit(AuditEntry{TraceID: traceID, SessionID: sessionID, Stage: "EXECUTE", Status: "blocked", Content: map[string]any{"tool": tc.Function.Name, "reason": securityResult, "detail": securityDetail}})
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
						a.writeAudit(AuditEntry{TraceID: traceID, SessionID: sessionID, Stage: "EXECUTE", Status: "blocked", Content: map[string]any{"tool": tc.Function.Name, "reason": "user_denied", "args": tc.Function.Arguments}})
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
			// Large output persistence for write tools too
			if len(content) > OutputPersistThreshold {
				path, preview, persistErr := PersistOutput(sessionID, traceID, tc.Function.Name, content)
				if persistErr == nil {
					out <- Event{Type: "output_persisted", Data: map[string]any{
						"path": path, "original_size": len(content), "tool": tc.Function.Name,
					}}
					content = preview
				} else {
					content = content[:4096] + "\n[Truncated]"
				}
			} else if len(content) > 4096 {
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
			"reply": "操作步骤过多，已达到最大轮次限制（1000轮），已中止。", "mode": "single",
		}}
		out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "ok"}}
}

func (a *Agent) buildToolDefs() []ToolDef {
	defs := a.registry.Definitions()
	isPlan := a.planState.IsPlanning()

	var result []ToolDef
	for _, d := range defs {
		// In plan mode: hide all write tools from the LLM so it can't call them
		if isPlan && d.Function.Name != "" {
			// Check if this tool is a write tool by looking it up in registry
			if t, ok := a.registry.Get(d.Function.Name); ok && t.Type() == tools.ToolWrite {
				continue // Skip write tools in plan mode
			}
		}
		result = append(result, ToolDef{
			Type: d.Type,
			Function: FunctionDef{
				Name:        d.Function.Name,
				Description: d.Function.Description,
				Parameters:  d.Function.Parameters,
			},
		})
	}

	// In plan mode: add a virtual "exit_plan_mode" tool for the LLM to signal completion
	if isPlan {
		result = append(result, ToolDef{
			Type: "function",
			Function: FunctionDef{
				Name:        "exit_plan_mode",
				Description: "当你完成信息收集并准备好操作计划后，调用此工具提交计划。计划内容通过 plan_text 参数传入，用户将审阅后决定是否批准执行。",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"plan_text": map[string]any{
							"type":        "string",
							"description": "完整的操作计划文本，包含编号步骤、具体命令和风险评估",
						},
					},
					"required": []string{"plan_text"},
				},
			},
		})
	}

	return result
}

func buildSystemPrompt(reg tools.ToolRegistry, planMode bool) string {
	workDir, _ := os.Getwd()
	base := prompt.GetPrompt(prompt.RoleCoder, workDir)
	if planMode {
		base += `

<PLAN_MODE>
你当前处于【计划模式】。在此模式下：

1. **禁止执行任何写操作**（重启服务、删文件、kill 进程等），所有写工具调用都会被拒绝
2. **只能使用只读探测工具**收集信息（probe_disk、probe_process、bash 只读命令如 df/ps/cat 等）
3. **目标是输出一份完整的操作计划**，包含：
   - 问题诊断结论
   - 建议执行的步骤列表（按顺序，每步写清做什么、用什么命令）
   - 每步的风险评估
   - 预期效果
4. 当你完成信息收集并准备好计划后，**直接以文本形式输出计划**，不要调用写工具
5. 用户会审阅你的计划，批准后系统会自动切换到执行模式

**格式要求：** 用编号列表输出计划步骤，每步包含：操作描述 + 具体命令 + 风险等级(低/中/高)
</PLAN_MODE>`
	}
	return base
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
	if r.Error != "" && r.Summary == "" {
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

// extractPlanSteps extracts numbered steps from a plan text.
// Looks for lines starting with numbers (1. 2. 3. etc.) or bullet points.
func extractPlanSteps(planText string) []string {
	var steps []string
	for _, line := range strings.Split(planText, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		// Match: "1. xxx", "1) xxx", "- xxx", "* xxx", "步骤1: xxx"
		if len(trimmed) > 2 && ((trimmed[0] >= '0' && trimmed[0] <= '9') ||
			trimmed[0] == '-' || trimmed[0] == '*' ||
			strings.HasPrefix(trimmed, "步骤")) {
			steps = append(steps, trimmed)
		}
	}
	if len(steps) == 0 {
		// Fallback: split by sentences
		steps = []string{truncateStr(planText, 200)}
	}
	return steps
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
